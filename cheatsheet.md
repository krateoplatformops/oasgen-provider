# Comprehensive Guide to Provider Generation with Krateo Operator Generator (KOG)

## Table of Contents

- [Comprehensive Guide to Provider Generation with Krateo Operator Generator (KOG)](#comprehensive-guide-to-provider-generation-with-krateo-operator-generator-kog)
  - [Table of Contents](#table-of-contents)
  - [Prerequisites](#prerequisites)
  - [Simple Case: External APIs Compatible with K8s Resource Management](#simple-case-external-apis-compatible-with-k8s-resource-management)
  - [Extended Example: External API that requires a webservice to handle external API calls](#extended-example-external-api-that-requires-a-webservice-to-handle-external-api-calls)
  - [Best Practices](#best-practices)
  - [Troubleshooting](#troubleshooting)



## Prerequisites
- Kubernetes cluster with Krateo installed
- `kubectl` configured to access your cluster
- OpenAPI Specification (OAS) 3.0+ for your target API

## Simple Case: External APIs Compatible with K8s Resource Management

This guide provides a step-by-step approach to generating a provider for managing GitHub repositories using the Krateo Operator Generator (KOG). It assumes you have a basic understanding of Kubernetes and OpenAPI specifications.

### Step 1: Prepare Your OpenAPI Specification

1. **Obtain or generate** the OAS for your target API
   - Example: GitHub API OAS available at [GitHub's REST API description](https://github.com/github/rest-api-description/blob/main/descriptions/ghes-3.9/ghes-3.9.yaml)
   
2. **Scope your OAS** to only include necessary endpoints:
   - Recommended for large APIs to reduce complexity
   - Create separate files for different resource types (e.g., `repositories.yaml`, `teamrepo.yaml`)

3. **Add authentication** information if missing from original OAS:
```diff
   components:
+    securitySchemes
+       oauth:
+         type: http
+        scheme: bearer
```

### Step 2: Prepare Kubernetes Environment

1. Create a dedicated namespace:
   ```bash
   kubectl create namespace gh-system
   ```

2. Store your OAS as a ConfigMap: (In this example, we use a sample OAS for GitHub repositories stored in `samples/cheatsheet/assets/repo.yaml` of this repository)
   ```bash
   kubectl create configmap repo --from-file=samples/cheatsheet/assets/repo.yaml -n gh-system
   ```

### Step 3: Create RestDefinition for GitHub Repositories

```bash
cat <<EOF | kubectl apply -f -
apiVersion: swaggergen.krateo.io/v1alpha1
kind: RestDefinition
metadata:
  name: gh-repo
  namespace: gh-system
spec:
  oasPath: configmap://gh-system/repo/repo.yaml
  resourceGroup: github.kog.krateo.io
  resource: 
    kind: Repo
    identifiers:
      - id 
      - name
      - html_url
    verbsDescription:
    - action: create
      method: POST
      path: /orgs/{org}/repos
    - action: delete
      method: DELETE
      path: /repos/{org}/{name}
    - action: get
      method: GET
      path: /repos/{org}/{name}
    - action: update
      method: PATCH
      path: /repos/{org}/{name}
EOF
```

#### Wait for the CRD and Controller to be Created
You can check the status of the creation by running:

```bash
kubectl wait restdefinition gh-repo --for condition=Ready=True --namespace gh-system --timeout=600s
```

#### Handling Schema Validation Issues

Common error you might encounter:
```text
Cannot create external resource [...] error: "generating CRD: missing type in schema 'Title'"
```

##### Solution 1: Simplify Complex Types (Recommended)
```yaml
title:
  type: string
  description: The title of the issue.
```
**Note:** Requires API wrapper for type conversion if original API expects different types.

##### Solution 2: Use AdditionalProperties (Flexible but Less Safe)
```yaml
title:
  additionalProperties: true
  type: object
```
**Note:** Bypasses validation but may cause runtime errors.

### Step 4: Verification Steps

1. Check CRD creation:
   ```bash
   kubectl get crds | grep github.kog.krateo.io 
   ```

   You should see:
   ```text
    bearerauths.github.kog.krateo.io           2025-06-13T08:28:06Z
    repoes.github.kog.krateo.io                2025-06-13T08:28:06Z
   ```

  If you see `bearerauths` and `repoes`, the CRDs have been created successfully. The second CRD represents the `repo` object. The first one is the `bearerauth` object, which is used to authenticate requests to the GitHub API.

1. Verify controller pod:
   ```bash
   kubectl get deploy -n gh-system
   ```

You should see a deployment named `gh-repo-controller`, which is responsible for managing the `Repo` resources.

   If you see the deployment, you can check the logs of the controller pod to see if it is running correctly:
   ```bash
   kubectl logs deploy/gh-repo-controller -n gh-system
   ```

2. Check RestDefinition status:
   ```bash
   kubectl get restdefinition -n gh-system
   kubectl describe restdefinition gh-repo -n gh-system
   ```

At this point you have a running operator able to handle GitHub repositories. You can create, update, and delete repositories using the custom resource.

### Step 5: Create the Custom Resources

Create a custom resource for the `bearerauth` object. This is used to authenticate requests to the GitHub API. The `bearerauth` object contains a reference to the token that is used to authenticate the requests. The token is stored in a Kubernetes secret.

So first, create a secret with your GitHub token: (generate a personal access token with the necessary permissions from your GitHub account settings)

```bash
kubectl create secret generic gh-token --from-literal=token=<token> -n gh-system 
```

```bash
cat <<EOF | kubectl apply -f -
apiVersion: github.kog.krateo.io/v1alpha1
kind: BearerAuth
metadata:
  name: gh-bearer
  namespace: gh-system
spec:
  tokenRef:
    key: token
    name: gh-token
    namespace: gh-system
EOF
```

Create a custom resource for the `Repo` object. This is used to create, update, and delete repositories in the GitHub API. The `Repo` object contains the reference to the `bearerauth` object that is used to authenticate the requests.

```bash
cat <<EOF | kubectl apply -f -
apiVersion: github.kog.krateo.io/v1alpha1
kind: Repo
metadata:
  name: gh-repo-1
  namespace: gh-system
spec:
  authenticationRefs:
    bearerAuthRef: gh-bearer 
  org: krateoplatformops-test
  name: krateo-test-repo
  description: A short description of the repository set by Krateo
  visibility: public
  has_issues: true
EOF
```

You will expect that the controller creates a repository in your GitHub account with the name `krateo-test-repo` under the organization `krateoplatformops-test`. You can check the status of the repository by running:

```bash
kubectl describe repo.github.kog.krateo.io/gh-repo-1 -n gh-system
```

You should see a successful creation event, which indicates that the repository was created successfully.

``` text
Events:
  Type     Reason                         Age                  From  Message
  ----     ------                         ----                 ----  -------
  Normal   CreatedExternalResource        6m30s                      Successfully requested creation of external resource
```

Any edits to the `Repo` custom resource will trigger the controller to update the corresponding repository in GitHub.

### Step 6: Patch the Custom Resource
To update the repository, you can patch the `Repo` custom resource. For example, to change the description of the repository:

```bash
cat <<EOF | kubectl apply -f -
apiVersion: github.kog.krateo.io/v1alpha1
kind: Repo
metadata:
  name: gh-repo-1
  namespace: gh-system
spec:
  authenticationRefs:
    bearerAuthRef: gh-bearer 
  org: krateoplatformops-test
  name: krateo-test-repo
  description: A new description of the repository set by Krateo
  visibility: public
  has_issues: true
EOF
```
This will trigger the controller to update the repository in GitHub with the new description.

You should see an event for the Repo resource indicating that the external resource was updated successfully:

```text
Events:
  Type     Reason                         Age                  From  Message
  ----     ------                         ----                 ----  -------
  Normal   UpdatedExternalResource        10s                       Successfully requested update of external resource
```

### Step 7: Delete the Custom Resource
To delete the repository, you can delete the `Repo` custom resource:

```bash
kubectl delete repo.github.kog.krateo.io gh-repo-1 -n gh-system
```

This will trigger the controller to delete the corresponding repository in GitHub.
You should see an event for the Repo resource indicating that the external resource was deleted successfully:

You can check the status of the deletion by running:

```bash
kubectl get events --sort-by='.lastTimestamp' -n gh-system | grep repo/gh-repo-1
```

```text
Events:
  Type     Reason                         Age                  From  Message
  ----     ------                         ----                 ----  -------
  Normal   DeletedExternalResource      repo/gh-repo-1        Successfully requested deletion of external resource
```

## Extended Example: External API that requires a webservice to handle external API calls

This example demonstrates how to create a Krateo provider for managing GitHub repositories using an external web service to handle API calls. This approach is useful when the API isn't directly compatible with Kubernetes resource management or requires additional processing.

For an API to be compatible with Kubernetes resource management, it should create, update, and delete resources in a way that is similar to Kubernetes resources. This means the API should support the same operations as Kubernetes resources, such as create, update, delete, and get. If the API doesn't support these operations or requires additional processing, you can use an external web service to handle the API calls.

This example assumes you have a basic understanding of Kubernetes, OpenAPI specifications, and web service development.

**Note:** In this example, we'll develop a web service that handles API calls to the GitHub API. The web service will be responsible for creating, updating, and deleting repositories in GitHub. While this example uses Go, you can use any programming language and framework you're comfortable with.

## Step 1: Prepare Your OpenAPI Specification

1. **Obtain or generate** the OAS for your target API
   - Example: GitHub API OAS available at [GitHub's REST API description](https://github.com/github/rest-api-description/blob/main/descriptions/ghes-3.9/ghes-3.9.yaml)
   
2. **Scope your OAS** to only include necessary endpoints:
   - Recommended for large APIs to reduce complexity
   - Create separate files for different resource types (e.g., `repositories.yaml`, `teamrepo.yaml`)

3. **Add authentication** information if missing from the original OAS:
```diff
    components:
+     securitySchemes:
+       oauth:
+         type: http
+         scheme: bearer
```

### Step 2: Prepare Kubernetes Environment

1. Create a dedicated namespace:
   ```bash
   kubectl create namespace gh-system
   ```

2. Store your OAS as a ConfigMap (in this example, we use a sample OAS for GitHub teamrepos stored in assets of this repository):
   ```bash
   kubectl create configmap teamrepo --from-file=samples/cheatsheet/assets/teamrepo_no_ws.yaml -n gh-system
   ```

### Step 3: Create RestDefinition for GitHub Repositories

```bash
cat <<EOF | kubectl apply -f -
apiVersion: swaggergen.krateo.io/v1alpha1
kind: RestDefinition
metadata:
  name: gh-teamrepo
  namespace: gh-system
spec:
  oasPath: configmap://gh-system/teamrepo/teamrepo_no_ws.yaml
  resourceGroup: github.kog.krateo.io
  resource: 
    kind: TeamRepo
    identifiers:
      - id 
      - name
      - full_name
      - permission
    verbsDescription:
    - action: create
      method: PUT
      path: /orgs/{org}/teams/{team_slug}/repos/{owner}/{repo}
    - action: delete
      method: DELETE
      path: /orgs/{org}/teams/{team_slug}/repos/{owner}/{repo}
    - action: get
      method: GET
      path: /orgs/{org}/teams/{team_slug}/repos/{owner}/{repo}
    - action: update
      method: PUT
      path: /orgs/{org}/teams/{team_slug}/repos/{owner}/{repo}
EOF
```

### Wait for the CRD and Controller to be Created

You can check the creation status by running:

```bash
kubectl wait restdefinition gh-teamrepo --for condition=Ready=True --namespace gh-system --timeout=600s
```

### Step 4: Verification Steps

1. **Check CRD creation:**
   ```bash
   kubectl get crds | grep github.kog.krateo.io 
   ```

   You should see:
   ```text
   bearerauths.github.kog.krateo.io           2025-06-13T08:28:06Z
   teamrepos.github.kog.krateo.io             2025-06-13T08:28:06Z
   ```

   If you see `bearerauths` and `teamrepos`, the CRDs are created successfully. The second CRD represents the `teamrepo` object, while the first one is the `bearerauth` object used to authenticate requests to the GitHub API.

   **Note:** If you've previously created the `repo` RestDefinition, you'll also see the `repoes.github.kog.krateo.io` CRD. However, the `bearerauths.github.kog.krateo.io` CRD is shared between RestDefinitions because they use the same group and authentication scheme.

2. **Verify controller deployment:**
   ```bash
   kubectl get deploy -n gh-system
   ```
   
   You should see a deployment named `gh-teamrepo-controller` responsible for managing the `teamrepo` resources.
   
   If you see the deployment, check the controller pod logs to verify it's running correctly:
   ```bash
   kubectl logs deploy/gh-teamrepo-controller -n gh-system
   ```

3. **Check RestDefinition status:**
   ```bash
   kubectl get restdefinition -n gh-system
   kubectl describe restdefinition gh-teamrepo -n gh-system
   ```

At this point, you have a running operator capable of handling GitHub teamrepos. You can create, update, and delete teamrepos using the custom resource.

### Step 5: Create the Custom Resources

Create a custom resource for the `bearerauth` object. (You can skip the secret and bearerAuth creation if you've completed the `repo` tutorial in the previous section.) This is used to authenticate requests to the GitHub API. The `bearerauth` object contains a reference to the token used for authentication, which is stored in a Kubernetes secret.

First, create a secret with your GitHub token (generate a personal access token with the necessary permissions from your GitHub account settings):

```bash
kubectl create secret generic gh-token --from-literal=token=<your-token> -n gh-system 
```

```bash
cat <<EOF | kubectl apply -f -
apiVersion: github.kog.krateo.io/v1alpha1
kind: BearerAuth
metadata:
  name: gh-bearer
  namespace: gh-system
spec:
  tokenRef:
    key: token
    name: gh-token
    namespace: gh-system
EOF
```

Create a custom resource for the `teamrepo` object. This is used to create, update, and delete teamrepos in the GitHub API. The `teamrepo` object contains a reference to the `bearerauth` object used for authentication:

```bash
cat <<EOF | kubectl apply -f -
apiVersion: github.kog.krateo.io/v1alpha1
kind: TeamRepo
metadata:
  name: test-teamrepo
  namespace: gh-system
spec:
  authenticationRefs:
    bearerAuthRef: gh-bearer
  org: krateoplatformops-test
  owner: krateoplatformops-test
  team_slug: prova
  repo: test-teamrepo
  permission: admin
EOF
```

The controller should add the GitHub team "prova" to the repository "test-teamrepo" in the organization "krateoplatformops-test" with "admin" permission. Check the teamrepo creation status by running:

```bash
kubectl describe teamrepo.github.kog.krateo.io/test-teamrepo -n gh-system
```

You should see the teamrepo creation status and any errors that occurred during the process.

In this case, you should see that the teamrepo was created successfully with status `Ready`: `True`, but you should also notice that the `Message` field states "Resource is assumed to be up-to-date. Returned body is nil."

```text
...
Status:
  Conditions:
    Last Transition Time:  2025-06-18T14:13:03Z
    Message:               Resource is assumed to be up-to-date. Returned body is nil.
    Reason:                Available
    Status:                True
    Type:                  Ready
Events:
  Type    Reason                   Age   From  Message
  ----    ------                   ----  ----  -------
  Normal  CreatedExternalResource  86s         Successfully requested creation of external resource
```

This message indicates that the controller was able to create the teamrepo in GitHub, but the response body from the GitHub API is nil. This is expected behavior as described [here](https://docs.github.com/en/rest/teams/teams?apiVersion=2022-11-28#check-team-permissions-for-a-repository). According to the GitHub API documentation, the response body for this endpoint is empty when the request is successful but the accept header isn't set to `application/vnd.github.v3+json`. This is a known limitation of the GitHub API, and it's not possible to change the accept header for requests made by the `rest-dynamic-controller` directly.

To add the header to the request, we need to implement a web service that handles API calls to the GitHub API. The web service will be responsible for adding the header to the request and returning the response body, since the `rest-dynamic-controller` doesn't support adding headers to requests made to external APIs.

### Step 6: Create the Web Service for TeamRepo Management

At this point, we need to implement a web service that handles API calls to the GitHub API. In this case, the web service will only be responsible for the `get` operation for teamrepos, because the `create`, `delete`, and `update` operations are handled directly by the controller.

To handle this case, we've implemented a web service that handles the `get` operation for teamrepos. You can check the implementation at this link: [GitHub Plugin for rest-dynamic-controller](https://github.com/krateoplatformops/github-rest-dynamic-controller-plugin/blob/main/internal/handlers/repo/repo.go). You can also check the [README](https://github.com/krateoplatformops/github-rest-dynamic-controller-plugin/blob/main/README.md) for more information on running and why it has been implemented.

**Note:** `rest-dynamic-controller` to check if the CR is up-to-date or not, checks the fields in the CR (spec and status) against the fields in the response body from the external API. It compare the fields at the same level, so if the response fields are more nested than the fields in the CR, it will not be able to compare them correctly. This is why we need to implement a web service that returns the response body with the same structure as the CR. This problem is quite common and a specific solution has been described [here](https://github.com/krateoplatformops/github-rest-dynamic-controller-plugin/blob/main/README.md#2-get-team-permission-in-a-repository)

```bash
cat <<EOF | kubectl apply -f -
---
apiVersion: v1
kind: Service
metadata:
  name: github-provider-plugin-krateo
  namespace: default
spec:
  selector:
    app: github-provider-plugin-krateo
  ports:
    - protocol: TCP
      port: 8080
      targetPort: 8080
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: github-provider-plugin-krateo
  namespace: default
spec:
  replicas: 1
  selector:
    matchLabels:
      app: github-provider-plugin-krateo
  template:
    metadata:
      labels:
        app: github-provider-plugin-krateo
    spec:
      containers:
        - name: github-provider-plugin-krateo
          image: ghcr.io/krateoplatformops/github-rest-dynamic-controller-plugin:0.0.2
          ports:
            - containerPort: 8080
EOF
```

### Step 7: Update the RestDefinition to Use the Web Service

Now we need to tell the `rest-dynamic-controller` to use the web service to handle the `get` operation for teamrepos. We can do this by adding the webservice URL to the OpenAPI specification of the `teamrepo` RestDefinition. We can accomplish this by adding the `servers` field to the endpoint in the OpenAPI specification (https://swagger.io/docs/specification/v3_0/api-host-and-base-path/#overriding-servers). In this case, the URL will be `http://github-provider-plugin-krateo.default.svc.cluster.local:8080` because the web service is running in the `default` namespace with the service name `github-provider-plugin-krateo`.

Let's create a new configmap with the updated OpenAPI specification:

```bash
kubectl create configmap teamrepo-ws --from-file=samples/cheatsheet/assets/teamrepo_ws.yaml -n gh-system
```

As you can see, the new OpenAPI specification also includes another endpoint for the `get` operation that points to the web service URL:

```yaml
...
paths:
  "/teamrepository/orgs/{org}/teams/{team_slug}/repos/{owner}/{repo}":
    get:
      servers:
      - url: http://github-provider-plugin-krateo.default.svc.cluster.local:8080
...
```

This means that when the `rest-dynamic-controller` handles the `get` operation for teamrepos, it will call the web service instead of the GitHub API directly. The web service will then add the necessary headers to the request and return the response body.

Now we need to update the `RestDefinition` to use the new configmap:

```bash
cat <<EOF | kubectl apply -f -
apiVersion: swaggergen.krateo.io/v1alpha1
kind: RestDefinition
metadata:
  name: gh-teamrepo
  namespace: gh-system
spec:
  oasPath: configmap://gh-system/teamrepo-ws/teamrepo_ws.yaml
  resourceGroup: github.kog.krateo.io
  resource: 
    kind: TeamRepo
    identifiers:
      - id
      - permissions
      - html_url
    verbsDescription:
    - action: create
      method: PUT
      path: /orgs/{org}/teams/{team_slug}/repos/{owner}/{repo}
    - action: delete
      method: DELETE
      path: /orgs/{org}/teams/{team_slug}/repos/{owner}/{repo}
    - action: get
      method: GET
      path: /teamrepository/orgs/{org}/teams/{team_slug}/repos/{owner}/{repo}
    - action: update
      method: PUT
      path: /orgs/{org}/teams/{team_slug}/repos/{owner}/{repo}
EOF
```

We expect the controller to update the RestDefinition and start using the web service to handle the `get` operation for teamrepos. Check the RestDefinition status by running:

```bash
kubectl describe restdefinition gh-teamrepo -n gh-system
```

You should see that the message field is now empty, which means the RestDefinition is ready and correctly observed by the controller.

To check if the remote resource changes along with the custom resource, you can run the following command to change the teamrepo's permission:

```bash
cat <<EOF | kubectl apply -f -
apiVersion: github.kog.krateo.io/v1alpha1
kind: TeamRepo
metadata:
  name: test-teamrepo
  namespace: gh-system
spec:
  authenticationRefs:
    bearerAuthRef: gh-bearer
  org: krateoplatformops-test
  owner: krateoplatformops-test
  team_slug: prova
  repo: test-teamrepo
  permission: pull
EOF
```

After a few seconds, you should see that the teamrepo's permission is updated in GitHub. Check the teamrepo status by running:

```bash
kubectl describe teamrepo.github.kog.krateo.io/test-teamrepo -n gh-system
```

You should see that the permission is updated to `pull`, the status is set to `Ready`: `True`, and events indicate that the external resource was updated successfully:

```text
Events:
  Type     Reason                         Age                  From  Message
  ----     ------                         ----                 ----  -------
  Normal   UpdatedExternalResource        77s (x2 over 80s)             Successfully requested update of external resource
```

### Step 8: Delete the Custom Resource

To delete the teamrepo, you can delete the `TeamRepo` custom resource:

```bash
kubectl delete teamrepo.github.kog.krateo.io test-teamrepo -n gh-system
```

This will trigger the controller to delete the corresponding teamrepo in GitHub.

You should see an event for the TeamRepo resource indicating that the external resource was deleted successfully. Check the deletion status by running:

```bash
kubectl get events --sort-by='.lastTimestamp' -n gh-system | grep teamrepo/test-teamrepo
```

```text
Events:
  Type     Reason                         Age                  From  Message
  ----     ------                         ----                 ----  -------
  Normal   DeletedExternalResource      teamrepo/test-teamrepo        Successfully requested deletion of external resource
```

This indicates that the teamrepo was deleted successfully from GitHub. You can manually check the GitHub UI to confirm that the teamrepo is no longer present.

## Best Practices

1. **Incremental Development**: Start with a small subset of endpoints
2. **Monitoring**: Set up alerts for the `rest-dynamic-controller` pods
3. **Documentation**: Maintain a changelog for your OAS modifications
   
## Troubleshooting

### Common Issues

- **CRD not created**: Check RestDefinition status and controller logs
- **Authentication issues**: Verify securitySchemes in OAS match actual API requirements
- **Type errors**: Ensure all fields have proper type definitions

### Detailed Solutions

1. **Conversion from OAS 2.0 to 3.0:**
   - Use [Swagger Editor](https://editor.swagger.io) for conversion
   - Manually review and correct any conversion issues

2. **Inconsistent API interfaces:**
   - Create a web service wrapper to normalize interfaces
   - Example implementations available in [Java](https://github.com/krateoplatformops/azuredevops-oas3-plugin) and [Python](https://github.com/krateoplatformops/github-oas3-plugin)

3. **Authentication issues:**
   - Verify secret references in BasicAuth CRs
   - Check network connectivity to API endpoints
   - Enable verbose logging with annotations (see below)

### Debug Mode

To troubleshoot Krateo controllers, add the annotation `krateo.io/connector-verbose: "true"` to the CR you want to debug. This enables verbose logging for the controller and helps you understand what's happening under the hood.

**Examples:**
- Adding this annotation to a `RestDefinition` will help you understand what happens when the controller tries to create the CRD and controller for the RestDefinition
- Adding it to a `Repo` or `TeamRepo` CR will help you understand what happens when the controller tries to create, update, or delete the external resource, and will log any HTTP requests and responses made to the external API


