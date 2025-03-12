package templates

import (
	"fmt"
	"testing"
)

const deploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ .resource }}-{{ .apiVersion }}-controller
  namespace: {{ .namespace }}
  labels:
	app.kubernetes.io/name: {{ .name }}
	app.kubernetes.io/instance: {{ .resource }}-{{ .apiVersion }}
	app.kubernetes.io/component: controller
	app.kubernetes.io/part-of: krateoplatformops
	app.kubernetes.io/managed-by: krateo
spec:
  replicas: 1
  selector:
	matchLabels:
	  app.kubernetes.io/name: {{ .name }}
  strategy:
	rollingUpdate:
	  maxSurge: 25%
	  maxUnavailable: 25%
	type: RollingUpdate
  template:
	metadata:
	  name: {{ .name }}
	  namespace: {{ .namespace }}
	  labels:
		app.kubernetes.io/name: {{ .name }}
	spec:
	  containers:
	  - name: {{ .resource }}-{{ .apiVersion }}-controller
		image: ghcr.io/krateoplatformops/composition-dynamic-controller:0.15.3
		imagePullPolicy: IfNotPresent
		env: 
		- name: HOME
		  value: /tmp
		args:
		  - -debug
		  - -group={{ .apiGroup }}
		  - -version={{ .apiVersion }}
		  - -resource={{ .resource }}
		  - -namespace={{ .namespace }}
		ports:
		- containerPort: 8080
		  name: metrics
		  protocol: TCP
		resources: {}
		terminationMessagePath: /dev/termination-log
		terminationMessagePolicy: File
	  dnsPolicy: ClusterFirst
	  restartPolicy: Always
	  schedulerName: default-scheduler
	  serviceAccount: {{ .name }}
	  serviceAccountName: {{ .name }}
	  terminationGracePeriodSeconds: 30
`

func TestDeploymentManifest(t *testing.T) {
	values := Values(Renderoptions{
		Group:     "composition.krateo.io",
		Version:   "v12-8-3",
		Resource:  "postgresqls",
		Name:      "postgres-tgz",
		Namespace: "default",
	})

	bin, err := Template(deploymentTemplate).Render(values)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(string(bin))
}
