package rbactools

import (
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type UninstallOptions struct {
	KubeClient     client.Client
	NamespacedName types.NamespacedName
	Log            func(msg string, keysAndValues ...any)
}
