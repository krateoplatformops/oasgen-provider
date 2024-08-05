# oasgen-provider

`oasgen-provider` is a Kubernetes controller aimed at simplifying the integration of OAS 3.0 definitions into Kubernetes environments. It accomplishes this by automatically generating CRDs and controllers based on the OAS 3.0 definitions. This automation significantly reduces manual coding efforts, enhances flexibility, and improves integration between Kubernetes and external services or applications.

## Core Features

- **CRD Generation**: Parses OAS 3.0 definitions to automatically generate CRDs, allowing users to define custom resources that align with the schema described in their API specifications for seamless integration and management within Kubernetes.

- **Controller Generation**: Automates the generation of controllers alongside CRDs. Controllers are crucial in Kubernetes for maintaining desired states of resources by watching for changes in resource states and acting upon them accordingly.

- **Custom Resource Management**: Enables direct management of custom resources within Kubernetes, including operations like creating, updating, deleting, and monitoring the state of these resources, all in alignment with the definitions provided in the OAS 3.0 specification.

## Benefits

- **Streamlined Development**: Reduces manual coding efforts, making the development process for Kubernetes-native applications more efficient.
  
- **Enhanced Flexibility**: Allows for easy adaptation of Kubernetes resources to match evolving API specifications.
  
- **Improved Integration**: Enhances the integration between Kubernetes and external services or applications.

## Technical Overview

`oasgen-provider` analyzes OAS 3.0 definitions to understand the structure and requirements of the intended resources. Leveraging this information, it orchestrates the deployment of the [composition-dynamic-controller-v2](https://github.com/krateoplatformops/composition-dynamic-controller-v2). This controller is responsible for managing resources that correspond to the type defined by the CRD.

## Getting Started

1. **Prepare OAS Definition**: Obtain or create an OAS 3.0 specification that outlines the API and resources you wish to manage within Kubernetes.

2. **Install oasgen-provider**: Install the provider on your cluster using Helm. For example:

    ```bash
    $ helm repo add krateo https://charts.krateo.io
    $ helm repo update krateo
    $ helm install oasgen-provider krateo/oasgen-provider
    ```

3. **Create a RestDefinition Manifest**: The RestDefinition is the kind of resource that `oasgen-provider` manages. Below is an example of a RestDefinition manifest that, when applied to the cluster, creates a GitRepository CRD with fields from the OpenAPI specification provided and deploys a controller ready to manage resources of type GitRepository.

    ```yaml
    kind: RestDefinition
    apiVersion: swaggergen.krateo.io/v1alpha1
    metadata:
      name: repository-def
      namespace: default
    spec:
      deletionPolicy: Delete
      oasPath: https://raw.githubusercontent.com/matteogastaldello/azuredevops-oas3/main/git/git-new.yaml
      resourceGroup: azure.devops.com
      resource: 
        kind: GitRepository
        identifiers: 
        - id
        - name
        verbsDescription:
        - action: create
          method: POST
          path:  /{organization}/{project}/_apis/git/repositories
          altFieldMapping:
            project.id: project
        - action: get
          method: GET
          path:  /{organization}/{project}/_apis/git/repositories/{repositoryId}
          altFieldMapping:
            id: repositoryId
            project.id: project
        - action: findby
          method: GET
          path:  /{organization}/{project}/_apis/git/repositories
          altFieldMapping:
            project.id: project
        - action: delete
          method: DELETE
          path:  /{organization}/{project}/_apis/git/repositories/{repositoryId}
          altFieldMapping:
            project.id: project
            id: repositoryId
    ```

4. **Deploy to Kubernetes**: Apply the manifest and verify the generated CRD. If necessary, adjust the `RestDefinition` manifest to correct any incorrect path parameter mappings in `spec.resource.verbsDescription[i].path`.

5. **Ready to Go!**: At this stage, you have a running controller capable of managing resources of type 'GitRepository'. You can retrieve the Kind and APIVersion of installed CRDs in the status of the RestDefinition CR.

## Note on API Authentication

If the provided OAS specification mentions authentication methods, `oasgen-provider` will generate the corresponding authentication CRDs. Additionally, it adds an `authenticationRefs` field to the specs of the resource CRD to reference the CR of the authentication.

