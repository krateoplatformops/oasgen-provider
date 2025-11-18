# API Reference

Packages:

- [ogen.krateo.io/v1alpha1](#ogenkrateoiov1alpha1)

# ogen.krateo.io/v1alpha1

Resource Types:

- [RestDefinition](#restdefinition)




## RestDefinition
<sup><sup>[↩ Parent](#ogenkrateoiov1alpha1 )</sup></sup>






RestDefinition is a RestDefinition type with a spec and a status.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>ogen.krateo.io/v1alpha1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>RestDefinition</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#restdefinitionspec">spec</a></b></td>
        <td>object</td>
        <td>
          RestDefinitionSpec is the specification of a RestDefinition.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#restdefinitionstatus">status</a></b></td>
        <td>object</td>
        <td>
          RestDefinitionStatus is the status of a RestDefinition.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### RestDefinition.spec
<sup><sup>[↩ Parent](#restdefinition)</sup></sup>



RestDefinitionSpec is the specification of a RestDefinition.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>oasPath</b></td>
        <td>string</td>
        <td>
          Path to the OpenAPI specification. This value can change over time, for example if the OAS file is updated but be sure to not change the requestbody of the `create` verb.
- configmap://<namespace>/<name>/<key>
- http(s)://<url><br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#restdefinitionspecresource">resource</a></b></td>
        <td>object</td>
        <td>
          The resource to manage<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>resourceGroup</b></td>
        <td>string</td>
        <td>
          Group: the group of the resource to manage<br/>
          <br/>
            <i>Validations</i>:<li>self == oldSelf: ResourceGroup is immutable, you cannot change that once the CRD has been generated</li>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### RestDefinition.spec.resource
<sup><sup>[↩ Parent](#restdefinitionspec)</sup></sup>



The resource to manage

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>kind</b></td>
        <td>string</td>
        <td>
          Name: the name of the resource to manage<br/>
          <br/>
            <i>Validations</i>:<li>self == oldSelf: Kind is immutable, you cannot change that once the CRD has been generated</li>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#restdefinitionspecresourceverbsdescriptionindex">verbsDescription</a></b></td>
        <td>[]object</td>
        <td>
          VerbsDescription: the list of verbs to use on this resource<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>additionalStatusFields</b></td>
        <td>[]string</td>
        <td>
          AdditionalStatusFields: the list of fields to use as additional status fields - used to populate the status of the resource<br/>
          <br/>
            <i>Validations</i>:<li>self == oldSelf: AdditionalStatusFields are immutable, you cannot change them once the CRD has been generated</li>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#restdefinitionspecresourceconfigurationfieldsindex">configurationFields</a></b></td>
        <td>[]object</td>
        <td>
          ConfigurationFields: the list of fields to use as configuration fields<br/>
          <br/>
            <i>Validations</i>:<li>self == oldSelf: ConfigurationFields are immutable, you cannot change them once the CRD has been generated</li>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>excludedSpecFields</b></td>
        <td>[]string</td>
        <td>
          ExcludedSpecFields: the list of fields to exclude from the spec of the generated CRD (for example server-generated technical IDs could be excluded)<br/>
          <br/>
            <i>Validations</i>:<li>self == oldSelf: ExcludedSpecFields are immutable, you cannot change them once the CRD has been generated</li>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>identifiers</b></td>
        <td>[]string</td>
        <td>
          Identifiers: the list of fields to use as identifiers - used to populate the status of the resource<br/>
          <br/>
            <i>Validations</i>:<li>self == oldSelf: Identifiers are immutable, you cannot change them once the CRD has been generated</li>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### RestDefinition.spec.resource.verbsDescription[index]
<sup><sup>[↩ Parent](#restdefinitionspecresource)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>action</b></td>
        <td>enum</td>
        <td>
          Name of the action to perform when this api is called [create, update, get, delete, findby]<br/>
          <br/>
            <i>Enum</i>: create, update, get, delete, findby<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>method</b></td>
        <td>enum</td>
        <td>
          Method: the http method to use [GET, POST, PUT, DELETE, PATCH]<br/>
          <br/>
            <i>Enum</i>: GET, POST, PUT, DELETE, PATCH<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>path</b></td>
        <td>string</td>
        <td>
          Path: the path to the api - has to be the same path as the one in the OAS file you are referencing<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>identifiersMatchPolicy</b></td>
        <td>enum</td>
        <td>
          IdentifiersMatchPolicy defines how to match identifiers for the 'findby' action. To be set only for 'findby' actions.
If not set, defaults to 'OR'.
Possible values are 'AND' or 'OR'.
- 'AND': all identifiers must match.
- 'OR': at least one identifier must match (the default behavior).<br/>
          <br/>
            <i>Enum</i>: AND, OR<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#restdefinitionspecresourceverbsdescriptionindexrequestfieldmappingindex">requestFieldMapping</a></b></td>
        <td>[]object</td>
        <td>
          RequestFieldMapping provides explicit mapping from API parameters (path, query, or body)
to fields in the Custom Resource.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### RestDefinition.spec.resource.verbsDescription[index].requestFieldMapping[index]
<sup><sup>[↩ Parent](#restdefinitionspecresourceverbsdescriptionindex)</sup></sup>



RequestFieldMappingItem defines a single mapping from a path parameter, query parameter or body field
to a field in the Custom Resource.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>inCustomResource</b></td>
        <td>string</td>
        <td>
          InCustomResource defines the JSONPath to the field within the Custom Resource that holds the value.
For example: 'spec.name' or 'status.metadata.id'.
Note: potentially we could add validation to ensure this is a valid path (e.g., starts with 'spec.' or 'status.').
Currently, no validation is enforced on the content of this field.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>inBody</b></td>
        <td>string</td>
        <td>
          InBody defines the name of the body parameter to be mapped.
Only one of 'inPath', 'inQuery' or 'inBody' can be set.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>inPath</b></td>
        <td>string</td>
        <td>
          InPath defines the name of the path parameter to be mapped.
Only one of 'inPath', 'inQuery' or 'inBody' can be set.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>inQuery</b></td>
        <td>string</td>
        <td>
          InQuery defines the name of the query parameter to be mapped.
Only one of 'inPath', 'inQuery' or 'inBody' can be set.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### RestDefinition.spec.resource.configurationFields[index]
<sup><sup>[↩ Parent](#restdefinitionspecresource)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#restdefinitionspecresourceconfigurationfieldsindexfromopenapi">fromOpenAPI</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#restdefinitionspecresourceconfigurationfieldsindexfromrestdefinition">fromRestDefinition</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### RestDefinition.spec.resource.configurationFields[index].fromOpenAPI
<sup><sup>[↩ Parent](#restdefinitionspecresourceconfigurationfieldsindex)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>in</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### RestDefinition.spec.resource.configurationFields[index].fromRestDefinition
<sup><sup>[↩ Parent](#restdefinitionspecresourceconfigurationfieldsindex)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>actions</b></td>
        <td>[]string</td>
        <td>
          Actions: the list of actions this configuration applies to. Use ["*"] to apply to all actions.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### RestDefinition.status
<sup><sup>[↩ Parent](#restdefinition)</sup></sup>



RestDefinitionStatus is the status of a RestDefinition.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>oasPath</b></td>
        <td>string</td>
        <td>
          OASPath: the path to the OAS Specification file.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#restdefinitionstatusconditionsindex">conditions</a></b></td>
        <td>[]object</td>
        <td>
          Conditions of the resource.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#restdefinitionstatusconfiguration">configuration</a></b></td>
        <td>object</td>
        <td>
          Configuration: the configuration of the resource<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>digest</b></td>
        <td>string</td>
        <td>
          Digest: the digest of the managed resources<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#restdefinitionstatusresource">resource</a></b></td>
        <td>object</td>
        <td>
          Resource: the resource to manage<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### RestDefinition.status.conditions[index]
<sup><sup>[↩ Parent](#restdefinitionstatus)</sup></sup>



A Condition that may apply to a resource.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>lastTransitionTime</b></td>
        <td>string</td>
        <td>
          LastTransitionTime is the last time this condition transitioned from one
status to another.<br/>
          <br/>
            <i>Format</i>: date-time<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>reason</b></td>
        <td>string</td>
        <td>
          A Reason for this condition's last transition from one status to another.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>status</b></td>
        <td>string</td>
        <td>
          Status of this condition; is it currently True, False, or Unknown?<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          Type of this condition. At most one of each condition type may apply to
a resource at any point in time.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>message</b></td>
        <td>string</td>
        <td>
          A Message containing details about this condition's last transition from
one status to another, if any.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### RestDefinition.status.configuration
<sup><sup>[↩ Parent](#restdefinitionstatus)</sup></sup>



Configuration: the configuration of the resource

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>apiVersion</b></td>
        <td>string</td>
        <td>
          APIVersion: the api version of the resource<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>kind</b></td>
        <td>string</td>
        <td>
          Kind: the kind of the resource<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### RestDefinition.status.resource
<sup><sup>[↩ Parent](#restdefinitionstatus)</sup></sup>



Resource: the resource to manage

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>apiVersion</b></td>
        <td>string</td>
        <td>
          APIVersion: the api version of the resource<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>kind</b></td>
        <td>string</td>
        <td>
          Kind: the kind of the resource<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>
