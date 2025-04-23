package controllers

import (
	"github.com/krateoplatformops/provider-runtime/pkg/controller"
	ctrl "sigs.k8s.io/controller-runtime"

	repo "github.com/krateoplatformops/oasgen-provider/internal/controllers/restdefinition"
)

// Setup creates all controllers with the supplied logger and adds them to
// the supplied manager.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	for _, setup := range []func(ctrl.Manager, controller.Options) error{
		repo.Setup,
	} {
		if err := setup(mgr, o); err != nil {
			return err
		}
	}
	return nil
}
