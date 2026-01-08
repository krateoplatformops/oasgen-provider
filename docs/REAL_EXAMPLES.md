# Real-world Examples of RestDefinition Manifests

This document provides real-world examples of `RestDefinition` manifests used with the OASGen Provider. These examples illustrate various use cases and configurations to help users understand how to create their own `RestDefinition` resources effectively and leverage all the features provided by the OASGen Provider and the Rest Dynamic Controller.

## `requestFieldMapping` Example 1

Example showing how to use `requestFieldMapping` to map fields from the Custom Resource to different locations in the HTTP request (query parameters, path parameters, request body). 
This is particularly useful when the field names in the OpenAPI Specification differ from those in the Custom Resource or when fields need to be placed in specific parts of the request.

**Resource**: ArubaCloud Subnet
**Full manifest**:
**RestDefinition manifest (partial)**:
```yaml
kind: RestDefinition
apiVersion: ogen.krateo.io/v1alpha1
metadata:
  name: {{ .Release.Name }}-subnet
spec:
  oasPath: configmap://{{ .Release.Namespace }}/{{ .Release.Name }}-subnet/subnet.yaml
  resourceGroup: arubacloud.ogen.krateo.io
  resource: 
    kind: Subnet
    identifiers:
      - metadata.name
    additionalStatusFields:
      - metadata.id
    excludedSpecFields:
      - id
    verbsDescription:
    - action: findby
      method: GET
      path: /projects/{projectId}/providers/Aruba.Network/vpcs/{vpcId}/subnets
    - action: get
      method: GET
      path: /projects/{projectId}/providers/Aruba.Network/vpcs/{vpcId}/subnets/{id}
      requestFieldMapping:
        - inPath: id
          inCustomResource: status.metadata.id
    - action: create
      method: POST
      path: /projects/{projectId}/providers/Aruba.Network/vpcs/{vpcId}/subnets
    - action: update
      method: PUT
      path: /projects/{projectId}/providers/Aruba.Network/vpcs/{vpcId}/subnets/{id}
      requestFieldMapping:
        - inPath: id
          inCustomResource: status.metadata.id
    - action: delete
      method: DELETE
      path: /projects/{projectId}/providers/Aruba.Network/vpcs/{vpcId}/subnets/{id}
      requestFieldMapping:
        - inPath: id
          inCustomResource: status.metadata.id
    configurationFields:
    - fromOpenAPI:
        name: api-version
        in: query
      fromRestDefinition:
        actions: ["*"] # star means all actions set in the verbsDescription above
    - fromOpenAPI:
        name: filter
        in: query
      fromRestDefinition:
        actions:
          - findby
    ...
```

**Description**:
In this example, the `requestFieldMapping` is used in the `get`, `update`, and `delete` actions to map the `id` path parameter from the Custom Resource's `status.metadata.id` field. This allows the controller to correctly populate the path parameter when making requests to the ArubaCloud API, even though the field is nested.
Indeed, the OASGen Provider at CRD generation time would add, among others, all the path parameters defined in the endpoints listed in the RestDefinition to the spec of the CRD. In this case, the `id` path parameter would be added to the spec.
Without `requestFieldMapping` setup, Rest Dynamic Controller would look for an `id` (since the path parameter is `id`) field directly in the spec or status, which does not exist. 
The field we are looking for is actually `metadata.id` inside the `status` section of the Custom Resource. The field is there because it is listed under `additionalStatusFields` in the RestDefinition.
Therefore, we need to use `requestFieldMapping` to tell the controller where to find the value for the `id` path parameter in the Custom Resource since it is not directly available as `id` in the spec or status due to the nested structure.

## `requestFieldMapping` Example 2 + dot escape in `excludedSpecFields`

Example showing how to use `requestFieldMapping` to map fields from the Custom Resource to different locations in the HTTP request (query parameters, path parameters, request body) and how to escape special characters in `excludedSpecFields` for fields with names including dots (edge case).

**Resource**: Azure DevOps PullRequest
**Full manifest**:
**RestDefinition manifest (partial)**:
```yaml
kind: RestDefinition
apiVersion: ogen.krateo.io/v1alpha1
metadata:
  name: {{ .Release.Name }}-pullrequest
spec:
  oasPath: configmap://{{ .Release.Namespace }}/{{ .Release.Name }}-pullrequest/pullrequest.yaml
  resourceGroup: azuredevops.ogen.krateo.io
  resource:
    kind: PullRequest
    identifiers:
    - title # Title is enough since the `findby` action supports query filtering
    additionalStatusFields:
    - closedBy
    - closedDate
    ...
    excludedSpecFields:
    - pullRequestId
    - maxCommentLength
    - $skip        
    - $top
    - completionOptions.triggeredByAutoComplete
    - '["searchCriteria.creatorId"]'
    - '["searchCriteria.includeLinks"]'
    - '["searchCriteria.maxTime"]'
    - '["searchCriteria.minTime"]'
    - '["searchCriteria.queryTimeRangeType"]'
    - '["searchCriteria.repositoryId"]'
    - '["searchCriteria.reviewerId"]'
    - '["searchCriteria.sourceRefName"]'
    - '["searchCriteria.sourceRepositoryId"]'
    - '["searchCriteria.status"]'
    - '["searchCriteria.targetRefName"]'
    - '["searchCriteria.title"]'
    ...
    verbsDescription:
    - action: findby
      method: GET
      path: /{organization}/{project}/_apis/git/repositories/{repositoryId}/pullrequests
      requestFieldMapping:
        - inQuery: '["searchCriteria.sourceRefName"]'
          inCustomResource: spec.sourceRefName
        - inQuery: '["searchCriteria.targetRefName"]'
          inCustomResource: spec.targetRefName
        - inQuery: '["searchCriteria.title"]'
          inCustomResource: spec.title
    - action: get
      method: GET
      path: /{organization}/{project}/_apis/git/repositories/{repositoryId}/pullrequests/{pullRequestId}
    - action: create
      method: POST
      path: /{organization}/{project}/_apis/git/repositories/{repositoryId}/pullrequests
    - action: update
      method: PATCH
      path: /api/{organization}/{project}/git/repositories/{repositoryId}/pullrequests/{pullRequestId} # Plugin endpoint
      requestFieldMapping:
        - inBody: lastMergeSourceCommit # this is needed otherwise the update fails in case of status change to `completed`
          inCustomResource: status.lastMergeSourceCommit
    # DELETE method is not supported for Pull Requests in Azure DevOps, set the status field of the spec to "abandoned" instead
    configurationFields:
    # Common among all actions
    - fromOpenAPI:
        name: api-version
        in: query
      fromRestDefinition:
        actions: ["*"]
    # Get
    - fromOpenAPI:
        name: includeCommits
        in: query
      fromRestDefinition:
        actions: 
          - get
    - fromOpenAPI:
        name: includeWorkItemRefs
        in: query
      fromRestDefinition:
        actions: 
          - get
```

**Description**:
In this example, the `requestFieldMapping` is used in both the `findby` and `update` actions to map fields from the Custom Resource to the appropriate locations in the HTTP request. Additionally, special characters in field names are properly escaped in the `excludedSpecFields` section to ensure correct parsing. The notation with brackets and quotes is used to escape the dots in field names.
Note the difference between the following:
```yaml
    - completionOptions.triggeredByAutoComplete
    - '["searchCriteria.creatorId"]'
```
The first line does not need escaping since `completionOptions` is an object and `triggeredByAutoComplete` is one of its fields.
The second line with `searchCriteria.creatorId` needs escaping since `searchCriteria` is not an object with field `creatorId`, but rather a single field name including a dot (unusual but possible).
Note that in `excludedSpecFields`, we list some fields that are actually mapped via `requestFieldMapping`. This is because we want to avoid having these fields in the spec which would clutter it with duplicated information. Instead we want to use the already existing fields in the spec and map them to the appropriate locations in the request.
Therefore `requestFieldMapping` and `excludedSpecFields` can be used together to achieve the desired mapping while keeping the spec clean from redundant fields.

## `identifiersMatchPolicy` with AND

Example showing how to use `identifiersMatchPolicy` set to `AND` for the `findby` action to require multiple fields to match when identifying a resource.

**Resource**: Azure DevOps PolicyConfiguration
**Full manifest**:
**RestDefinition manifest (partial)**:
```yaml
kind: RestDefinition
apiVersion: ogen.krateo.io/v1alpha1
metadata:
  name: {{ .Release.Name }}-policyconfiguration
spec:
  oasPath: configmap://{{ .Release.Namespace }}/{{ .Release.Name }}-policyconfiguration/policyconfiguration.yaml
  resourceGroup: azuredevops.ogen.krateo.io
  resource:
    kind: PolicyConfiguration
    identifiers:
    - type.id
    - settings
    additionalStatusFields:
    - id
    excludedSpecFields:
    - id
    - scope # legacy, deprecated field. `scope` in `settings` should be used instead
    verbsDescription:
    - action: findby
      method: GET
      path: /{organization}/{project}/_apis/policy/configurations
      identifiersMatchPolicy: AND
    - action: get
      method: GET
      path: /{organization}/{project}/_apis/policy/configurations/{id}
    - action: create
      method: POST
      path: /{organization}/{project}/_apis/policy/configurations
    - action: update
      method: PUT
      path: /{organization}/{project}/_apis/policy/configurations/{id}
    - action: delete
      method: DELETE
      path: /{organization}/{project}/_apis/policy/configurations/{id}
    configurationFields:
    # Common among all actions
    - fromOpenAPI:
        name: api-version
        in: query
      fromRestDefinition:
        actions: ["*"]
    # FindBy
    - fromOpenAPI:
        name: $top
        in: query
      fromRestDefinition:
        actions:
          - findby
    - fromOpenAPI:
        name: continuationToken
        in: query
      fromRestDefinition:
        actions:
          - findby
    - fromOpenAPI:
        name: policyType
        in: query
      fromRestDefinition:
        actions:
          - findby

```

**Description**:
In this example, the `identifiersMatchPolicy` is set to `AND` for the `findby` action. This means that when searching for a `PolicyConfiguration` resource, both the `type.id` and `settings` fields must match the corresponding fields in the API response for a resource to be considered a match. This is useful when multiple fields are required to uniquely identify a resource. The default behavior is `OR`, which would consider a match if any of the specified identifiers match but in this particular case both identifiers are necessary to uniquely identify the resource.
