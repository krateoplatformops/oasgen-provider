package definition

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/gobuffalo/flect"
	rtv1 "github.com/krateoplatformops/provider-runtime/apis/common/v1"

	fgetter "github.com/hashicorp/go-getter"

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

	"github.com/krateoplatformops/provider-runtime/pkg/reconciler"
	"github.com/krateoplatformops/provider-runtime/pkg/resource"

	"github.com/krateoplatformops/oasgen-provider/internal/controllers/compositiondefinition/generator"
	"github.com/krateoplatformops/oasgen-provider/internal/tools/crds"
	"github.com/krateoplatformops/oasgen-provider/internal/tools/deployment"
	"github.com/krateoplatformops/oasgen-provider/internal/tools/generation"
	"github.com/krateoplatformops/oasgen-provider/internal/tools/rbactools"

	"github.com/krateoplatformops/oasgen-provider/internal/crdgen"
	"github.com/krateoplatformops/oasgen-provider/internal/tools/generator/text"
)

const (
	errNotRestDefinition = "managed resource is not a RestDefinition"
	labelKeyGroup        = "krateo.io/crd-group"
	labelKeyVersion      = "krateo.io/crd-version"
	labelKeyResource     = "krateo.io/crd-resource"
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
		reconciler.WithLogger(log),
		reconciler.WithRecorder(event.NewAPIRecorder(recorder)))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&definitionv1alpha1.RestDefinition{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
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
	var err error
	swaggerPath := cr.Spec.OASPath

	basePath := "/tmp/swaggergen-provider"
	err = os.MkdirAll(basePath, os.ModePerm)
	defer os.RemoveAll(basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}
	const errorLocalPath = "relative paths require a module with a pwd"
	err = fgetter.GetFile(filepath.Join(basePath, filepath.Base(swaggerPath)), swaggerPath)
	if err != nil && err.Error() == errorLocalPath {
		swaggerPath, err = filepath.Abs(swaggerPath)
		if err != nil {
			return nil, fmt.Errorf("failed to get absolute path: %w", err)
		}
		err = fgetter.GetFile(filepath.Join(basePath, filepath.Base(swaggerPath)), swaggerPath)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}

	contents, _ := os.ReadFile(path.Join(basePath, path.Base(swaggerPath)))
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
		Version: "v1alpha1",
		Kind:    cr.Spec.Resource.Kind,
	}

	gvr := deployment.ToGroupVersionResource(gvk)
	log.Printf("[DBG] Observing (gvk: %s, gvr: %s)\n", gvk.String(), gvr.String())

	crdOk, err := deployment.LookupCRD(ctx, e.kube, gvr)
	if err != nil {
		return reconciler.ExternalObservation{}, err
	}

	if !crdOk {
		log.Printf("[DBG] CRD does not exists yet (gvr: %q)\n", gvr.String())

		cr.SetConditions(rtv1.Unavailable().
			WithMessage(fmt.Sprintf("CRD for '%s' does not exists yet", gvr.String())))
		return reconciler.ExternalObservation{
			ResourceExists:   false,
			ResourceUpToDate: true,
		}, nil
	}

	log.Printf("[DBG] Searching for Dynamic Controller (gvr: %q)\n", gvr.String())

	obj, err := deployment.CreateDeployment(gvr, types.NamespacedName{
		Namespace: cr.Namespace,
		Name:      cr.Name,
	})
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

	gen, err, errors := generator.GenerateByteSchemas(e.doc, cr.Spec.Resource, cr.Spec.Resource.Identifiers)
	if err != nil {
		return fmt.Errorf("generating byte schemas: %w", err)
	}
	if meta.IsVerbose(cr) {
		for _, er := range errors {
			e.log.Debug("Generating Byte Schemas", "Error:", er)
		}
	}

	role, err := rbactools.InitRole(types.NamespacedName{
		Namespace: cr.GetNamespace(),
		Name:      cr.GetName(),
	})
	if err != nil {
		return fmt.Errorf("initializing role: %w", err)
	}

	resource := crdgen.Generate(ctx, crdgen.Options{
		Managed: true,
		WorkDir: fmt.Sprintf("gen-crds/%s", cr.Spec.Resource.Kind),
		GVK: schema.GroupVersionKind{
			Group:   cr.Spec.ResourceGroup,
			Version: "v1alpha1",
			Kind:    text.CapitaliseFirstLetter(cr.Spec.Resource.Kind),
		},
		Categories:             []string{strings.ToLower(cr.Spec.Resource.Kind)},
		SpecJsonSchemaGetter:   gen.OASSpecJsonSchemaGetter(),
		StatusJsonSchemaGetter: gen.OASStatusJsonSchemaGetter(),
	})
	if resource.Err != nil {
		return fmt.Errorf("generating CRD: %w", resource.Err)
	}

	crd, err := crds.UnmarshalCRD(resource.Manifest)
	if err != nil {
		return fmt.Errorf("unmarshalling CRD: %w", err)
	}

	err = crds.InstallCRD(ctx, e.kube, crd)
	if err != nil {
		return fmt.Errorf("installing CRD: %w", err)
	}
	rbactools.PopulateRole(resource.GVK, &role)

	for secSchemaPair := e.doc.Model.Components.SecuritySchemes.First(); secSchemaPair != nil; secSchemaPair = secSchemaPair.Next() {
		authSchemaName, err := generation.GenerateAuthSchemaName(secSchemaPair.Value())
		if err != nil {
			e.log.Debug("Generating Auth Schema Name", "Error:", err)
			continue
		}

		crdOk, err := deployment.LookupCRD(ctx, e.kube, schema.GroupVersionResource{
			Group:    cr.Spec.ResourceGroup,
			Version:  "v1alpha1",
			Resource: flect.Pluralize(strings.ToLower(authSchemaName)),
		})
		if err != nil {
			return fmt.Errorf("looking up CRD: %w", err)
		}
		if crdOk {
			e.log.Debug("CRD already exists", "Kind:", authSchemaName)
			rbactools.PopulateRole(schema.GroupVersionKind{
				Group:   cr.Spec.ResourceGroup,
				Version: "v1alpha1",
				Kind:    text.CapitaliseFirstLetter(authSchemaName),
			}, &role)
			continue
		}

		resource = crdgen.Generate(ctx, crdgen.Options{
			Managed: false,
			WorkDir: fmt.Sprintf("gen-crds/%s", authSchemaName),
			GVK: schema.GroupVersionKind{
				Group:   cr.Spec.ResourceGroup,
				Version: "v1alpha1",
				Kind:    text.CapitaliseFirstLetter(authSchemaName),
			},
			Categories:             []string{strings.ToLower(cr.Spec.Resource.Kind)},
			SpecJsonSchemaGetter:   gen.OASAuthJsonSchemaGetter(authSchemaName),
			StatusJsonSchemaGetter: generator.StaticJsonSchemaGetter(),
		})

		if resource.Err != nil {
			return fmt.Errorf("generating CRD: %w", resource.Err)
		}

		crd, err := crds.UnmarshalCRD(resource.Manifest)
		if err != nil {
			return fmt.Errorf("unmarshalling CRD: %w", err)
		}

		err = crds.InstallCRD(ctx, e.kube, crd)
		if err != nil {
			return fmt.Errorf("installing CRD: %w", err)
		}

		rbactools.PopulateRole(resource.GVK, &role)
	}

	err = deployment.Deploy(ctx, deployment.DeployOptions{
		KubeClient: e.kube,
		NamespacedName: types.NamespacedName{
			Namespace: cr.Namespace,
			Name:      cr.Name,
		},
		Spec:            &cr.Spec,
		ResourceVersion: "v1alpha1",
		Role:            role,
	})
	if err != nil {
		return fmt.Errorf("deploying controller: %w", err)
	}

	cr.SetConditions(rtv1.Creating())
	err = e.kube.Status().Update(ctx, cr)

	e.log.Debug("Created RestDefinition", "Kind:", cr.Spec.Resource.Kind, "Group:", cr.Spec.ResourceGroup)
	e.rec.Eventf(cr, corev1.EventTypeNormal, "RestDefinitionCreating",
		"RestDefinition '%s/%s' creating", cr.Spec.Resource.Kind, cr.Spec.ResourceGroup)
	return err
}

func (e *external) Update(ctx context.Context, mg resource.Managed) error {
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

	opts := deployment.UndeployOptions{
		KubeClient: e.kube,
		NamespacedName: types.NamespacedName{
			Namespace: cr.Namespace,
			Name:      cr.Name,
		},
		GVR: schema.GroupVersionResource{
			Group:    cr.Spec.ResourceGroup,
			Version:  "v1alpha1",
			Resource: flect.Pluralize(strings.ToLower(cr.Spec.Resource.Kind)),
		},
		Log:             e.log.Debug,
		SecuritySchemes: e.doc.Model.Components.SecuritySchemes,
	}
	if meta.IsVerbose(cr) {
		opts.Log = e.log.Debug
	}

	err := deployment.Undeploy(ctx, opts)
	if err != nil {
		return fmt.Errorf("uninstalling controller: %w", err)
	}

	err = e.kube.Status().Update(ctx, cr)

	e.log.Debug("Deleting RestDefinition", "Kind:", cr.Spec.Resource.Kind, "Group:", cr.Spec.ResourceGroup)
	e.rec.Eventf(cr, corev1.EventTypeNormal, "RestDefinitionDeleting",
		"RestDefinition '%s/%s' deleting", cr.Spec.Resource.Kind, cr.Spec.ResourceGroup)
	return err
}
