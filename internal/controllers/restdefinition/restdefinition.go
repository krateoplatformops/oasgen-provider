package restdefinition

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"os"
	"path"
	"strings"

	appsv1 "k8s.io/api/apps/v1"

	"github.com/krateoplatformops/plumbing/env"
	rtv1 "github.com/krateoplatformops/provider-runtime/apis/common/v1"

	definitionv1alpha1 "github.com/krateoplatformops/oasgen-provider/apis/restdefinitions/v1alpha1"
	"github.com/krateoplatformops/provider-runtime/pkg/controller"
	"github.com/krateoplatformops/provider-runtime/pkg/event"
	"github.com/krateoplatformops/provider-runtime/pkg/logging"
	"github.com/krateoplatformops/provider-runtime/pkg/meta"
	"github.com/krateoplatformops/provider-runtime/pkg/ratelimiter"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/krateoplatformops/oasgen-provider/internal/tools/crd"
	"github.com/krateoplatformops/oasgen-provider/internal/tools/deploy"
	"github.com/krateoplatformops/oasgen-provider/internal/tools/deployment"
	"github.com/krateoplatformops/oasgen-provider/internal/tools/filegetter"
	"github.com/krateoplatformops/oasgen-provider/internal/tools/kube"
	"github.com/krateoplatformops/oasgen-provider/internal/tools/oas2jsonschema"
	"github.com/krateoplatformops/oasgen-provider/internal/tools/objects"
	"github.com/krateoplatformops/oasgen-provider/internal/tools/plurals"
	"github.com/krateoplatformops/provider-runtime/pkg/reconciler"
	"github.com/krateoplatformops/provider-runtime/pkg/resource"

	//"github.com/krateoplatformops/crdgen/v2"
	"github.com/krateoplatformops/oasgen-provider/internal/tools/text"
	"github.com/krateoplatformops/plumbing/crdgen"
)

const (
	reconcileGracePeriod = 1 * time.Minute
	reconcileTimeout     = 4 * time.Minute
)

const (
	errNotRestDefinition = "managed resource is not a RestDefinition"
	resourceVersion      = "v1alpha1"

	restresourcesStillExistFinalizer = "composition.krateo.io/restresources-still-exist-finalizer"
)

var (
	RDCtemplateDeploymentPath = path.Join(os.TempDir(), "assets/rdc-deployment/deployment.yaml")
	RDCtemplateConfigmapPath  = path.Join(os.TempDir(), "assets/rdc-configmap/configmap.yaml")
	RDCrbacConfigFolder       = path.Join(os.TempDir(), "assets/rdc-rbac/")
)

func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := reconciler.ControllerName(definitionv1alpha1.RestDefinitionGroupKind)

	log := o.Logger.WithValues("controller", name)

	recorder := mgr.GetEventRecorderFor(name)

	cfg := mgr.GetConfig()
	cli, err := client.New(cfg, client.Options{})
	if err != nil {
		return fmt.Errorf("failed to create kube client: %w", err)
	}

	discovery, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return fmt.Errorf("failed to create discovery client: %w", err)
	}

	r := reconciler.NewReconciler(mgr,
		resource.ManagedKind(definitionv1alpha1.RestDefinitionGroupVersionKind),
		reconciler.WithExternalConnecter(&connector{
			kube:     cli,
			log:      log,
			recorder: recorder,
			disc:     discovery,
			parser:   oas2jsonschema.NewLibOASParser(),
		}),
		reconciler.WithTimeout(reconcileTimeout),
		reconciler.WithCreationGracePeriod(reconcileGracePeriod),
		reconciler.WithPollInterval(o.PollInterval),
		reconciler.WithLogger(log),
		reconciler.WithRecorder(event.NewAPIRecorder(recorder)))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&definitionv1alpha1.RestDefinition{}).
		Complete(ratelimiter.New(name, r, o.GlobalRateLimiter))
}

type connector struct {
	kube     client.Client
	log      logging.Logger
	recorder record.EventRecorder
	disc     discovery.DiscoveryInterface
	parser   oas2jsonschema.Parser
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (reconciler.ExternalClient, error) {
	cr, ok := mg.(*definitionv1alpha1.RestDefinition)
	if !ok {
		return nil, errors.New(errNotRestDefinition)
	}
	RDCtemplateDeploymentPath = env.String("RDC_TEMPLATE_DEPLOYMENT_PATH", RDCtemplateDeploymentPath)
	RDCtemplateConfigmapPath = env.String("RDC_TEMPLATE_CONFIGMAP_PATH", RDCtemplateConfigmapPath)
	RDCrbacConfigFolder = env.String("RDC_RBAC_CONFIG_FOLDER", RDCrbacConfigFolder)

	log := c.log.WithValues("name", cr.Name, "namespace", cr.Namespace)

	return &external{
		kube:   c.kube,
		log:    log,
		rec:    c.recorder,
		disc:   c.disc,
		parser: c.parser,
	}, nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	kube   client.Client
	log    logging.Logger
	rec    record.EventRecorder
	disc   discovery.DiscoveryInterface
	parser oas2jsonschema.Parser
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (reconciler.ExternalObservation, error) {
	cr, ok := mg.(*definitionv1alpha1.RestDefinition)
	if !ok {
		return reconciler.ExternalObservation{}, errors.New(errNotRestDefinition)
	}

	gvk := schema.GroupVersionKind{
		Group:   cr.Spec.ResourceGroup,
		Version: resourceVersion,
		Kind:    cr.Spec.Resource.Kind,
	}

	if meta.WasDeleted(cr) {
		e.log.Debug("RestDefinition was deleted, skipping observation")
		err := manageFinalizers(ctx, e.kube, cr, e.log.Debug)
		if err != nil {
			return reconciler.ExternalObservation{}, fmt.Errorf("managing finalizers: %w", err)
		}
		return reconciler.ExternalObservation{
			ResourceExists:   false,
			ResourceUpToDate: true,
		}, e.Delete(ctx, cr)
	}

	configurationGVR := getConfigurationGVR(cr)

	gvr := plurals.ToGroupVersionResource(gvk)
	e.log.Debug("Observing RestDefinition", "gvr", gvr.String())

	crdOk, err := crd.Lookup(ctx, e.kube, gvr)
	if err != nil {
		return reconciler.ExternalObservation{}, err
	}

	if !crdOk {
		e.log.Debug("CRD not found", "gvr", gvr.String())

		cr.SetConditions(rtv1.Unavailable().
			WithMessage(fmt.Sprintf("CRD for '%s' does not exists yet", gvr.String())))
		return reconciler.ExternalObservation{
			ResourceExists:   false,
			ResourceUpToDate: true,
		}, nil
	}
	e.log.Debug("Searching for Dynamic Controller", "gvr", gvr.String())

	deploymentNSName := types.NamespacedName{
		Namespace: cr.Namespace,
		Name:      cr.Name + deploy.ControllerResourceSuffix,
	}
	obj := appsv1.Deployment{}
	err = objects.CreateK8sObject(&obj, gvr, deploymentNSName, RDCtemplateDeploymentPath)
	if err != nil {
		return reconciler.ExternalObservation{}, err
	}
	deployOk, deployReady, err := deployment.LookupDeployment(ctx, e.kube, &obj)
	if err != nil {
		return reconciler.ExternalObservation{}, err
	}
	if !deployOk {
		e.log.Debug("Dynamic Controller not deployed yet",
			"name", obj.Name, "namespace", obj.Namespace, "gvr", gvr.String())

		cr.SetConditions(rtv1.Unavailable().
			WithMessage(fmt.Sprintf("Dynamic Controller '%s' not deployed yet", obj.Name)))

		return reconciler.ExternalObservation{
			ResourceExists:   false,
			ResourceUpToDate: true,
		}, nil
	}

	e.log.Debug("Dynamic Controller already deployed",
		"name", obj.Name, "namespace", obj.Namespace,
		"gvr", gvr.String(), "ready", deployReady,
		"replicas", *obj.Spec.Replicas, "readyReplicas", obj.Status.ReadyReplicas)

	if !deployReady {
		e.log.Debug("Dynamic Controller not ready yet",
			"name", obj.Name, "namespace", obj.Namespace,
		)

		cr.SetConditions(rtv1.Unavailable().
			WithMessage(fmt.Sprintf("Dynamic Controller '%s' not ready yet", obj.Name)))

		return reconciler.ExternalObservation{
			ResourceExists:   false,
			ResourceUpToDate: true,
		}, nil
	}
	opts := deploy.DeployOptions{
		ConfigurationGVR:       configurationGVR,
		RBACFolderPath:         RDCrbacConfigFolder,
		DeploymentTemplatePath: RDCtemplateDeploymentPath,
		ConfigmapTemplatePath:  RDCtemplateConfigmapPath,
		KubeClient:             e.kube,
		NamespacedName: types.NamespacedName{
			Namespace: cr.Namespace,
			Name:      cr.Name,
		},
		GVR:          gvr,
		Log:          e.log.Debug,
		DryRunServer: true,
	}

	dig, err := deploy.Deploy(ctx, e.kube, opts)
	if err != nil {
		return reconciler.ExternalObservation{}, err
	}

	if cr.Status.Digest != dig {
		e.log.Debug("Rendered resources digest changed", "status", cr.Status.Digest, "rendered", dig)
		return reconciler.ExternalObservation{
			ResourceExists:   true,
			ResourceUpToDate: false,
		}, nil
	}

	dig, err = deploy.Lookup(ctx, e.kube, opts)
	if err != nil {
		return reconciler.ExternalObservation{}, err
	}
	if cr.Status.Digest != dig {
		e.log.Debug("Deployed resources digest changed", "status", cr.Status.Digest, "deployed", dig)
		return reconciler.ExternalObservation{
			ResourceExists:   true,
			ResourceUpToDate: false,
		}, nil
	}

	err = manageFinalizers(ctx, e.kube, cr, e.log.Debug)
	if err != nil {
		return reconciler.ExternalObservation{}, fmt.Errorf("managing finalizers: %w", err)
	}

	cr.SetConditions(rtv1.Available())
	return reconciler.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: true,
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*definitionv1alpha1.RestDefinition)
	if !ok {
		return errors.New(errNotRestDefinition)
	}

	if !meta.IsActionAllowed(cr, meta.ActionCreate) {
		e.log.Debug("External resource should not be created by provider, skip creating.")
		return nil
	}

	e.log.Info("Creating RestDefinition", "Kind:", cr.Spec.Resource.Kind, "Group:", cr.Spec.ResourceGroup)

	gvk := schema.GroupVersionKind{
		Group:   cr.Spec.ResourceGroup,
		Version: resourceVersion,
		Kind:    text.CapitaliseFirstLetter(cr.Spec.Resource.Kind),
	}
	gvr := plurals.ToGroupVersionResource(gvk)

	crdOk, err := crd.Lookup(ctx, e.kube, gvr)
	if err != nil {
		return err
	}

	if !crdOk {
		doc, err := e.getDocumentModelFromCR(ctx, cr)
		if err != nil {
			return fmt.Errorf("getting document model from CR: %w", err)
		}

		// Shim needed to convert definitionv1alpha1.VerbsDescription to oas2jsonschema.Verbs
		// Verbs is a type defined within the oas2jsonschema package
		// and so it's not tied with the RestDefinition CRD
		verbs := make([]oas2jsonschema.Verb, len(cr.Spec.Resource.VerbsDescription))
		for i, v := range cr.Spec.Resource.VerbsDescription {
			verbs[i] = oas2jsonschema.Verb{
				Action: v.Action,
				Method: v.Method,
				Path:   v.Path,
			}
		}

		// Shim needed to convert definitionv1alpha1.ConfigurationFields to oas2jsonschema.ConfigurationFields
		configurationFields := make([]oas2jsonschema.ConfigurationField, 0, len(cr.Spec.Resource.ConfigurationFields))
		for _, v := range cr.Spec.Resource.ConfigurationFields {
			actions, err := expandWildcardActions(v.FromRestDefinition.Actions, cr.Spec.Resource.VerbsDescription)
			if err != nil {
				return fmt.Errorf("expanding wildcard for actions in configurationFields: %w", err)
			}

			configurationFields = append(configurationFields, oas2jsonschema.ConfigurationField{
				FromOpenAPI: oas2jsonschema.FromOpenAPI{
					Name: v.FromOpenAPI.Name,
					In:   v.FromOpenAPI.In,
				},
				FromRestDefinition: oas2jsonschema.FromRestDefinition{
					Actions: actions,
				},
			})
		}

		// Create the resource configuration for the OAS schema generator
		// We pass only relevant fields from the RestDefinition needed for schema generation
		resourceConfig := &oas2jsonschema.ResourceConfig{
			Verbs:                  verbs,
			Identifiers:            cr.Spec.Resource.Identifiers,
			AdditionalStatusFields: cr.Spec.Resource.AdditionalStatusFields,
			ConfigurationFields:    configurationFields,
			ExcludedSpecFields:     cr.Spec.Resource.ExcludedSpecFields,
		}

		// Create the OAS schema generator
		generator := oas2jsonschema.NewOASSchemaGenerator(
			doc,
			oas2jsonschema.DefaultGeneratorConfig(),
			resourceConfig,
		)

		result, err := generator.Generate()
		if err != nil {
			// Fatal error, we cannot continue
			return fmt.Errorf("generating schemas: %w", err)
		}
		if len(result.GenerationWarnings) > 0 {
			e.log.Debug("Some schema generation warnings were found, below the list")
			for _, er := range result.GenerationWarnings {
				e.log.Debug("Schema generation warning", "Warning", er)
			}
		}
		if len(result.ValidationWarnings) > 0 {
			e.log.Debug("Some schema validation warnings were found, below the list")
			for _, er := range result.ValidationWarnings {
				e.log.Debug("Schema validation warning", "Warning", er)
			}
		}

		e.log.Debug("Generating CRD for", "Kind:", cr.Spec.Resource.Kind, "Group:", cr.Spec.ResourceGroup)

		opts := crdgen.Options{
			Group:        gvk.Group,
			Version:      gvk.Version,
			Kind:         gvk.Kind,
			Categories:   []string{strings.ToLower(cr.Spec.Resource.Kind), "restresources", "rr"},
			SpecSchema:   result.SpecSchema,
			StatusSchema: result.StatusSchema,
			Managed:      true,
		}

		res, err := crdgen.Generate(opts)
		if err != nil {
			return fmt.Errorf("generating CRD: %w", err)
		}

		crdu, err := crd.Unmarshal(res)
		if err != nil {
			return fmt.Errorf("unmarshalling CRD: %w", err)
		}

		e.log.Debug("Applying CRD for", "Kind:", cr.Spec.Resource.Kind, "Group:", cr.Spec.ResourceGroup)
		err = kube.Apply(ctx, e.kube, crdu, kube.ApplyOptions{})
		if err != nil {
			return fmt.Errorf("installing CRD: %w", err)
		}

		e.log.Debug("Configuration fields defined, generating Configuration CRD")

		cfgGVK := schema.GroupVersionKind{
			Group:   cr.Spec.ResourceGroup,
			Version: resourceVersion,
			Kind:    text.CapitaliseFirstLetter(cr.Spec.Resource.Kind) + "Configuration",
		}

		e.log.Debug("Generating Configuration CRD", "Kind", cfgGVK.Kind, "Group", cfgGVK.Group)

		e.log.Debug("Configuration Schema", "Schema", string(result.ConfigurationSchema))

		cfgOpts := crdgen.Options{
			Group:      cfgGVK.Group,
			Version:    cfgGVK.Version,
			Kind:       cfgGVK.Kind,
			Categories: []string{strings.ToLower(cr.Spec.Resource.Kind), "restconfigs", "rc"},
			SpecSchema: result.ConfigurationSchema,
			Managed:    false,
		}

		cfgResource, err := crdgen.Generate(cfgOpts)
		if err != nil {
			return fmt.Errorf("generating configuration CRD: %w", err)
		}

		cfgCRDU, err := crd.Unmarshal(cfgResource)
		if err != nil {
			return fmt.Errorf("unmarshalling configuration CRD: %w", err)
		}

		e.log.Debug("Applying Configuration CRD", "Kind", cfgGVK.Kind, "Group", cfgGVK.Group)
		err = kube.Apply(ctx, e.kube, cfgCRDU, kube.ApplyOptions{})
		if err != nil {
			return fmt.Errorf("installing configuration CRD: %w", err)
		}
		e.log.Debug("Applied Configuration CRD", "Kind", cfgGVK.Kind, "Group", cfgGVK.Group)

		cr.SetConditions(rtv1.Creating())
		err = e.kube.Status().Update(ctx, cr)
		if err != nil {
			return fmt.Errorf("updating status: %w", err)
		}
		e.log.Debug("Applied CRD", "Kind:", cr.Spec.Resource.Kind, "Group:", cr.Spec.ResourceGroup)
		e.rec.Eventf(cr, corev1.EventTypeNormal, "RestDefinitionCreating",
			"RestDefinition '%s/%s' creating", cr.Spec.Resource.Kind, cr.Spec.ResourceGroup)

		return nil
	}

	configurationGVR := getConfigurationGVR(cr)
	opts := deploy.DeployOptions{
		ConfigurationGVR:       configurationGVR,
		RBACFolderPath:         RDCrbacConfigFolder,
		DeploymentTemplatePath: RDCtemplateDeploymentPath,
		ConfigmapTemplatePath:  RDCtemplateConfigmapPath,
		KubeClient:             e.kube,
		NamespacedName: types.NamespacedName{
			Namespace: cr.Namespace,
			Name:      cr.Name,
		},
		GVR: gvr,
		Log: e.log.Debug,
	}
	dig, err := deploy.Deploy(ctx, e.kube, opts)
	if err != nil {
		return fmt.Errorf("installing controller: %w", err)
	}

	cr.SetConditions(rtv1.Creating())
	cr.Status.Resource = definitionv1alpha1.KindApiVersion{
		Kind:       gvk.Kind,
		APIVersion: gvk.GroupVersion().String(),
	}
	cfgGVK := getConfigurationGVK(cr)
	cr.Status.Configuration = definitionv1alpha1.KindApiVersion{
		Kind:       cfgGVK.Kind,
		APIVersion: cfgGVK.GroupVersion().String(),
	}
	cr.Status.OASPath = cr.Spec.OASPath
	cr.Status.Digest = dig

	err = e.kube.Status().Update(ctx, cr)
	if err != nil {
		return fmt.Errorf("updating status: %w", err)
	}

	e.log.Debug("Created RestDefinition", "Kind:", cr.Spec.Resource.Kind, "Group:", cr.Spec.ResourceGroup)
	e.rec.Eventf(cr, corev1.EventTypeNormal, "RestDefinitionCreating",
		"RestDefinition '%s/%s' creating", cr.Spec.Resource.Kind, cr.Spec.ResourceGroup)
	return err
}

func (e *external) Update(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*definitionv1alpha1.RestDefinition)
	if !ok {
		return errors.New(errNotRestDefinition)
	}

	if !meta.IsActionAllowed(cr, meta.ActionUpdate) {
		e.log.Debug("External resource should not be updated by provider, skip updating.")
		return nil
	}

	e.log.Info("Updating RestDefinition", "Kind:", cr.Spec.Resource.Kind, "Group:", cr.Spec.ResourceGroup)

	gvk := schema.GroupVersionKind{
		Group:   cr.Spec.ResourceGroup,
		Version: resourceVersion,
		Kind:    text.CapitaliseFirstLetter(cr.Spec.Resource.Kind),
	}
	gvr := plurals.ToGroupVersionResource(gvk)

	configurationGVR := getConfigurationGVR(cr)
	opts := deploy.DeployOptions{
		ConfigurationGVR:       configurationGVR,
		RBACFolderPath:         RDCrbacConfigFolder,
		DeploymentTemplatePath: RDCtemplateDeploymentPath,
		ConfigmapTemplatePath:  RDCtemplateConfigmapPath,
		KubeClient:             e.kube,
		NamespacedName: types.NamespacedName{
			Namespace: cr.Namespace,
			Name:      cr.Name,
		},
		GVR: gvr,
		Log: e.log.Debug,
	}
	dig, err := deploy.Deploy(ctx, e.kube, opts)
	if err != nil {
		return fmt.Errorf("installing controller: %w", err)
	}

	cr.SetConditions(rtv1.Creating())
	cr.Status.Resource = definitionv1alpha1.KindApiVersion{
		Kind:       gvk.Kind,
		APIVersion: gvk.GroupVersion().String(),
	}
	cfgGVK := getConfigurationGVK(cr)
	cr.Status.Configuration = definitionv1alpha1.KindApiVersion{
		Kind:       cfgGVK.Kind,
		APIVersion: cfgGVK.GroupVersion().String(),
	}
	cr.Status.OASPath = cr.Spec.OASPath
	cr.Status.Digest = dig

	err = e.kube.Status().Update(ctx, cr)
	if err != nil {
		return fmt.Errorf("updating status: %w", err)
	}

	e.log.Debug("Updated RestDefinition", "Kind:", cr.Spec.Resource.Kind, "Group:", cr.Spec.ResourceGroup)
	e.rec.Eventf(cr, corev1.EventTypeNormal, "RestDefinitionUpdating",
		"RestDefinition '%s/%s' updating", cr.Spec.Resource.Kind, cr.Spec.ResourceGroup)
	return nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*definitionv1alpha1.RestDefinition)
	if !ok {
		return errors.New(errNotRestDefinition)
	}

	if !meta.IsActionAllowed(cr, meta.ActionDelete) {
		e.log.Debug("External resource should not be deleted by provider, skip deleting.")
		return nil
	}

	e.log.Info("Deleting RestDefinition", "Kind:", cr.Spec.Resource.Kind, "Group:", cr.Spec.ResourceGroup)

	gvr := plurals.ToGroupVersionResource(schema.GroupVersionKind{
		Group:   cr.Spec.ResourceGroup,
		Version: resourceVersion,
		Kind:    cr.Spec.Resource.Kind,
	})

	skipDeploy := meta.FinalizerExists(cr, restresourcesStillExistFinalizer)

	configurationGVR := getConfigurationGVR(cr)
	opts := deploy.UndeployOptions{
		ConfigurationGVR: configurationGVR,
		SkipCRD:          false,
		SkipDeploy:       skipDeploy,
		RBACFolderPath:   RDCrbacConfigFolder,
		KubeClient:       e.kube,
		NamespacedName: types.NamespacedName{
			Namespace: cr.Namespace,
			Name:      cr.Name,
		},
		GVR:                    gvr,
		Log:                    e.log.Debug,
		DeploymentTemplatePath: RDCtemplateDeploymentPath,
		ConfigmapTemplatePath:  RDCtemplateConfigmapPath,
	}

	err := deploy.Undeploy(ctx, e.kube, opts)
	if err != nil {
		return fmt.Errorf("uninstalling controller: %w", err)
	}

	if skipDeploy || opts.SkipCRD {
		e.log.Debug(" RestResources still exist",
			"Group", gvr.Group, "Resource", gvr.Resource)
		return fmt.Errorf("restResources still exist")
	}

	e.log.Debug("Deleting RestDefinition", "Kind:", cr.Spec.Resource.Kind, "Group:", cr.Spec.ResourceGroup)
	e.rec.Eventf(cr, corev1.EventTypeNormal, "RestDefinitionDeleting",
		"RestDefinition '%s/%s' deleting", cr.Spec.Resource.Kind, cr.Spec.ResourceGroup)
	return err
}

func getConfigurationGVR(cr *definitionv1alpha1.RestDefinition) schema.GroupVersionResource {
	cfgGVK := getConfigurationGVK(cr)
	return plurals.ToGroupVersionResource(cfgGVK)
}

func getConfigurationGVK(cr *definitionv1alpha1.RestDefinition) schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   cr.Spec.ResourceGroup,
		Version: resourceVersion,
		Kind:    text.CapitaliseFirstLetter(cr.Spec.Resource.Kind) + "Configuration",
	}
}

func manageFinalizers(ctx context.Context, kubecli client.Client, cr *definitionv1alpha1.RestDefinition, log func(msg string, keysAndValues ...any)) error {
	log("Managing finalizers for RestDefinition", "name", cr.Name, "namespace", cr.Namespace)

	// Check if RestResources still exist for this RestDefinition
	gvk := schema.GroupVersionKind{
		Group:   cr.Spec.ResourceGroup,
		Version: resourceVersion,
		Kind:    text.CapitaliseFirstLetter(cr.Spec.Resource.Kind),
	}

	uli := unstructured.UnstructuredList{}
	uli.SetGroupVersionKind(gvk)
	err := kubecli.List(ctx, &uli)
	if err != nil && !strings.Contains(err.Error(), "no matches for") {

		// If the CRD is missing, we assume no resources exist
		if strings.Contains(err.Error(), "the server could not find the requested resource") {
			log("CRD not found, treating as no resources exist",
				"Group", cr.Spec.ResourceGroup,
				"Kind", cr.Spec.Resource.Kind,
				"Version", resourceVersion,
				"error", err.Error())
			uli.Items = nil
			err = nil
		} else {
			return fmt.Errorf("listing RestResources: %w", err)
		}
	}

	restResourceCount := len(uli.Items)

	// Manage restresources-still-exist finalizer
	if restResourceCount > 0 {
		if !meta.FinalizerExists(cr, restresourcesStillExistFinalizer) {
			log("Existing RestResources found", "Group", cr.Spec.ResourceGroup, "Kind", cr.Spec.Resource.Kind, "Count", restResourceCount)
			log("Adding finalizer to RestDefinition", "name", cr.Name, "finalizer", restresourcesStillExistFinalizer)
			meta.AddFinalizer(cr, restresourcesStillExistFinalizer)
			err = kubecli.Update(ctx, cr)
			if err != nil {
				return fmt.Errorf("adding restresources finalizer: %w", err)
			}
		}
	} else {
		if meta.FinalizerExists(cr, restresourcesStillExistFinalizer) {
			log("No RestResources found, removing finalizer", "name", cr.Name, "finalizer", restresourcesStillExistFinalizer)
			meta.RemoveFinalizer(cr, restresourcesStillExistFinalizer)
			err = kubecli.Update(ctx, cr)
			if err != nil {
				return fmt.Errorf("removing restresources finalizer: %w", err)
			}
		}
	}

	return nil
}

func (e *external) getDocumentModelFromCR(ctx context.Context, cr *definitionv1alpha1.RestDefinition) (oas2jsonschema.OASDocument, error) {
	OASPath := cr.Spec.OASPath
	basePath := "/tmp/ogen-provider"
	err := os.MkdirAll(basePath, os.ModePerm)
	defer os.RemoveAll(basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	filegetter := &filegetter.Filegetter{
		Client:     http.DefaultClient,
		KubeClient: e.kube,
	}

	err = filegetter.GetFile(ctx, path.Join(basePath, path.Base(OASPath)), OASPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}

	contents, err := os.ReadFile(path.Join(basePath, path.Base(OASPath)))
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return e.parser.Parse(contents)
}
