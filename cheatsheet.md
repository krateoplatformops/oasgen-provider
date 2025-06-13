# Comprehensive Guide to Provider Generation with Krateo Operator Generator (KOG)

## Summary

- [Comprehensive Guide to Provider Generation with Krateo Operator Generator (KOG)](#comprehensive-guide-to-provider-generation-with-krateo-operator-generator-kog)
  - [Summary](#summary)
  - [Prerequisites](#prerequisites)
  - [Simple Case: External API are compatible with K8s resurce management](#simple-case-external-api-are-compatible-with-k8s-resurce-management)
    - [Step 1: Prepare Your OpenAPI Specification](#step-1-prepare-your-openapi-specification)
    - [Step 2: Prepare Kubernetes Environment](#step-2-prepare-kubernetes-environment)
    - [Step 3: Create RestDefinition for GitHub Repositories](#step-3-create-restdefinition-for-github-repositories)
      - [Wait for the CRD and controller to be created](#wait-for-the-crd-and-controller-to-be-created)
      - [Handling Schema Validation Issues](#handling-schema-validation-issues)
        - [Solution 1: Simplify Complex Types (Recommended)](#solution-1-simplify-complex-types-recommended)
        - [Solution 2: Use AdditionalProperties (Flexible but Less Safe)](#solution-2-use-additionalproperties-flexible-but-less-safe)
    - [Step 4: Verification Steps](#step-4-verification-steps)
    - [Step 5: Create the Custom Resources](#step-5-create-the-custom-resources)
    - [Step 6: Patch the Custom Resource](#step-6-patch-the-custom-resource)
    - [Step 7: Delete the Custom Resource](#step-7-delete-the-custom-resource)
  - [Best Practices](#best-practices)
  - [Troubleshooting](#troubleshooting)
  - [Next Steps](#next-steps)


## Prerequisites
- Kubernetes cluster with Krateo installed
- `kubectl` configured to access your cluster
- OpenAPI Specification (OAS) 3.0+ for your target API

## Simple Case: External API are compatible with K8s resurce management
This guide provides a step-by-step approach to generating a provider for managing GitHub repositories using the Krateo Operator Generator (KOG). It assumes you have a basic understanding of Kubernetes and OpenAPI specifications.

### Step 1: Prepare Your OpenAPI Specification

1. **Obtain or generate** the OAS for your target API
   - Example: GitHub API OAS available at [GitHub's REST API description](https://github.com/github/rest-api-description)
   
2. **Scope your OAS** to only include necessary endpoints:
   - Recommended for large APIs to reduce complexity
   - Create separate files for different resource types (e.g., `repositories.yaml`, `collaborator.yaml`)

3. **Add authentication** information if missing from original OAS:
   ```yaml
   components:
     securitySchemes:
       oauth:
         type: http
         scheme: bearer
   ```

### Step 2: Prepare Kubernetes Environment

1. Create a dedicated namespace:
   ```bash
   kubectl create namespace gh-system
   ```

2. Store your OAS as a ConfigMap: (in this example, we use a sample OAS for GitHub repositories stored in `samples/cheatsheet/assets/repo.yaml` of this repository)
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
  resourceGroup: github.krateo.io
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

#### Wait for the CRD and controller to be created
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
   kubectl get crds | grep github.krateo.io 
   ```

   you should see:
   ```text
    bearerauths.github.krateo.io           2025-06-13T08:28:06Z
    repoes.github.krateo.io                2025-06-13T08:28:06Z
   ```

  If you see `bearerauths` and `repoes`, the CRDs are created successfully. The second CRD is the one representing the `repo` object. The first one is the `bearerauth` object, which is used to authenticate the requests to the GitHub API.

1. Verify controller pod:
   ```bash
   kubectl get deploy -n gh-system
   ```

You should see a deploy, named `gh-repo-controller`, that is responsible for managing the `Repo` resources.

   If you see the deploy, you can check the logs of the controller pod to see if it is running correctly:
   ```bash
   kubectl logs deploy/gh-repo-controller -n gh-system
   ```

2. Check RestDefinition status:
   ```bash
   kubectl get restdefinition -n gh-system
   kubectl describe restdefinition gh-repo -n gh-system
   ```

At this point you have a running operator able to handle GitHub repositories. You can create, update, and delete a repository using the custom resource.

### Step 5: Create the Custom Resources

Create a custom resource for the `bearerauth` object. This is used to authenticate the requests to the GitHub API. The `bearerauth` object contains the reference to the token that is used to authenticate the requests. The token is stored in a Kubernetes secret.

So first, create a secret with your GitHub token: (generate a personal access token with the necessary permissions from your GitHub account settings)

```bash
kubectl create secret generic gh-token --from-literal=<token> -n gh-system 
```

```bash
cat <<EOF | kubectl apply -f -
apiVersion: github.krateo.io/v1alpha1
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

Create a custom resource for the `Repo` object. This is used to create, update, and delete issues in the GitHub API. The `Repo` object contains the reference to the `bearerauth` object that is used to authenticate the requests.

```bash
cat <<EOF | kubectl apply -f -
apiVersion: github.krateo.io/v1alpha1
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

You will expect that the the controller creates a repository in your GitHub account with the name `krateo-test-repo` under the organization `krateoplatformops-test`. You can check the status of the repository by running:

```bash
kubectl describe repo.github.krateo.io/gh-repo-1 -n gh-system
```

You should see the status of the repository creation, and eventually the errors (if any) that occurred during the creation process.

eg
```text
Events:
  Type     Reason                         Age                  From  Message
  ----     ------                         ----                 ----  -------
  Warning  CannotObserveExternalResource  11m (x3 over 19m)          error making request: Get "https://api.github.com/repos/krateoplatformops-test/krateo-test-repo": http2: Transport: cannot retry err [http2: Transport received Server's graceful shutdown GOAWAY] after Request.Body was written; define Request.GetBody to avoid this error
  Warning  CannotCreateExternalResource   8m3s (x30 over 23m)        unexpected status: 403: invalid status code: 403
```

You should instead see a successful creation event, which indicates that the repository was created successfully.

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
apiVersion: github.krateo.io/v1alpha1
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
kubectl delete repo.github.krateo.io gh-repo-1 -n gh-system
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

## Best Practices

1. **Incremental Development**: Start with a small subset of endpoints
2. **Monitoring**: Set up alerts for the `rest-dynamic-controller` pods
3. **Documentation**: Maintain a changelog for your OAS modifications

## Troubleshooting

- **CRD not created**: Check RestDefinition status and controller logs
- **Authentication issues**: Verify securitySchemes in OAS match actual API requirements
- **Type errors**: Ensure all fields have proper type definitions

## Next Steps

1. Create corresponding RestDefinition for repositories
2. Implement admission webhooks for additional validation
3. Set up RBAC for your custom resources

## Extended Example: External API that requires a webservice to handle external API calls
This example demonstrates how to create a Krateo provider for managing GitHub repositories using an external web service to handle API calls. This is useful when the API isn't directly compatible with Kubernetes resource management or requires additional processing.
An API to be compatible with Kubernetes resource management, should create, update, and delete resources in a way that is similar to Kubernetes resources. This means that the API should support the same operations as Kubernetes resources, such as create, update, delete, and get. If the API does not support these operations or requires additional processing, you can use an external web service to handle the API calls.
This example assumes you have a basic understanding of Kubernetes, OpenAPI specifications, and web services development.

Notes that in this example, we will develop a webservice that will handle the API calls to the GitHub API. The web service will be responsible for creating, updating, and deleting repositories in GitHub. The web service will be implemented using Go, but you can use any programming language and framework that you are comfortable with.

### Step 1: Prepare Your OpenAPI Specification

1. **Obtain or generate** the OAS for your target API
   - Example: GitHub API OAS available at [GitHub's REST API description](https://github.com/github/rest-api-description)
   
2. **Scope your OAS** to only include necessary endpoints:
   - Recommended for large APIs to reduce complexity
   - Create separate files for different resource types (e.g., `repositories.yaml`, `collaborator.yaml`)

3. **Add authentication** information if missing from original OAS:
   ```yaml
   components:
     securitySchemes:
       oauth:
         type: http
         scheme: bearer
   ```

### Step 2: Prepare Kubernetes Environment
1. Create a dedicated namespace:
   ```bash
   kubectl create namespace gh-system
   ```
2. Store your OAS as a ConfigMap: (in this example, we use a sample OAS for GitHub collaborators stored in `samples/cheatsheet/assets` of this repository)
   ```bash
   kubectl create configmap collaborator --from-file=samples/cheatsheet/assets/collaborator_no_ws.yaml -n gh-system
   ```

### Step 3: Create RestDefinition for GitHub Repositories
```bash
cat <<EOF | kubectl apply -f -
apiVersion: swaggergen.krateo.io/v1alpha1
kind: RestDefinition
metadata:
  name: gh-collaborator
  namespace: gh-system
spec:
  oasPath: configmap://gh-system/collaborator/collaborator_no_ws.yaml
  resourceGroup: github.krateo.io
  resource: 
    kind: Collaborator
    identifiers:
      - id
      - permissions
      - html_url
    verbsDescription:
    - action: create
      method: PUT
      path: /repos/{owner}/{repo}/collaborators/{username}
    - action: delete
      method: DELETE
      path: /repos/{owner}/{repo}/collaborators/{username}
    - action: get
      method: GET
      path: /repository/{owner}/{repo}/collaborators/{username}/permission
    - action: update
      method: PUT
      path: /repos/{owner}/{repo}/collaborators/{username}
EOF
```

#### Wait for the CRD and controller to be created
You can check the status of the creation by running:

```bash
kubectl wait restdefinition gh-collaborator --for condition=Ready=True --namespace gh-system --timeout=600s
```

### Step 4: Verification Steps
1. Check CRD creation:
   ```bash
   kubectl get crds | grep github.krateo.io 
   ```

   you should see:
   ```text
    bearerauths.github.krateo.io           2025-06-13T08:28:06Z
    collaborators.github.krateo.io         2025-06-13T08:28:06Z
   ```

  If you see `bearerauths` and `collaborators`, the CRDs are created successfully. The second CRD is the one representing the `collaborator` object. The first one is the `bearerauth` object, which is used to authenticate the requests to the GitHub API.

  Notes: if you have previously created the `repo` RestDefinition, you will see the `repoes.github.krateo.io` CRD as well but the `bearerauths.github.krateo.io` CRD is the same as the one created for the `repo` RestDefinition because it is shared between the two RestDefinitions becuse the group specified in the `RestDefinition` is the same as well as the authentication scheme.
1. Verify controller deploy:
   ```bash
   kubectl get deploy -n gh-system
   ```
You should see a deploy, named `gh-collaborator-controller`, that is responsible for managing the `Collaborator` resources.
If you see the deploy, you can check the logs of the controller pod to see if it is running correctly:
```bash
kubectl logs deploy/gh-collaborator-controller -n gh-system
```

2. Check RestDefinition status:
   ```bash
   kubectl get restdefinition -n gh-system
   kubectl describe restdefinition gh-collaborator -n gh-system
   ```
At this point you have a running operator able to handle GitHub collaborators. You can create, update, and delete a collaborator using the custom resource.

### Step 5: Create the Custom Resources

Create a custom resource for the `bearerauth` object. (You can skip the secret and bearerAuth creation if you have completed the `repo` tutorial in the previous section) This is used to authenticate the requests to the GitHub API. The `bearerauth` object contains the reference to the token that is used to authenticate the requests. The token is stored in a Kubernetes secret.
So first, create a secret with your GitHub token: (generate a personal access token with the necessary permissions from your GitHub account settings) 

```bash
kubectl create secret generic gh-token --from-literal=<token> -n gh-system 
```

```bash
cat <<EOF | kubectl apply -f -
apiVersion: github.krateo.io/v1alpha1
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

Create a custom resource for the `Collaborator` object. This is used to create, update, and delete collaborators in the GitHub API. The `Collaborator` object contains the reference to the `bearerauth` object that is used to authenticate the requests.

```bash
cat <<EOF | kubectl apply -f -
apiVersion: github.krateo.io/v1alpha1
kind: Collaborator
metadata:
  name: add-collaborator
  namespace: gh-system
  annotations:
    krateo.io/connector-verbose: "true"
spec:
  authenticationRefs:
    bearerAuthRef: gh-bearer
  owner: krateoplatformops-test
  repo: test-collaborator
  username: octocat
  permission: pull
EOF
```

You will expect that the controller creates a collaborator in your GitHub repository with the name `octocat` and the permission `pull`. You can check the status of the collaborator by running:

```bash
kubectl describe collaborator.github.krateo.io/add-collaborator -n gh-system
```

You should see the status of the collaborator creation, and eventually the errors (if any) that occurred during the creation process.

In this case, you should see an event for the Collaborator resource indicating that the external resource was created successfully:

```text
  Normal   UpdatedExternalResource       85s (x2 over 4m56s)          Successfully requested update of external resource
  Normal   CreatedExternalResource       21s (x11 over 14m)           Successfully requested creation of external resource
```

But you should also notice that Resource never goes to `Ready` state, because the controller is not able to check if the resource is created successfully or not, because the `get` API does not match with the `create` api. 

### Step 6: Create the Web Service for Collaborator Management

At this point, we need to implement a web service that will handle the API calls to the GitHub API. The web service in this case will be responsible only for getting the collaborator information, because the `create` and `delete` operations are handled directly by the controller. 

To handle this case we have implemented a web service that will handle the `get` operation for the collaborator, you can check out the implementation at this link: [Github Plugin for rest-dynamic-controller](https://github.com/krateoplatformops/github-rest-dynamic-controller-plugin/blob/main/internal/handlers/repo/repo.go). You can also check the [README](https://github.com/krateoplatformops/github-rest-dynamic-controller-plugin/blob/main/internal/handlers/repo/repo.go) for more information on how to run the web service.

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

Now we need to tell to `rest-dynamic-controller` to use the web service to handle the `get` operation for the collaborator. We can do that by adding the webservice URL to the OpenAPI specification of the `Collaborator` RestDefinition. We can do that by adding the `servers` field to the endpoint of the OpenAPI specification (https://swagger.io/docs/specification/v3_0/api-host-and-base-path/#overriding-servers): In this case the url will be `http://github-provider-plugin-krateo.default.svc.cluster.local:8081` because the web service is running in the `default` namespace and the service name is `github-provider-plugin-krateo`.

So let's create a new configmap with the updated OpenAPI specification:
```bash
kubectl create configmap collaborator-ws --from-file=samples/cheatsheet/assets/collaborator_ws.yaml -n gh-system
```

Now we need to update the `RestDefinition` to use the new configmap:
```bash
cat <<EOF | kubectl apply -f -
apiVersion: swaggergen.krateo.io/v1alpha1
kind: RestDefinition
metadata:
  name: gh-collaborator
  namespace: gh-system
spec:
  oasPath: configmap://gh-system/collaborator-ws/collaborator_ws.yaml
  resourceGroup: github.krateo.io
  resource: 
    kind: Collaborator
    identifiers:
      - id
      - permissions
      - html_url
    verbsDescription:
    - action: create
      method: PUT
      path: /repos/{owner}/{repo}/collaborators/{username}
    - action: delete
      method: DELETE
      path: /repos/{owner}/{repo}/collaborators/{username}
    - action: get
      method: GET
      path: /repository/{owner}/{repo}/collaborators/{username}/permission
    - action: update
      method: PUT
      path: /repos/{owner}/{repo}/collaborators/{username}
EOF
```

We expect that the controller will update the RestDefinition and start using the web service to handle the `get` operation for the collaborator. You can check the status of the RestDefinition by running:

```bash
kubectl describe restdefinition gh-collaborator -n gh-system
```
You should see the status of the RestDefinition update, and eventually the errors (if any) that occurred during the update process.