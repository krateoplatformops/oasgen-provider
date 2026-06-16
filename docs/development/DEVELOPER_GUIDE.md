# Developer Guide

This guide is for people **working on** the OASGen Provider codebase (not just using it). It explains the
architecture, the OAS → CRD generation pipeline, the reconciliation model, how the code is organized, and how to
build, run, test, and extend the provider locally.

If you are a *user* (writing `RestDefinition`s, not Go code), start with the
[README](../../README.md) and the [Usage Guide](../USAGE_GUIDE.md) instead.

## Table of contents

- [Mental model: what this provider actually does](#mental-model-what-this-provider-actually-does)
- [Repository layout](#repository-layout)
- [The reconciliation lifecycle](#the-reconciliation-lifecycle)
- [The OAS → CRD generation pipeline](#the-oas--crd-generation-pipeline)
- [The `oas2jsonschema` package in depth](#the-oas2jsonschema-package-in-depth)
- [RDC deployment & the drift digest](#rdc-deployment--the-drift-digest)
- [How OASGen and the rest-dynamic-controller fit together](#how-oasgen-and-the-rest-dynamic-controller-fit-together)
- [Local development workflow](#local-development-workflow)
- [Code generation](#code-generation)
- [Testing](#testing)
- [How to extend the provider](#how-to-extend-the-provider)
- [Conventions & gotchas](#conventions--gotchas)
- [Glossary](#glossary)

---

## Mental model: what this provider actually does

OASGen Provider is a **meta-controller**: it does not talk to any external API itself. Given a `RestDefinition`
custom resource that points at an OpenAPI 3.0/3.1 document, it does two things:

1. **Generates CRDs.** It parses the OAS, builds JSON schemas for the resource's `spec` and `status` (and an
   optional Configuration CRD for parameters/authentication), and applies the resulting CRD(s) to the cluster.
2. **Deploys a controller for those CRDs.** It deploys an instance of the generic
   **rest-dynamic-controller (RDC)** — a separate program — configured to reconcile instances of the new CRD
   against the REST API described by the OAS, plus all the RBAC/ConfigMap it needs.

> In short: *OASGen is "an operator that writes operators from OpenAPI."* The CRD instances a user later
> creates (the "RestResources") are reconciled by RDC, **not** by this provider.

```
                 ┌────────────────────────────────────────────────────────┐
   RestDefinition│  oasgen-provider (this repo)                            │
   (+ OAS doc) ──┼─▶ parse OAS ─▶ build spec/status/config schemas         │
                 │   ─▶ crdgen ─▶ apply CRD (+ Configuration CRD)          │
                 │   ─▶ deploy RDC Deployment + ConfigMap + RBAC           │
                 └───────────────────────────────┬────────────────────────┘
                                                  │ deploys
                                                  ▼
   user creates       ┌──────────────────────────────────────────┐
   a "RestResource" ─▶│  rest-dynamic-controller (separate repo)  │─▶ external REST API
   (instance of the   │  reconciles the instance via HTTP         │
    generated CRD)    └──────────────────────────────────────────┘
```

---

## Repository layout

```
main.go                          # entrypoint: flags, manager wiring, controller setup
apis/
  restdefinitions/v1alpha1/
    types.go                     # the RestDefinition API types (the source of truth for the CRD)
    zz_generated.deepcopy.go     # generated (controller-gen) — DO NOT EDIT
    groupversion_info.go         # scheme registration
  generate.go                    # //go:generate directives for controller-gen
internal/
  controllers/
    restdefinition/
      restdefinition.go          # THE controller: Observe/Create/Update/Delete + finalizers
      helper.go                  # small helpers (wildcard action expansion, etc.)
  tools/
    oas2jsonschema/              # ★ the heart: OAS document → CRD JSON schemas
    deploy/                      # renders + applies the RDC Deployment/ConfigMap/RBAC, computes digest
    deployment/                  # deployment readiness lookup + rollout restart
    objects/ (+ templates/)      # render a YAML template into a typed k8s object
    crd/                         # install/lookup/uninstall CRDs (+ multi-version "vacuum" trick)
    kube/                        # generic apply/get/delete helpers
    filegetter/                  # fetch the OAS from configmap:// or http(s)://
    plurals/                     # GVK → GVR (pluralization)
    hash/                        # FNV digest used for drift detection
    safety/                      # recursion guard (depth/nodes/timeout) for schema walking
    pathparsing/                 # dot/bracket JSON path parsing
    text/                        # string helpers for identifier generation
crds/                            # generated CRD YAML (the RestDefinition CRD) — generated, do not hand-edit
manifests/                       # dev install manifests (provider + the RDC asset ConfigMaps)
  rdc/{cm,depl,rbac}.yaml        # the RDC templates mounted into the provider at /tmp/assets
samples/                         # example RestDefinitions and CRs
scripts/                         # local dev helpers (kind, ko build, reload, test)
hack/boilerplate.go.txt          # license header injected into generated code
```

The single most important package to understand is **`internal/tools/oas2jsonschema`**. The single most
important file is **`internal/controllers/restdefinition/restdefinition.go`**.

---

## The reconciliation lifecycle

The controller is built on `provider-runtime`'s managed-reconciler model. The reconciler calls
`Observe`, and based on the returned `ExternalObservation` decides whether to call `Create`, `Update`, or
`Delete`. The "external resource" being managed is, conceptually, *"the generated CRD + the deployed RDC"*.

`Observe` (in `restdefinition.go`) walks a chain of existence/readiness checks and returns early at each step:

```
Observe(restDefinition):
  if being deleted        → manage finalizers → Delete()
  resolve hasSecuritySchemes (from status cache; else fetch the OAS once)
  CRD exists?             → no  → ResourceExists=false   (reconciler will call Create → generates CRDs)
  RDC Deployment exists?  → no  → ResourceExists=false   (reconciler will call Create → deploys RDC)
  RDC Deployment ready?   → no  → Unavailable
  Deploy(dry-run) digest == status.Digest?  → no → ResourceUpToDate=false (→ Update)
  Lookup       digest == status.Digest?      → no → ResourceUpToDate=false (→ Update)
  else → Available, UpToDate
```

- **`Create`** has two modes, branching on *whether the generated CRD already exists*:
  - **CRD missing** → run the OAS → schema pipeline, `crdgen.Generate`, and apply the CRD (and the Configuration
    CRD if there are configuration fields or security schemes). It then returns; a later reconcile deploys RDC.
  - **CRD present** → call `deploy.Deploy(...)` to render+apply the RDC Deployment/ConfigMap/RBAC, and store the
    resulting digest in `status.digest`.
- **`Update`** re-runs `deploy.Deploy(...)` and refreshes the digest. (It does **not**
  regenerate the CRD — the CRD's `kind`, group, identifiers, etc. are immutable; see [Conventions](#conventions--gotchas).)
- **`Delete`** undeploys RDC. Two finalizers are involved:
  - `composition.krateo.io/restresources-still-exist-finalizer` — blocks deletion while instances of the
    generated CRD still exist (managed in `manageFinalizers`).

### Status caching of `hasSecuritySchemes`

To avoid fetching and parsing the OAS on *every* `Observe`, the controller caches whether the OAS has security
schemes in `status.hasSecuritySchemes` (a `*bool`). On the first `Observe` (before `Create` has populated it) it
falls back to fetching the document once, defaulting to `true` on error. See the `hasSecuritySchemes` resolution
near the top of `Observe`.

---

## The OAS → CRD generation pipeline

This runs inside `Create` when the CRD does not yet exist. End to end:

```
OAS bytes (from filegetter)
  │
  ▼  parser.Parse()                         internal/tools/oas2jsonschema/libopenapi_adapter.go
OASDocument (library-agnostic interface)
  │
  ▼  NewOASSchemaGenerator(doc, genConfig, resourceConfig).Generate()    generator.go
  │     ├─ BuildSpecSchema()           spec_builder.go   (create requestBody + params + configurationRef)
  │     ├─ BuildStatusSchema()         status_builder.go (get/findby response → identifiers + additionalStatusFields)
  │     ├─ BuildConfigurationSchema()  configuration_builder.go (params grouped by `in` + auth methods)
  │     └─ ValidateSchemas()           validator.go      (compares create/update/findby vs the base get/findby schema)
  │
  ▼  GenerationResult { SpecSchema, StatusSchema, ConfigurationSchema, GenerationWarnings, ValidationWarnings }
  │
  ▼  crdgen.Generate(opts)             github.com/krateoplatformops/plumbing/crdgen
  │
  ▼  crd.Unmarshal + kube.Apply        the CRD (and Configuration CRD) are applied to the cluster
```

Internally each builder produces a library-agnostic `Schema` (`types.go`), which is then post-processed by
`prepareSchemaForCRD` (number/allOf normalization) and serialized to JSON by `schemaToMap`/`GenerateJsonSchema`
(`helpers.go`) before being handed to `crdgen`.

The mapping from a `RestDefinition` to the generator's inputs happens in `Create`,
where the CRD-facing `VerbsDescription`/`ConfigurationField` types are
"shimmed" into the package-local `oas2jsonschema.Verb`/`ConfigurationField` types so that the schema package has
**no dependency on the CRD API types**.

---

## The `oas2jsonschema` package in depth

This package is deliberately decoupled from the OpenAPI parsing library behind interfaces, which is what makes it
unit-testable without real OAS parsing.

| File | Responsibility |
|---|---|
| `interfaces.go` | `Parser`, `OASDocument`, `PathItem`, `Operation` — the library-agnostic contracts |
| `libopenapi_adapter.go` | The only file that imports `libopenapi`; converts library types → domain `Schema` |
| `types.go` | The domain model: `Schema`, `Property`, `ResourceConfig`, `GeneratorConfig`, `GenerationResult` |
| `generator.go` | `OASSchemaGenerator.Generate()` — orchestrates the builders + validator |
| `spec_builder.go` | Builds the CRD `spec` schema |
| `status_builder.go` | Builds the CRD `status` schema |
| `configuration_builder.go` | Builds the Configuration CRD schema (parameters + authentication) |
| `validator.go` | Cross-checks that create/update/findby response shapes are compatible with the base |
| `extractor.go` | Locates base schemas (create requestBody, get/findby response) and parameters |
| `helpers.go` | `prepareSchemaForCRD`, `schemaToMap`, type helpers |
| `schema_reflection.go` | Reflection-based schema generation for the built-in `BasicAuth`/`BearerAuth` types |
| `errors.go` | Structured error/warning types and codes |
| `annotator.go` | (currently commented out) historical duplicate-field-name annotation for crdgen |

### Key domain types

- **`Schema`** is the library-agnostic JSON-Schema-ish node. Note `Properties` is a **`[]Property` slice, not a
  map** — this is intentional, to preserve field ordering. Keep that invariant in mind when adding logic that
  iterates properties (don't introduce map-driven ordering, or generated CRDs become non-deterministic).
- **`GeneratorConfig`** holds defaults like `AcceptedMIMETypes` (`application/json`), `SuccessCodes`
  (`{200, 201}`), and recursion limits. `DefaultGeneratorConfig()` is the production config.
- **`ResourceConfig`** carries the per-RestDefinition inputs: verbs, identifiers, additional status fields,
  configuration fields, excluded spec fields.

### Recursion safety

OAS documents can be deeply nested or circular. Schema-walking functions use a `safety.RecursionGuard`
(depth + node-count + timeout) and a `visited` map for cycle detection. When you add a new recursive traversal,
follow the existing pattern (`NewRecursionGuard` → `WithContext` → `defer cancel` → `…Rec` with a `visited` map).

---

## RDC deployment & the drift digest

When the CRD exists, `Create`/`Update` call `deploy.Deploy(...)` (`internal/tools/deploy/deploy.go`), which:

1. Renders 7 objects from on-disk YAML templates: `ServiceAccount`, `ClusterRole`, `ClusterRoleBinding`,
   `Role`, `RoleBinding`, `ConfigMap`, `Deployment`.
2. Applies them to the cluster.
3. Computes a cumulative **FNV-64 digest** (`internal/tools/hash`) over the applied objects, stored in
   `status.digest`.

On every `Observe`, the controller recomputes the digest two ways — `deploy.Deploy(DryRunServer:true)` and
`deploy.Lookup(...)` — and compares both against `status.digest` to decide whether RDC needs re-applying /
restarting.

> ⚠️ **The templates are not compiled into the image.** They are mounted from ConfigMaps
> (`manifests/rdc/{cm,depl,rbac}.yaml`) into the provider pod at `/tmp/assets/...`. The paths are overridable
> via the env vars `RDC_TEMPLATE_DEPLOYMENT_PATH`, `RDC_TEMPLATE_CONFIGMAP_PATH`, and `RDC_RBAC_CONFIG_FOLDER`
> (the integration tests use these to point at `testdata/setup/rdc/`).

> ⚠️ **The digest is order-dependent.** It is accumulated by a sequence of `SumHash` calls
> (RBAC → ConfigMap → Deployment). If you change what is rendered, applied, or the order of hashing in `Deploy`,
> you must make the identical change in `Lookup`, or drift detection will flap and cause reconcile/restart loops.
> Note that `Deploy` and `Lookup` currently re-implement this sequence independently, so the two paths must be
> kept byte-for-byte in sync by hand.

---

## How OASGen and the rest-dynamic-controller fit together

These are two repositories that form one product:

- **oasgen-provider** (this repo) — generates the CRD and deploys RDC. Compile-time dependency on `crdgen` and
  the parsed OAS.
- **rest-dynamic-controller (RDC)** — the generic runtime controller that reconciles instances of the generated
  CRD against the REST API. Deployed *by* this provider, one instance per `RestDefinition`, configured via
  `REST_CONTROLLER_*` env vars (group/version/resource/namespace).

Some behaviors are **shared contracts that must agree on both sides**, so when you change one, check the other:

- **Success codes / response shapes.** OASGen validates schemas against `SuccessCodes = {200, 201}`; RDC accepts
  only spec-declared 2xx at call time. An API that returns `202` or a `200` with an empty body can fall between
  the two. If you touch success-code handling, change both repos together.
- **Response codes as map keys.** Both sides currently key responses by integer code, so OAS `default`/`2XX`
  range responses are not handled. Same caveat.
- **Identifiers / status contract.** OASGen decides which response fields become `status`; RDC populates and
  compares them. Their notion of the status shape must match.
- **libopenapi fork.** Both repos `replace` `pb33f/libopenapi` with the Krateo fork. Keep the fork version in
  sync so OAS parsing is identical at generation time and runtime.

---

## Local development workflow

### Prerequisites

- Go **1.25.3+** (matches `go.mod`; with an older local toolchain use `GOTOOLCHAIN=auto`).
- Docker, [`kind`](https://kind.sigs.k8s.io/), `kubectl`, [`ko`](https://ko.build/), and `jq`.

### Spin up a cluster and deploy

The `scripts/` helpers wrap the common loop:

```sh
./scripts/kind-up.sh     # create a kind cluster (no-op if one exists)
./scripts/build.sh       # ko build the image into the kind cluster (KO_DOCKER_REPO=kind.local)
./scripts/reload.sh      # build + reapply crds/, manifests/rdc/, manifests/  (the full redeploy loop)
./scripts/kind-down.sh   # tear the cluster down
```

`reload.sh` is the inner loop: it rebuilds the image and re-applies the CRD, the RDC asset ConfigMaps
(`manifests/rdc/`), and the provider Deployment (`manifests/`). After it runs, apply a sample to exercise it:

```sh
kubectl apply -f samples/<example>/...
```

> Note: `manifests/deploy.yaml` is a **dev** manifest (`kind.local/oasgen-provider:latest`,
> `imagePullPolicy: Never`, namespace `demo-system`). Production install is via the Krateo Helm chart, not these
> manifests.

### Running with debug logging

The provider takes flags (also settable via `OASGEN_PROVIDER_*` env vars) — see `main.go`:

- `--debug` / `OASGEN_PROVIDER_DEBUG` — debug-level logs (the schema-generation warnings are logged at debug).
- `--poll` / `OASGEN_PROVIDER_POLL_INTERVAL` — drift poll interval (default `3m`).
- `--sync` / `OASGEN_PROVIDER_SYNC` — manager cache resync (default `1h`).
- `--max-reconcile-rate`, `--leader-election`, `--min/max-error-retry-interval`.

---

## Code generation

The API types in `apis/restdefinitions/v1alpha1/types.go` are the **source of truth**. The deepcopy code and the
`RestDefinition` CRD YAML are generated from them via `controller-gen` (driven by `apis/generate.go`):

```sh
./scripts/generate.sh    # runs: go mod tidy && go generate ./...
```

This regenerates `apis/.../zz_generated.deepcopy.go` and rewrites `crds/`. **Never hand-edit those files.**

When you change `types.go` (add a field, a `+kubebuilder` marker, a validation rule), always re-run generation
and commit the regenerated `crds/` and `zz_generated.deepcopy.go` alongside it.

Kubebuilder markers already in use that are worth knowing:
- `+kubebuilder:validation:XValidation` (CEL) — e.g. immutability rules (`self == oldSelf`) on `kind`,
  `identifiers`, `resourceGroup`, etc.
- `+kubebuilder:validation:Enum`, `+kubebuilder:validation:Pattern`, `+required`/`+optional`.

---

## Testing

### Unit tests

```sh
go test -cover -v ./...
```

The unit tests live next to the code. The richest suite is `internal/tools/oas2jsonschema/*_test.go`: it uses
behavioral `t.Run` subtests, table-driven validator cases, interface mocks (`mocks_test.go`), and OAS fixtures
under `internal/tools/oas2jsonschema/testdata/`. **When you add or change schema-generation logic, add a
behavioral test with a small OAS fixture** — that is the established pattern and the fastest way to lock in
behavior.

### Integration tests

```sh
go test -tags=integration -cover -v ./...
```

Integration tests (gated by the `integration` build tag) use `sigs.k8s.io/e2e-framework` and provision a **real
kind cluster** in `TestMain` (create cluster, install the CRD from `crds/`, create namespaces, tear down). They
drive the actual `connector`/`ExternalClient` and assert on the generated CRD's OpenAPI schema. They point the
`RDC_TEMPLATE_*` env vars at `testdata/setup/rdc/`.

> Requirements: Docker + kind must be available; these tests are slow. If you add lifecycle scenarios, prefer
> `wait.For(conditions...)` polling over fixed `time.Sleep` (the create path currently uses sleeps — don't copy
> that pattern).

### Coverage helper

```sh
./scripts/test.sh        # go test ./... -coverprofile + per-func summary
```

---

## How to extend the provider

A few common tasks and where they touch:

### Add a field to the RestDefinition API
1. Edit `apis/restdefinitions/v1alpha1/types.go`; add `+kubebuilder` markers (and immutability `XValidation` if
   the field must not change after CRD generation).
2. If the field influences schema generation, thread it through the shim in `Create`
   into `oas2jsonschema.ResourceConfig`, and consume it in the relevant builder.
3. `./scripts/generate.sh`; commit the regenerated `crds/` + deepcopy.
4. Add unit tests (schema builders) and, if it affects lifecycle, an integration scenario.

### Change how the spec/status/configuration schema is built
- Work in the relevant builder (`spec_builder.go` / `status_builder.go` / `configuration_builder.go`).
- Remember `Schema.Properties` is an **ordered slice** — preserve ordering; do not introduce map iteration into
  emitted output (it makes CRDs non-deterministic).
- Surface problems as **warnings** on `GenerationResult` (and an error code in `errors.go`) rather than dropping
  them silently — see the silent-failure caveat below.
- Add a `testdata` OAS fixture + behavioral test.

### Support a new authentication scheme
- Add a `SecuritySchemeType` constant in `types.go`, handle it in `configuration_builder.go`
  (`createSchemaForSecurityScheme`), and add a corresponding auth struct if needed
  (`schema_reflection.go` generates schemas for `BasicAuth`/`BearerAuth` today).
- Coordinate with RDC: RDC must know how to inject the new scheme into requests.

### Change what gets deployed for RDC
- Edit the templates in `manifests/rdc/{depl,cm,rbac}.yaml` (and the matching `testdata/setup/rdc/` fixtures).
- If you change *which* objects are rendered/applied or the hashing order, update **both** `deploy.Deploy` and
  `deploy.Lookup` to keep the digest consistent.

---

## Conventions & gotchas

- **Generated files are off-limits.** `crds/`, `apis/**/zz_generated.deepcopy.go` are produced by
  `controller-gen`. Edit `types.go` and regenerate.
- **Immutability matters.** `kind`, `resourceGroup`, `identifiers`, `additionalStatusFields`,
  `configurationFields`, `excludedSpecFields` carry `self == oldSelf` CEL rules — the generated CRD's identity
  can't change after creation. `Update` deliberately does *not* regenerate the CRD.
- **Ordering is load-bearing.** `Schema.Properties` is a slice precisely so generated schemas are deterministic.
  Avoid emitting output from Go map iteration.
- **Prefer warnings over silent drops.** The codebase has a tendency to swallow edge cases behind commented-out
  logs. New code should append to `GenerationResult.GenerationWarnings`/`ValidationWarnings` (with a code from
  `errors.go`) so misconfigurations are visible at debug level.
- **The digest must stay symmetric.** `deploy.Deploy` and `deploy.Lookup` must hash the same objects in the same
  order; the two paths re-implement the sequence independently, so changing one without the other causes drift
  detection to flap.
- **The OAS is untrusted input.** It comes from a user-controlled `oasPath`. Be mindful when touching
  `filegetter` / template rendering — consider SSRF, response size limits, and cross-namespace reads.
- **Two repos, one contract.** Behaviors around success codes, response shapes, identifiers, and the libopenapi
  fork must stay aligned with rest-dynamic-controller.

---

## Glossary

- **RestDefinition** — the CR you give this provider; describes the OAS, verbs, identifiers, etc.
- **RestResource** — an instance of the *generated* CRD (e.g. a `Repository`), reconciled by RDC.
- **RDC** — rest-dynamic-controller, the generic runtime controller deployed per RestDefinition.
- **Configuration CRD** — an auxiliary generated CRD holding parameters and authentication for a RestResource.
- **Verb / action** — `create`, `get`, `update`, `delete`, `findby`; mapped to OAS operations.
- **Identifiers** — fields used to locate the external resource (used by `findby` and to populate status).
- **Digest** — FNV hash of the rendered RDC objects, stored in `status.digest` for drift detection.
- **Vacuum version** — the non-served storage CRD version trick in `crd/utils.go` for multi-version storage.

---

*This developer guide describes how the provider is intended to work. For user-facing documentation, see the
[README](../../README.md), the [Usage Guide](../USAGE_GUIDE.md), and the
[RestDefinition CRD reference](../restdefinition-crd-reference.md).*
