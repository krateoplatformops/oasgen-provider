package restdefinition

import (
	"context"
	"errors"
	"fmt"

	"net/http"
	"os"
	"path"
	"strings"

	appsv1 "k8s.io/api/apps/v1"

	rtv1 "github.com/krateoplatformops/provider-runtime/apis/common/v1"
	"github.com/krateoplatformops/snowplow/plumbing/env"

	definitionv1alpha1 "github.com/krateoplatformops/oasgen-provider/apis/restdefinitions/v1alpha1"
	"github.com/krateoplatformops/provider-runtime/pkg/controller"
	"github.com/krateoplatformops/provider-runtime/pkg/event"
	"github.com/krateoplatformops/provider-runtime/pkg/logging"
	"github.com/krateoplatformops/provider-runtime/pkg/meta"
	"github.com/krateoplatformops/provider-runtime/pkg/ratelimiter"
	"github.com/pb33f/libopenapi"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/krateoplatformops/oasgen-provider/internal/tools/crd"
	"github.com/krateoplatformops/oasgen-provider/internal/tools/deploy"
	"github.com/krateoplatformops/oasgen-provider/internal/tools/deployment"
	"github.com/krateoplatformops/oasgen-provider/internal/tools/filegetter"
	"github.com/krateoplatformops/oasgen-provider/internal/tools/generation"
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
	errNotRestDefinition = "managed resource is not a RestDefinition"
	resourceVersion      = "v1alpha1"
)

var (
	RDCtemplateDeploymentPath = path.Join(os.TempDir(), "assets/rdc-deployment/deployment.yaml")
	RDCtemplateConfigmapPath  = path.Join(os.TempDir(), "assets/rdc-configmap/configmap.yaml")
	RDCrbacConfigFolder       = path.Join(os.TempDir(), "assets/rdc-rbac/")

	DEMO_FLAG = env.Bool("DEMO", true)
)

func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := reconciler.ControllerName(definitionv1alpha1.RestDefinitionGroupKind)

	log := o.Logger.WithValues("controller", name)

	recorder := mgr.GetEventRecorderFor(name)

	r := reconciler.NewReconciler(mgr,
		resource.ManagedKind(definitionv1alpha1.RestDefinitionGroupVersionKind),
		reconciler.WithExternalConnecter(&connector{
			kube:     mgr.GetClient(),
			log:      log,
			recorder: recorder,
		}),
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
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (reconciler.ExternalClient, error) {
	cr, ok := mg.(*definitionv1alpha1.RestDefinition)
	if !ok {
		return nil, errors.New(errNotRestDefinition)
	}
	RDCtemplateDeploymentPath = env.String("RDC_TEMPLATE_DEPLOYMENT_PATH", RDCtemplateDeploymentPath)
	RDCtemplateConfigmapPath = env.String("RDC_TEMPLATE_CONFIGMAP_PATH", RDCtemplateConfigmapPath)
	RDCrbacConfigFolder = env.String("RDC_RBAC_CONFIG_FOLDER", RDCrbacConfigFolder)

	var err error
	swaggerPath := cr.Spec.OASPath
	basePath := "/tmp/swaggergen-provider"
	err = os.MkdirAll(basePath, os.ModePerm)
	defer os.RemoveAll(basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	filegetter := &filegetter.Filegetter{
		Client:     http.DefaultClient,
		KubeClient: c.kube,
	}

	err = filegetter.GetFile(ctx, path.Join(basePath, path.Base(swaggerPath)), swaggerPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}

	contents, err := os.ReadFile(path.Join(basePath, path.Base(swaggerPath)))
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	d, err := libopenapi.NewDocument(contents)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	doc, modelErrors := d.BuildV3Model()
	if len(modelErrors) > 0 {
		return nil, fmt.Errorf("failed to build model: %w", errors.Join(modelErrors...))
	}
	if doc == nil {
		return nil, fmt.Errorf("failed to build model")
	}

	// Resolve model references
	resolvingErrors := doc.Index.GetResolver().Resolve()
	errs := []error{}
	for i := range resolvingErrors {
		c.log.Debug("Resolving error", "error", resolvingErrors[i].Error())
		errs = append(errs, resolvingErrors[i].ErrorRef)
	}
	if len(resolvingErrors) > 0 {
		return nil, fmt.Errorf("failed to resolve model references: %w", errors.Join(errs...))
	}

	return &external{
		kube: c.kube,
		log:  c.log,
		doc:  doc,
		rec:  c.recorder,
	}, nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	kube client.Client
	log  logging.Logger
	doc  *libopenapi.DocumentModel[v3.Document]
	rec  record.EventRecorder
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

	if DEMO_FLAG {
		//remove krateo.io/external-create-pending if it exists
		if cr.GetAnnotations() != nil {
			if _, ok := cr.GetAnnotations()["krateo.io/external-create-pending"]; ok {
				delete(cr.GetAnnotations(), "krateo.io/external-create-pending")
				e.log.Debug("Removing annotation krateo.io/external-create-pending. Demo Mode")
			}
		}

		err := e.kube.Update(ctx, cr)
		if err != nil {
			return reconciler.ExternalObservation{}, fmt.Errorf("error updating annotation: %w", err)
		}
	}

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

	obj := appsv1.Deployment{}
	err = objects.CreateK8sObject(&obj, gvr, types.NamespacedName{
		Namespace: cr.Namespace,
		Name:      cr.Name,
	}, RDCtemplateDeploymentPath)
	if err != nil {
		return reconciler.ExternalObservation{}, err
	}
	deployOk, deployReady, err := deployment.LookupDeployment(ctx, e.kube, &obj)
	if err != nil {
		return reconciler.ExternalObservation{
			ResourceExists:   true,
			ResourceUpToDate: true,
		}, err
	}
	if !deployOk {
		if meta.IsVerbose(cr) {
			e.log.Debug("Dynamic Controller not deployed yet",
				"name", obj.Name, "namespace", obj.Namespace, "gvr", gvr.String())
		}

		cr.SetConditions(rtv1.Unavailable().
			WithMessage(fmt.Sprintf("Dynamic Controller '%s' not deployed yet", obj.Name)))

		return reconciler.ExternalObservation{
			ResourceExists:   false,
			ResourceUpToDate: true,
		}, nil
	}

	if meta.IsVerbose(cr) {
		e.log.Debug("Dynamic Controller already deployed",
			"name", obj.Name, "namespace", obj.Namespace,
			"gvr", gvr.String())
	}

	if !deployReady {
		cr.SetConditions(rtv1.Unavailable().
			WithMessage(fmt.Sprintf("Dynamic Controller '%s' not ready yet", obj.Name)))

		return reconciler.ExternalObservation{
			ResourceExists:   true,
			ResourceUpToDate: true,
		}, nil
	}

	log := logging.NewNopLogger()
	if meta.IsVerbose(cr) {
		log = e.log
	}
	authenticationGVRs, err := getAuthenticationGVRs(ctx, e.kube, e.doc, cr)
	if err != nil {
		return reconciler.ExternalObservation{}, fmt.Errorf("getting authentication GVRs: %w", err)
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
		Log:          log.Debug,
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

	e.log.Debug("Creating RestDefinition", "Kind:", cr.Spec.Resource.Kind, "Group:", cr.Spec.ResourceGroup)

	gen, errors, err := oas2jsonschema.GenerateByteSchemas(e.doc, cr.Spec.Resource, cr.Spec.Resource.Identifiers)
	if err != nil {
		return fmt.Errorf("generating byte schemas: %w", err)
	}
	if meta.IsVerbose(cr) {
		for _, er := range errors {
			e.log.Debug("Generating Byte Schemas", "Error:", er)
		}
	}

	gvk := schema.GroupVersionKind{
		Group:   cr.Spec.ResourceGroup,
		Version: resourceVersion,
		Kind:    text.CapitaliseFirstLetter(cr.Spec.Resource.Kind),
	}
	gvr := plurals.ToGroupVersionResource(gvk)

	resource := crdgen.Generate(ctx, crdgen.Options{
		Managed:                true,
		WorkDir:                fmt.Sprintf("gen-crds/%s", cr.Spec.Resource.Kind),
		GVK:                    gvk,
		Categories:             []string{strings.ToLower(cr.Spec.Resource.Kind), "restresources", "rr"},
		SpecJsonSchemaGetter:   gen.OASSpecJsonSchemaGetter(),
		StatusJsonSchemaGetter: gen.OASStatusJsonSchemaGetter(),
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

	var authenticationGVRs []schema.GroupVersionResource
	for secSchemaPair := e.doc.Model.Components.SecuritySchemes.First(); secSchemaPair != nil; secSchemaPair = secSchemaPair.Next() {
		authSchemaName, err := generation.GenerateAuthSchemaName(secSchemaPair.Value())
		if err != nil {
			e.log.Debug("Generating Auth Schema Name", "Error:", err)
			return fmt.Errorf("generating Auth Schema Name: %w", err)
		}
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
			authenticationGVRs = append(authenticationGVRs, plurals.ToGroupVersionResource(gvk))
			cr.Status.Authentications = append(cr.Status.Authentications, definitionv1alpha1.KindApiVersion{
				Kind:       gvk.Kind,
				APIVersion: gvk.GroupVersion().String(),
			})
			continue
		}

		resource = crdgen.Generate(ctx, crdgen.Options{
			Managed:                false,
			WorkDir:                fmt.Sprintf("gen-crds/%s", authSchemaName),
			GVK:                    gvk,
			Categories:             []string{strings.ToLower(cr.Spec.Resource.Kind), "restauths", "ra"},
			SpecJsonSchemaGetter:   gen.OASAuthJsonSchemaGetter(authSchemaName),
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

		authenticationGVRs = append(authenticationGVRs, plurals.ToGroupVersionResource(gvk))

		cr.Status.Authentications = append(cr.Status.Authentications, definitionv1alpha1.KindApiVersion{
			Kind:       gvk.Kind,
			APIVersion: gvk.GroupVersion().String(),
		})
	}

	log := logging.NewNopLogger()
	if meta.IsVerbose(cr) {
		log = e.log
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
		GVR: gvr,
		Log: log.Debug,
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
		fmt.Println("Error updating status")
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

	e.log.Debug("Updating RestDefinition", "Kind:", cr.Spec.Resource.Kind, "Group:", cr.Spec.ResourceGroup)

	gvk := schema.GroupVersionKind{
		Group:   cr.Spec.ResourceGroup,
		Version: resourceVersion,
		Kind:    text.CapitaliseFirstLetter(cr.Spec.Resource.Kind),
	}
	gvr := plurals.ToGroupVersionResource(gvk)

	authenticationGVRs, err := getAuthenticationGVRs(ctx, e.kube, e.doc, cr)
	if err != nil {
		return fmt.Errorf("getting authentication GVRs: %w", err)
	}

	log := logging.NewNopLogger()
	if meta.IsVerbose(cr) {
		log = e.log
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
		GVR: gvr,
		Log: log.Debug,
	}
	_, err = deploy.Deploy(ctx, e.kube, opts)
	if err != nil {
		return fmt.Errorf("installing controller: %w", err)
	}

	cr.SetConditions(rtv1.Creating())
	cr.Status.Resource = definitionv1alpha1.KindApiVersion{
		Kind:       gvk.Kind,
		APIVersion: gvk.GroupVersion().String(),
	}
	cr.Status.OASPath = cr.Spec.OASPath

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

	e.log.Debug("Deleting RestDefinition", "Kind:", cr.Spec.Resource.Kind, "Group:", cr.Spec.ResourceGroup)

	var authenticationGVRs []schema.GroupVersionResource
	for secSchemaPair := e.doc.Model.Components.SecuritySchemes.First(); secSchemaPair != nil; secSchemaPair = secSchemaPair.Next() {
		authSchemaName, err := generation.GenerateAuthSchemaName(secSchemaPair.Value())
		if err != nil {
			return fmt.Errorf("generating Auth Schema Name: %w", err)
		}

		gvk := schema.GroupVersionKind{
			Group:   cr.Spec.ResourceGroup,
			Version: resourceVersion,
			Kind:    text.CapitaliseFirstLetter(authSchemaName),
		}

		gvr := plurals.ToGroupVersionResource(gvk)

		crdOk, err := crd.Lookup(ctx, e.kube, gvr)
		if err != nil {
			return fmt.Errorf("looking up CRD: %w", err)
		}

		if crdOk {
			e.log.Debug("CRD already exists, deleting", "Kind:", authSchemaName)
			err = crd.Uninstall(ctx, e.kube, schema.GroupResource{
				Group:    gvr.Group,
				Resource: gvr.Resource,
			})

			if err != nil {
				return fmt.Errorf("uninstalling authentication CRD: %w", err)
			}
			authenticationGVRs = append(authenticationGVRs, gvr)
		}
	}

	gvr := plurals.ToGroupVersionResource(schema.GroupVersionKind{
		Group:   cr.Spec.ResourceGroup,
		Version: resourceVersion,
		Kind:    cr.Spec.Resource.Kind,
	})

	log := logging.NewNopLogger()
	if meta.IsVerbose(cr) {
		log = e.log
	}
	opts := deploy.UndeployOptions{
		AuthenticationGVRs: authenticationGVRs,
		SkipCRD:            false,
		RBACFolderPath:     RDCrbacConfigFolder,
		KubeClient:         e.kube,
		NamespacedName: types.NamespacedName{
			Namespace: cr.Namespace,
			Name:      cr.Name,
		},
		GVR:                    gvr,
		Log:                    log.Debug,
		DeploymentTemplatePath: RDCtemplateDeploymentPath,
		ConfigmapTemplatePath:  RDCtemplateConfigmapPath,
	}

	err := deploy.Undeploy(ctx, e.kube, opts)
	if err != nil {
		return fmt.Errorf("uninstalling controller: %w", err)
	}

	err = e.kube.Status().Update(ctx, cr)

	e.log.Debug("Deleting RestDefinition", "Kind:", cr.Spec.Resource.Kind, "Group:", cr.Spec.ResourceGroup)
	e.rec.Eventf(cr, corev1.EventTypeNormal, "RestDefinitionDeleting",
		"RestDefinition '%s/%s' deleting", cr.Spec.Resource.Kind, cr.Spec.ResourceGroup)
	return err
}

func getAuthenticationGVRs(ctx context.Context, kube client.Client, doc *libopenapi.DocumentModel[v3.Document], cr *definitionv1alpha1.RestDefinition) ([]schema.GroupVersionResource, error) {
	var authenticationGVRs []schema.GroupVersionResource
	for secSchemaPair := doc.Model.Components.SecuritySchemes.First(); secSchemaPair != nil; secSchemaPair = secSchemaPair.Next() {
		authSchemaName, err := generation.GenerateAuthSchemaName(secSchemaPair.Value())
		if err != nil {
			return nil, fmt.Errorf("generating Auth Schema Name: %w", err)
		}
		gvk := schema.GroupVersionKind{
			Group:   cr.Spec.ResourceGroup,
			Version: resourceVersion,
			Kind:    text.CapitaliseFirstLetter(authSchemaName),
		}

		crdOk, err := crd.Lookup(ctx, kube, plurals.ToGroupVersionResource(gvk))
		if err != nil {
			return nil, fmt.Errorf("looking up CRD: %w", err)
		}
		if crdOk {
			authenticationGVRs = append(authenticationGVRs, plurals.ToGroupVersionResource(gvk))
			cr.Status.Authentications = append(cr.Status.Authentications, definitionv1alpha1.KindApiVersion{
				Kind:       gvk.Kind,
				APIVersion: gvk.GroupVersion().String(),
			})
			continue
		}
	}
	return authenticationGVRs, nil
}
