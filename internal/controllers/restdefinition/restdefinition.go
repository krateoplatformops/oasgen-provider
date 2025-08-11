package restdefinition

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
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

	"github.com/krateoplatformops/oasgen-provider/internal/controllers/logger"
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

	"github.com/krateoplatformops/crdgen"
	"github.com/krateoplatformops/oasgen-provider/internal/tools/text"
)

const (
	reconcileGracePeriod = 1 * time.Minute
	reconcileTimeout     = 4 * time.Minute
)

const (
	errNotRestDefinition = "managed resource is not a RestDefinition"
	resourceVersion      = "v1alpha1"

	restresourcesStillExistFinalizer = "composition.krateo.io/restresources-still-exist-finalizer"
	authInUseFinalizer               = "composition.krateo.io/auth-in-use-finalizer"
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

	log := logger.Logger{
		Verbose: meta.IsVerbose(cr),
		Logger:  c.log.WithValues("name", cr.Name, "namespace", cr.Namespace),
	}

	return &external{
		kube:   c.kube,
		log:    &log,
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
		e.log.Info("RestDefinition was deleted, skipping observation")
		var authenticationGVRs []schema.GroupVersionResource

		if len(cr.Status.Authentications) == 0 {
			e.log.Debug("No authentications found in status, trying to get from document")
			doc, err := e.getDocumentModelFromCR(ctx, cr)
			if err != nil {
				return reconciler.ExternalObservation{}, fmt.Errorf("getting document model from CR: %w", err)
			}
			authenticationGVRs = getAuthenticationGVRs(doc, cr)
		}

		for _, auth := range cr.Status.Authentications {
			gvk := schema.FromAPIVersionAndKind(auth.APIVersion, auth.Kind)
			gvr := plurals.ToGroupVersionResource(gvk)
			authenticationGVRs = append(authenticationGVRs, gvr)
		}

		err := manageFinalizers(ctx, e.kube, e.disc, authenticationGVRs, cr, e.log.Debug)
		if err != nil {
			return reconciler.ExternalObservation{}, fmt.Errorf("managing finalizers: %w", err)
		}
		return reconciler.ExternalObservation{
			ResourceExists:   false,
			ResourceUpToDate: true,
		}, e.Delete(ctx, cr)
	}

	doc, err := e.getDocumentModelFromCR(ctx, cr)
	if err != nil {
		return reconciler.ExternalObservation{}, fmt.Errorf("getting document model from CR: %w", err)
	}
	authenticationGVRs := getAuthenticationGVRs(doc, cr)

	gvr := plurals.ToGroupVersionResource(gvk)
	e.log.Info("Observing RestDefinition", "gvr", gvr.String())

	crdOk, err := crd.Lookup(ctx, e.kube, gvr)
	if err != nil {
		return reconciler.ExternalObservation{}, err
	}

	if !crdOk {
		e.log.Info("CRD not found", "gvr", gvr.String())

		cr.SetConditions(rtv1.Unavailable().
			WithMessage(fmt.Sprintf("CRD for '%s' does not exists yet", gvr.String())))
		return reconciler.ExternalObservation{
			ResourceExists:   false,
			ResourceUpToDate: true,
		}, nil
	}
	e.log.Info("Searching for Dynamic Controller", "gvr", gvr.String())

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
		e.log.Info("Dynamic Controller not ready yet",
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
		AuthenticationGVRs:     authenticationGVRs,
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
		e.log.Info("Rendered resources digest changed", "status", cr.Status.Digest, "rendered", dig)
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
		e.log.Info("Deployed resources digest changed", "status", cr.Status.Digest, "deployed", dig)
		return reconciler.ExternalObservation{
			ResourceExists:   true,
			ResourceUpToDate: false,
		}, nil
	}

	authenticationGVKs := getAuthenticationGVKs(doc, cr)
	for _, gvk := range authenticationGVKs {
		if !slices.Contains(cr.Status.Authentications, definitionv1alpha1.KindApiVersion{
			Kind:       gvk.Kind,
			APIVersion: gvk.GroupVersion().String(),
		}) {
			cr.Status.Authentications = append(cr.Status.Authentications, definitionv1alpha1.KindApiVersion{
				Kind:       gvk.Kind,
				APIVersion: gvk.GroupVersion().String(),
			})
		}
	}

	err = manageFinalizers(ctx, e.kube, e.disc, authenticationGVRs, cr, e.log.Debug)
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
		e.log.Info("External resource should not be created by provider, skip creating.")
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

		generator := oas2jsonschema.NewOASSchemaGenerator(doc, oas2jsonschema.DefaultGeneratorConfig())
		result, err := generator.Generate(cr.Spec.Resource, cr.Spec.Resource.Identifiers)
		if err != nil {
			return fmt.Errorf("generating schemas: %w", err)
		}
		for _, er := range result.GenerationWarnings {
			e.log.Debug("Schema generation warning", "Warning", er)
		}
		for _, er := range result.ValidationWarnings {
			e.log.Debug("Schema validation warning", "Warning", er)
		}

		resource := crdgen.Generate(ctx, crdgen.Options{
			Managed:                true,
			WorkDir:                fmt.Sprintf("gen-crds/%s", cr.Spec.Resource.Kind),
			GVK:                    gvk,
			Categories:             []string{strings.ToLower(cr.Spec.Resource.Kind), "restresources", "rr"},
			SpecJsonSchemaGetter:   result.OASSpecJsonSchemaGetter(),
			StatusJsonSchemaGetter: result.OASStatusJsonSchemaGetter(),
		})
		if resource.Err != nil {
			return fmt.Errorf("generating CRD: %w", resource.Err)
		}

		crdu, err := crd.Unmarshal(resource.Manifest)
		if err != nil {
			return fmt.Errorf("unmarshalling CRD: %w", err)
		}

		err = kube.Apply(ctx, e.kube, crdu, kube.ApplyOptions{})
		if err != nil {
			return fmt.Errorf("installing CRD: %w", err)
		}

		for authSchemaName := range result.AuthCRDSchemas {
			gvk := schema.GroupVersionKind{
				Group:   cr.Spec.ResourceGroup,
				Version: resourceVersion,
				Kind:    text.CapitaliseFirstLetter(authSchemaName),
			}

			crdOk, err := crd.Lookup(ctx, e.kube, plurals.ToGroupVersionResource(gvk))
			if err != nil {
				return fmt.Errorf("looking up CRD: %w", err)
			}
			if crdOk {
				e.log.Debug("CRD already exists", "Kind:", authSchemaName)
				if !slices.Contains(cr.Status.Authentications, definitionv1alpha1.KindApiVersion{
					Kind:       gvk.Kind,
					APIVersion: gvk.GroupVersion().String(),
				}) {
					e.log.Debug("Adding authentication CRD to status", "Kind:", authSchemaName)
					cr.Status.Authentications = append(cr.Status.Authentications, definitionv1alpha1.KindApiVersion{
						Kind:       gvk.Kind,
						APIVersion: gvk.GroupVersion().String(),
					})
				}

				err = e.kube.Status().Update(ctx, cr)
				if err != nil {
					return fmt.Errorf("updating status: %w", err)
				}
				continue
			}

			resource = crdgen.Generate(ctx, crdgen.Options{
				Managed:                false,
				WorkDir:                fmt.Sprintf("gen-crds/%s", authSchemaName),
				GVK:                    gvk,
				Categories:             []string{strings.ToLower(cr.Spec.Resource.Kind), "restauths", "ra"},
				SpecJsonSchemaGetter:   result.OASAuthCRDSchemaGetter(authSchemaName),
				StatusJsonSchemaGetter: oas2jsonschema.StaticJsonSchemaGetter(),
			})

			if resource.Err != nil {
				return fmt.Errorf("generating CRD: %w", resource.Err)
			}

			crdu, err := crd.Unmarshal(resource.Manifest)
			if err != nil {
				return fmt.Errorf("unmarshalling CRD: %w", err)
			}

			err = kube.Apply(ctx, e.kube, crdu, kube.ApplyOptions{})
			if err != nil {
				return fmt.Errorf("installing CRD: %w", err)
			}

			e.log.Info("Created authentication CRD", "Kind:", authSchemaName, "Group:", gvk.Group)

			if !slices.Contains(cr.Status.Authentications, definitionv1alpha1.KindApiVersion{
				Kind:       gvk.Kind,
				APIVersion: gvk.GroupVersion().String(),
			}) {
				e.log.Debug("Adding authentication CRD to status", "Kind:", authSchemaName)
				cr.Status.Authentications = append(cr.Status.Authentications, definitionv1alpha1.KindApiVersion{
					Kind:       gvk.Kind,
					APIVersion: gvk.GroupVersion().String(),
				})
			}

			err = e.kube.Status().Update(ctx, cr)
			if err != nil {
				return fmt.Errorf("updating status: %w", err)
			}
		}

		cr.SetConditions(rtv1.Creating())
		err = e.kube.Status().Update(ctx, cr)
		if err != nil {
			return fmt.Errorf("updating status: %w", err)
		}
		e.log.Info("Created CRD", "Kind:", cr.Spec.Resource.Kind, "Group:", cr.Spec.ResourceGroup)
		e.rec.Eventf(cr, corev1.EventTypeNormal, "RestDefinitionCreating",
			"RestDefinition '%s/%s' creating", cr.Spec.Resource.Kind, cr.Spec.ResourceGroup)

		return nil
	}

	doc, err := e.getDocumentModelFromCR(ctx, cr)
	if err != nil {
		return fmt.Errorf("getting document model from CR: %w", err)
	}
	authenticationGVRs := getAuthenticationGVRs(doc, cr)
	opts := deploy.DeployOptions{
		AuthenticationGVRs:     authenticationGVRs,
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
	cr.Status.OASPath = cr.Spec.OASPath
	cr.Status.Digest = dig

	err = e.kube.Status().Update(ctx, cr)
	if err != nil {
		return fmt.Errorf("updating status: %w", err)
	}

	e.log.Info("Created RestDefinition", "Kind:", cr.Spec.Resource.Kind, "Group:", cr.Spec.ResourceGroup)
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
		e.log.Info("External resource should not be updated by provider, skip updating.")
		return nil
	}

	doc, err := e.getDocumentModelFromCR(ctx, cr)
	if err != nil {
		return fmt.Errorf("getting document model from CR: %w", err)
	}

	e.log.Info("Updating RestDefinition", "Kind:", cr.Spec.Resource.Kind, "Group:", cr.Spec.ResourceGroup)

	gvk := schema.GroupVersionKind{
		Group:   cr.Spec.ResourceGroup,
		Version: resourceVersion,
		Kind:    text.CapitaliseFirstLetter(cr.Spec.Resource.Kind),
	}
	gvr := plurals.ToGroupVersionResource(gvk)

	authenticationGVRs := getAuthenticationGVRs(doc, cr)

	opts := deploy.DeployOptions{
		AuthenticationGVRs:     authenticationGVRs,
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
	cr.Status.OASPath = cr.Spec.OASPath
	cr.Status.Digest = dig

	err = e.kube.Status().Update(ctx, cr)
	if err != nil {
		return fmt.Errorf("updating status: %w", err)
	}

	e.log.Info("Updated RestDefinition", "Kind:", cr.Spec.Resource.Kind, "Group:", cr.Spec.ResourceGroup)
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
		e.log.Info("External resource should not be deleted by provider, skip deleting.")
		return nil
	}

	e.log.Info("Deleting RestDefinition", "Kind:", cr.Spec.Resource.Kind, "Group:", cr.Spec.ResourceGroup)

	var authenticationGVRs []schema.GroupVersionResource

	if len(cr.Status.Authentications) == 0 {
		e.log.Debug("No authentications found in status, trying to get from document")
		doc, err := e.getDocumentModelFromCR(ctx, cr)
		if err != nil {
			return fmt.Errorf("getting document model from CR: %w", err)
		}
		authenticationGVRs = getAuthenticationGVRs(doc, cr)
	}

	for _, auth := range cr.Status.Authentications {
		gvk := schema.FromAPIVersionAndKind(auth.APIVersion, auth.Kind)
		gvr := plurals.ToGroupVersionResource(gvk)
		authenticationGVRs = append(authenticationGVRs, gvr)
	}

	gvr := plurals.ToGroupVersionResource(schema.GroupVersionKind{
		Group:   cr.Spec.ResourceGroup,
		Version: resourceVersion,
		Kind:    cr.Spec.Resource.Kind,
	})
	skipDeploy := meta.FinalizerExists(cr, restresourcesStillExistFinalizer) || meta.FinalizerExists(cr, authInUseFinalizer)
	opts := deploy.UndeployOptions{
		AuthenticationGVRs: authenticationGVRs,
		SkipCRD:            false,
		SkipDeploy:         skipDeploy,
		RBACFolderPath:     RDCrbacConfigFolder,
		KubeClient:         e.kube,
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

	// Check if CRD exists and if there are any RestResources with the same group
	for _, gvr := range authenticationGVRs {
		crdOk, err := crd.Lookup(ctx, e.kube, gvr)
		if err != nil {
			return fmt.Errorf("looking up CRD: %w", err)
		}
		if crdOk {
			authExist, rrCount, err := kube.CountRestDefinitionsWithGroup(ctx, e.kube, e.disc, gvr.Group)
			if err != nil {
				return fmt.Errorf("counting resources: %w", err)
			}
			if rrCount == 1 && authExist {
				e.log.Info("CRD exists, deleting", "Group:", gvr.Group, "Resource:", gvr.Resource)
				err = crd.Uninstall(ctx, e.kube, schema.GroupResource{
					Group:    gvr.Group,
					Resource: gvr.Resource,
				})
				if err != nil {
					return fmt.Errorf("uninstalling authentication CRD: %w", err)
				}

			} else if rrCount > 1 {
				e.log.Debug("Skipping CRD deletion, RestDefinitions that reference this auth still exist",
					"Group", gvr.Group, "Count", rrCount)
			}
		}
	}

	if skipDeploy || opts.SkipCRD {
		e.log.Info(" RestResources still exist",
			"Group", gvr.Group, "Resource", gvr.Resource)
		return fmt.Errorf("restResources still exist")
	}

	e.log.Info("Deleting RestDefinition", "Kind:", cr.Spec.Resource.Kind, "Group:", cr.Spec.ResourceGroup)
	e.rec.Eventf(cr, corev1.EventTypeNormal, "RestDefinitionDeleting",
		"RestDefinition '%s/%s' deleting", cr.Spec.Resource.Kind, cr.Spec.ResourceGroup)
	return err
}

func getAuthenticationGVRs(doc oas2jsonschema.OASDocument, cr *definitionv1alpha1.RestDefinition) []schema.GroupVersionResource {
	var authenticationGVRs []schema.GroupVersionResource
	for _, scheme := range doc.SecuritySchemes() {
		gvk := schema.GroupVersionKind{
			Group:   cr.Spec.ResourceGroup,
			Version: resourceVersion,
			Kind:    text.CapitaliseFirstLetter(scheme.Name),
		}
		authenticationGVRs = append(authenticationGVRs, plurals.ToGroupVersionResource(gvk))
	}
	return authenticationGVRs
}

func getAuthenticationGVKs(doc oas2jsonschema.OASDocument, cr *definitionv1alpha1.RestDefinition) []schema.GroupVersionKind {
	var authenticationGVKs []schema.GroupVersionKind
	for _, scheme := range doc.SecuritySchemes() {
		gvk := schema.GroupVersionKind{
			Group:   cr.Spec.ResourceGroup,
			Version: resourceVersion,
			Kind:    text.CapitaliseFirstLetter(scheme.Name),
		}
		authenticationGVKs = append(authenticationGVKs, gvk)
	}
	return authenticationGVKs
}

func manageFinalizers(ctx context.Context, kubecli client.Client, disc discovery.DiscoveryInterface, authenticationGVRs []schema.GroupVersionResource, cr *definitionv1alpha1.RestDefinition, log func(msg string, keysAndValues ...any)) error {
	var n int
	var finalizer string
	var err error
	var authExist bool
	if len(authenticationGVRs) > 0 {
		authExist, n, err = kube.CountRestResourcesWithGroup(ctx, kubecli, disc, cr.Spec.ResourceGroup)
		if err != nil {
			return fmt.Errorf("counting resources: %w", err)
		}
		finalizer = authInUseFinalizer
	} else {
		uli := unstructured.UnstructuredList{}
		uli.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   cr.Spec.ResourceGroup,
			Version: resourceVersion,
			Kind:    text.CapitaliseFirstLetter(cr.Spec.Resource.Kind),
		})
		err := kubecli.List(ctx, &uli)
		if err != nil {
			if !strings.Contains(err.Error(), "no matches for") {
				return fmt.Errorf("listing resources: %w", err)
			}
		}
		n = len(uli.Items)
		finalizer = restresourcesStillExistFinalizer
	}
	if n > 0 {
		if !meta.FinalizerExists(cr, finalizer) && len(cr.Status.Authentications) > 0 {
			log("Existing Rest Resources with group", "Group", cr.Spec.ResourceGroup, "Count", n)
			log("Adding finalizer to RestDefinition", "name", cr.Name, "finalizer", finalizer)
			meta.AddFinalizer(cr, finalizer)
			err = kubecli.Update(ctx, cr)
			if err != nil {
				return err
			}
		}
	} else if !authExist {
		if meta.FinalizerExists(cr, finalizer) {
			log("Removing finalizer from RestDefinition", "name", cr.Name, "finalizer", finalizer)
			meta.RemoveFinalizer(cr, finalizer)
			err = kubecli.Update(ctx, cr)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

//func getDocumentModelFromCR(ctx context.Context, kube client.Client, cr *definitionv1alpha1.RestDefinition) (*libopenapi.DocumentModel[v3.Document], //error) {
//	var err error
//	swaggerPath := cr.Spec.OASPath
//	basePath := "/tmp/swaggergen-provider"
//	err = os.MkdirAll(basePath, os.ModePerm)
//	defer os.RemoveAll(basePath)
//	if err != nil {
//		return nil, fmt.Errorf("failed to create directory: %w", err)
//	}
//
//	filegetter := &filegetter.Filegetter{
//		Client:     http.DefaultClient,
//		KubeClient: kube,
//	}
//
//	err = filegetter.GetFile(ctx, path.Join(basePath, path.Base(swaggerPath)), swaggerPath, nil)
//	if err != nil {
//		return nil, fmt.Errorf("failed to download file: %w", err)
//	}
//
//	contents, err := os.ReadFile(path.Join(basePath, path.Base(swaggerPath)))
//	if err != nil {
//		return nil, fmt.Errorf("failed to read file: %w", err)
//	}
//
//	d, err := libopenapi.NewDocument(contents)
//	if err != nil {
//		return nil, fmt.Errorf("failed to read file: %w", err)
//	}
//
//	doc, modelErrors := d.BuildV3Model()
//	if len(modelErrors) > 0 {
//		return nil, fmt.Errorf("failed to build model: %w", errors.Join(modelErrors...))
//	}
//	if doc == nil {
//		return nil, fmt.Errorf("failed to build model")
//	}
//
//	// Resolve model references
//	resolvingErrors := doc.Index.GetResolver().Resolve()
//	errs := []error{}
//	for i := range resolvingErrors {
//		errs = append(errs, resolvingErrors[i].ErrorRef)
//	}
//	if len(resolvingErrors) > 0 {
//		return nil, fmt.Errorf("failed to resolve model references: %w", errors.Join(errs...))
//	}
//	return doc, nil
//}

func (e *external) getDocumentModelFromCR(ctx context.Context, cr *definitionv1alpha1.RestDefinition) (oas2jsonschema.OASDocument, error) {
	swaggerPath := cr.Spec.OASPath
	basePath := "/tmp/swaggergen-provider"
	err := os.MkdirAll(basePath, os.ModePerm)
	defer os.RemoveAll(basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	filegetter := &filegetter.Filegetter{
		Client:     http.DefaultClient,
		KubeClient: e.kube,
	}

	err = filegetter.GetFile(ctx, path.Join(basePath, path.Base(swaggerPath)), swaggerPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}

	contents, err := os.ReadFile(path.Join(basePath, path.Base(swaggerPath)))
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return e.parser.Parse(contents)
}
