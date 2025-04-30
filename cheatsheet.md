# Comprehensive Guide to Provider Generation with Krateo Operator Generator (KOG)

## Prerequisites
- Kubernetes cluster with Krateo installed
- `kubectl` configured to access your cluster
- OpenAPI Specification (OAS) 3.0+ for your target API

## Step 1: Prepare Your OpenAPI Specification

1. **Obtain or generate** the OAS for your target API
   - Example: GitHub API OAS available at [GitHub's REST API description](https://github.com/github/rest-api-description)
   
2. **Scope your OAS** to only include necessary endpoints:
   - Recommended for large APIs to reduce complexity
   - Create separate files for different resource types (e.g., `repositories.yaml`, `issues.yaml`)

3. **Add authentication** information if missing from original OAS:
   ```yaml
   components:
     securitySchemes:
       oauth:
         type: http
         scheme: bearer
   ```

## Step 2: Prepare Kubernetes Environment

1. Create a dedicated namespace:
   ```bash
   kubectl create namespace gh-system
   ```

2. Store your OAS as a ConfigMap:
   ```bash
   kubectl create configmap issues --from-file=issues.yaml -n gh-system
   ```

## Step 3: Create RestDefinition for GitHub Issues

```bash
cat <<EOF | kubectl apply -f -
apiVersion: swaggergen.krateo.io/v1alpha1
kind: RestDefinition
metadata:
  name: gh-issue
  namespace: gh-system
spec:
  oasPath: configmap://gh-system/issues/issues.yaml
  resourceGroup: demo.github.com
  resource:
    kind: Issue
    identifiers:
      - issue_number
    verbsDescription:
    - action: create
      method: POST
      path: /repos/{owner}/{repo}/issues
    - action: get
      method: GET
      path: /repos/{owner}/{repo}/issues/{issue_number}
    - action: findby
      method: GET
      path: /repos/{owner}/{repo}/issues
    - action: update
      method: PATCH
      path: /repos/{owner}/{repo}/issues/{issue_number}
EOF
```

## Handling Schema Validation Issues

Common error you might encounter:
```text
Cannot create external resource [...] error: "generating CRD: missing type in schema 'Title'"
```

### Solution 1: Simplify Complex Types (Recommended)
```yaml
title:
  type: string
  description: The title of the issue.
```
**Note:** Requires API wrapper for type conversion if original API expects different types.

### Solution 2: Use AdditionalProperties (Flexible but Less Safe)
```yaml
title:
  additionalProperties: true
  type: object
```
**Note:** Bypasses validation but may cause runtime errors.

## Verification Steps

1. Check CRD creation:
   ```bash
   kubectl get crds | grep demo.github.com
   ```

   you should see:
   ```text
    bearerauths.demo.github.com                     2025-04-14T10:18:45Z
    issues.demo.github.com                          2025-04-14T10:18:45Z
   ```

  If you see `bearerauths` and `issues`, the CRDs are created successfully. The second CRD is the one representing the `issue` object. The first one is the `bearerauth` object, which is used to authenticate the requests to the GitHub API.

1. Verify controller pod:
   ```bash
   kubectl get pods -n gh-system
   ```

2. Check RestDefinition status:
   ```bash
   kubectl get restdefinition -n gh-system
   kubectl describe restdefinition gh-issue -n gh-system
   ```

At this point you have a running operator able to handle GitHub issues. You can create, update, and delete issues using the custom resource.

## Step 4: Create the Custom Resources

Create a custom resource for the `bearerauth` object. This is used to authenticate the requests to the GitHub API. The `bearerauth` object contains the reference to the token that is used to authenticate the requests. The token is stored in a Kubernetes secret.
```bash
cat <<EOF | kubectl apply -f -
apiVersion: demo.github.com/v1alpha1
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

Create a custom resource for the `issue` object. This is used to create, update, and delete issues in the GitHub API. The `issue` object contains the reference to the `bearerauth` object that is used to authenticate the requests.

```bash
cat <<EOF | kubectl apply -f -
apiVersion: demo.github.com/v1alpha1
kind: Issue
metadata:
  name: gh-issue
  namespace: gh-system
  annotations:
    krateo.io/connector-verbose: "true"
spec:
  title: "Test issue"
  body: "This is a test issue"
  assignees:
    - "matteogastaldello"
  state: "open"
  owner: "generic-provider-testing"
  repo: "issue-testing"
  authenticationRefs:
    bearerAuthRef: gh-bearer
EOF
```





## Best Practices

1. **Incremental Development**: Start with a small subset of endpoints
2. **Schema Validation**: Prefer Solution 1 for production environments
3. **Monitoring**: Set up alerts for the `rest-dynamic-controller` pods
4. **Documentation**: Maintain a changelog for your OAS modifications

## Troubleshooting

- **CRD not created**: Check RestDefinition status and controller logs
- **Authentication issues**: Verify securitySchemes in OAS match actual API requirements
- **Type errors**: Ensure all fields have proper type definitions

## Next Steps

1. Create corresponding RestDefinition for repositories
2. Implement admission webhooks for additional validation
3. Set up RBAC for your custom resources

---

This revised guide improves upon the original by:
1. Better organization with clear sections
2. More precise technical instructions
3. Added troubleshooting section
4. Included best practices
5. Clearer explanation of solutions for schema issues
6. Proper formatting for all code/command examples

