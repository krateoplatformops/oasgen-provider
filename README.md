# Krateo OASGen Provider

The Krateo OASGen Provider is a Kubernetes controller that generates Custom Resource Definitions (CRDs) and controllers to manage resources directly from OpenAPI Specification (OAS) 3.1 documents (with support for OAS 3.0). It enables seamless integration of API-defined resources into Kubernetes environments.

## Summary

- [Summary](#summary)
- [Glossary](#glossary)
- [Architecture](#architecture)
- [Workflow](#workflow)
- [Requirements](#requirements)
- [RestDefinition Specifications](#restdefinition-specifications)
  - [Authentication](#authentication)
  - [API Endpoints Requirements](#api-endpoints-requirements)
- [Getting Started](#getting-started)
- [Example](#example)
- [Security Features](#security-features)
- [Best Practices](#best-practices)
- [Troubleshooting](#troubleshooting)

## Glossary

- **CRD (Custom Resource Definition):** A Kubernetes resource that defines custom objects and their schemas.
- **RestDefinition:** A custom resource that defines how API resources are managed in Kubernetes based on OAS specifications.
- **RDC (Rest Dynamic Controller):** A controller deployed by the provider to manage resources defined by a RestDefinition.
- **OAS (OpenAPI Specification):** A standard, language-agnostic interface description for REST APIs.
- **WebService:** A wrapper service that maintains consistent API interfaces when needed.
- **BasicAuth:** A simple authentication method using username and password credentials.

## Architecture

![Generator Architecture Image](_diagram/generator.png "Generator Architecture")

The diagram illustrates how the OASGen Provider processes OpenAPI Specifications to generate CRDs and deploy the Rest Dynamic Controller (RDC). The RDC manages custom resources and interacts with external APIs, optionally through wrapper web services when needed.

## Workflow

1. User applies a RestDefinition CR
2. Provider fetches the OAS specification
3. Provider generates CRD based on the OAS schema
4. Provider deploys the Rest Dynamic Controller
5. Controller manages custom resources according to API specifications
6. Resources are synchronized with external APIs

## Requirements

- Kubernetes cluster (v1.20+ recommended)
- Krateo BFF installed in the `krateo-system` namespace
- OpenAPI Specification 3.0+ documents for your APIs
- Network access to API endpoints from the cluster

## RestDefinition Specifications

### Authentication

The provider supports Basic Authentication out of the box and will generate appropriate CRDs if the OAS specifies authentication methods.

Example BasicAuth CR:
```yaml
kind: BasicAuth
apiVersion: azure.devops.com/v1alpha1
metadata:
  name: basicauth-azure
spec:
  username: admin
  passwordRef:
    name: azdevops-gen
    namespace: default
    key: token
```

### API Endpoints Requirements

1. Field names must be consistent across all actions (`create`, `update`, `findby`, `get`, `delete`)
2. API responses must contain all fields defined in the CRD
3. Path parameters and body fields should use consistent naming

## Getting Started

1. Install Krateo BFF:
```sh
helm repo add krateo https://charts.krateo.io
helm repo update
helm install bff krateo/bff --namespace krateo-system --create-namespace
```

2. Install OASGen Provider:
```sh
helm install krateo-oasgen-provider krateo/oasgen-provider --namespace krateo-system
```

3. Apply a RestDefinition:
```yaml
kind: RestDefinition
apiVersion: swaggergen.krateo.io/v1alpha1
metadata:
  name: def-pipelinepermissions
  namespace: default
spec: 
  oasPath: https://raw.githubusercontent.com/krateoplatformops/azuredevops-oas3/main/approvalandchecks/pipelinepermissions.yaml
  resourceGroup: azure.devops.com
  resource: 
    kind: PipelinePermission
    identifiers:
      - resourceType
      - resourceId
    verbsDescription:
    - action: findby
      method: POST
      path: /{organization}/{project}/_apis/pipelines/pipelinepermissions/{resourceType}/{resourceId}
    - action: get
      method: GET
      path: /{organization}/{project}/_apis/pipelines/pipelinepermissions/{resourceType}/{resourceId}
    - action: update
      method: PATCH
      path: /{organization}/{project}/_apis/pipelines/pipelinepermissions/{resourceType}/{resourceId}
```

## Example

Here's a complete example for managing Azure DevOps Pipeline Permissions:

1. Apply the RestDefinition (as shown above)
2. Create a BasicAuth secret:
```sh
kubectl create secret generic azdevops-gen --from-literal=token=your-pat-token
```

3. Apply the PipelinePermission CR:
```yaml
kind: PipelinePermission
apiVersion: azure.devops.com/v1alpha1
metadata:
  name: pipelineperm-1
  namespace: default
  annotations:
    krateo.io/connector-verbose: "true"
spec:
  authenticationRefs:
    basicAuthRef: basicauth-azure
  api-version: 7.0-preview.1
  organization: yourorg
  project: yourproject
  resourceType: environment
  resourceId: "1"
  pipelines:
  - id: 42
    authorized: false
```

## Security Features

- Automatic generation of RBAC policies for custom resources
- Secure credential management through Kubernetes secrets
- Field validation based on OAS schemas
- Optional web service wrappers for additional security layers

## Best Practices

1. Always use OAS 3.0+ specifications
2. Maintain consistent field naming across API endpoints
3. Use web service wrappers when API interfaces are inconsistent
4. Regularly update OAS documents to match API changes
5. Monitor controller logs with `krateo.io/connector-verbose: "true"`

## Troubleshooting

1. **Conversion from OAS 2.0 to 3.0:**
   - Use [Swagger Editor](https://editor.swagger.io) for conversion
   - Manually review and correct any conversion issues

2. **Inconsistent API interfaces:**
   - Create a web service wrapper to normalize interfaces
   - Example implementations available in [Java](https://github.com/krateoplatformops/azuredevops-oas3-plugin) and [Python](https://github.com/krateoplatformops/github-oas3-plugin)

3. **Authentication issues:**
   - Verify secret references in BasicAuth CRs
   - Check network connectivity to API endpoints
   - Enable verbose logging with annotations

For complete CRD specifications, visit [this link](https://doc.crds.dev/github.com/krateoplatformops/oasgen-provider).