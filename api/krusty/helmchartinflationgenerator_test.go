// Copyright 2020 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package krusty_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	kusttest_test "sigs.k8s.io/kustomize/api/testutils/kusttest"
	"sigs.k8s.io/kustomize/kyaml/copyutil"
)

const expectedHelm = `
apiVersion: v1
data:
  rcon-password: Q0hBTkdFTUUh
kind: Secret
metadata:
  labels:
    app: test-minecraft
    chart: minecraft-3.1.3
    heritage: Helm
    release: test
  name: test-minecraft
type: Opaque
---
apiVersion: v1
kind: Service
metadata:
  labels:
    app: test-minecraft
    chart: minecraft-3.1.3
    heritage: Helm
    release: test
  name: test-minecraft
spec:
  ports:
  - name: minecraft
    port: 25565
    protocol: TCP
    targetPort: minecraft
  selector:
    app: test-minecraft
  type: ClusterIP
`

func TestHelmChartInflationGeneratorOld(t *testing.T) {
	th := kusttest_test.MakeEnhancedHarnessWithTmpRoot(t)
	defer th.Reset()
	if err := th.ErrIfNoHelm(); err != nil {
		t.Skip("skipping: " + err.Error())
	}

	th.WriteK(th.GetRoot(), `
helmChartInflationGenerator:
- chartName: minecraft
  chartRepoUrl: https://itzg.github.io/minecraft-server-charts
  chartVersion: 3.1.3
  releaseName: test
`)

	m := th.Run(th.GetRoot(), th.MakeOptionsPluginsEnabled())
	th.AssertActualEqualsExpected(m, expectedHelm)
}

func TestHelmChartInflationGenerator(t *testing.T) {
	th := kusttest_test.MakeEnhancedHarnessWithTmpRoot(t)
	defer th.Reset()
	if err := th.ErrIfNoHelm(); err != nil {
		t.Skip("skipping: " + err.Error())
	}

	th.WriteK(th.GetRoot(), `
helmCharts:
- name: minecraft
  repo: https://itzg.github.io/minecraft-server-charts
  version: 3.1.3
  releaseName: test
`)

	m := th.Run(th.GetRoot(), th.MakeOptionsPluginsEnabled())
	th.AssertActualEqualsExpected(m, expectedHelm)
}

// Last mile helm - show how kustomize puts helm charts into different
// namespaces with different customizations.
func TestHelmChartProdVsDev(t *testing.T) {
	th := kusttest_test.MakeEnhancedHarnessWithTmpRoot(t)
	defer th.Reset()
	if err := th.ErrIfNoHelm(); err != nil {
		t.Skip("skipping: " + err.Error())
	}
	dirBase := th.MkDir("base")
	dirProd := th.MkDir("prod")
	dirDev := th.MkDir("dev")
	dirBoth := th.MkDir("both")

	th.WriteK(dirBase, `
helmCharts:
- name: minecraft
  repo: https://itzg.github.io/minecraft-server-charts
  version: 3.1.3
  releaseName: test
`)
	th.WriteK(dirProd, `
namespace: prod
namePrefix: myProd-
resources:
- ../base
`)
	th.WriteK(dirDev, `
namespace: dev
namePrefix: myDev-
resources:
- ../base
`)
	th.WriteK(dirBoth, `
resources:
- ../dev
- ../prod
`)

	// Base unchanged
	m := th.Run(dirBase, th.MakeOptionsPluginsEnabled())
	th.AssertActualEqualsExpected(m, expectedHelm)

	// Prod has a "prod" namespace and a prefix.
	m = th.Run(dirProd, th.MakeOptionsPluginsEnabled())
	th.AssertActualEqualsExpected(m, `
apiVersion: v1
data:
  rcon-password: Q0hBTkdFTUUh
kind: Secret
metadata:
  labels:
    app: test-minecraft
    chart: minecraft-3.1.3
    heritage: Helm
    release: test
  name: myProd-test-minecraft
  namespace: prod
type: Opaque
---
apiVersion: v1
kind: Service
metadata:
  labels:
    app: test-minecraft
    chart: minecraft-3.1.3
    heritage: Helm
    release: test
  name: myProd-test-minecraft
  namespace: prod
spec:
  ports:
  - name: minecraft
    port: 25565
    protocol: TCP
    targetPort: minecraft
  selector:
    app: test-minecraft
  type: ClusterIP
`)

	// Both has two namespaces.
	m = th.Run(dirBoth, th.MakeOptionsPluginsEnabled())
	th.AssertActualEqualsExpected(m, `
apiVersion: v1
data:
  rcon-password: Q0hBTkdFTUUh
kind: Secret
metadata:
  labels:
    app: test-minecraft
    chart: minecraft-3.1.3
    heritage: Helm
    release: test
  name: myDev-test-minecraft
  namespace: dev
type: Opaque
---
apiVersion: v1
kind: Service
metadata:
  labels:
    app: test-minecraft
    chart: minecraft-3.1.3
    heritage: Helm
    release: test
  name: myDev-test-minecraft
  namespace: dev
spec:
  ports:
  - name: minecraft
    port: 25565
    protocol: TCP
    targetPort: minecraft
  selector:
    app: test-minecraft
  type: ClusterIP
---
apiVersion: v1
data:
  rcon-password: Q0hBTkdFTUUh
kind: Secret
metadata:
  labels:
    app: test-minecraft
    chart: minecraft-3.1.3
    heritage: Helm
    release: test
  name: myProd-test-minecraft
  namespace: prod
type: Opaque
---
apiVersion: v1
kind: Service
metadata:
  labels:
    app: test-minecraft
    chart: minecraft-3.1.3
    heritage: Helm
    release: test
  name: myProd-test-minecraft
  namespace: prod
spec:
  ports:
  - name: minecraft
    port: 25565
    protocol: TCP
    targetPort: minecraft
  selector:
    app: test-minecraft
  type: ClusterIP
`)
}

func TestHelmChartInflationGeneratorMultipleValuesFiles(t *testing.T) {
	th := kusttest_test.MakeEnhancedHarnessWithTmpRoot(t)
	defer th.Reset()
	if err := th.ErrIfNoHelm(); err != nil {
		t.Skip("skipping: " + err.Error())
	}

	copyValuesFilesTestChartsIntoHarness(t, th)

	th.WriteK(th.GetRoot(), `
helmCharts:
  - name: test-chart
    releaseName: test-chart
    additionalValuesFiles:
    - charts/valuesFiles/file1.yaml
    - charts/valuesFiles/file2.yaml
`)

	m := th.Run(th.GetRoot(), th.MakeOptionsPluginsEnabled())
	asYaml, err := m.AsYaml()
	require.NoError(t, err)
	require.Equal(t, string(asYaml), `apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    chart: test-1.0.0
  name: my-deploy
  namespace: file-2
spec:
  replicas: 1
  selector:
    matchLabels:
      app: test
  template:
    spec:
      containers:
      - image: test-image-file1:file1
        imagePullPolicy: Never
---
apiVersion: apps/v1
kind: Pod
metadata:
  annotations:
    helm.sh/hook: test
  name: test-chart
`)
}

func TestHelmChartInflationGeneratorApiVersions(t *testing.T) {
	th := kusttest_test.MakeEnhancedHarnessWithTmpRoot(t)
	defer th.Reset()
	if err := th.ErrIfNoHelm(); err != nil {
		t.Skip("skipping: " + err.Error())
	}

	copyValuesFilesTestChartsIntoHarness(t, th)

	th.WriteK(th.GetRoot(), `
helmCharts:
  - name: test-chart
    releaseName: test-chart
    apiVersions:
    - foo/v1
`)

	m := th.Run(th.GetRoot(), th.MakeOptionsPluginsEnabled())
	asYaml, err := m.AsYaml()
	require.NoError(t, err)
	require.Equal(t, string(asYaml), `apiVersion: foo/v1
kind: Deployment
metadata:
  labels:
    chart: test-1.0.0
  name: my-deploy
  namespace: default
spec:
  replicas: 1
  selector:
    matchLabels:
      app: test
  template:
    spec:
      containers:
      - image: test-image:v1.0.0
        imagePullPolicy: Always
---
apiVersion: foo/v1
kind: Pod
metadata:
  annotations:
    helm.sh/hook: test
  name: test-chart
`)
}

func TestHelmChartInflationGeneratorSkipTests(t *testing.T) {
	th := kusttest_test.MakeEnhancedHarnessWithTmpRoot(t)
	defer th.Reset()
	if err := th.ErrIfNoHelm(); err != nil {
		t.Skip("skipping: " + err.Error())
	}

	copyValuesFilesTestChartsIntoHarness(t, th)

	th.WriteK(th.GetRoot(), `
helmCharts:
  - name: test-chart
    releaseName: test-chart
    skipTests: true
`)

	m := th.Run(th.GetRoot(), th.MakeOptionsPluginsEnabled())
	asYaml, err := m.AsYaml()
	require.NoError(t, err)
	require.Equal(t, string(asYaml), `apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    chart: test-1.0.0
  name: my-deploy
  namespace: default
spec:
  replicas: 1
  selector:
    matchLabels:
      app: test
  template:
    spec:
      containers:
      - image: test-image:v1.0.0
        imagePullPolicy: Always
`)
}

func TestHelmChartInflationGeneratorNameTemplate(t *testing.T) {
	th := kusttest_test.MakeEnhancedHarnessWithTmpRoot(t)
	defer th.Reset()
	if err := th.ErrIfNoHelm(); err != nil {
		t.Skip("skipping: " + err.Error())
	}

	copyValuesFilesTestChartsIntoHarness(t, th)

	th.WriteK(th.GetRoot(), `
helmCharts:
  - name: test-chart
    nameTemplate: name-template
`)

	m := th.Run(th.GetRoot(), th.MakeOptionsPluginsEnabled())
	asYaml, err := m.AsYaml()
	require.NoError(t, err)
	require.Equal(t, string(asYaml), `apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    chart: test-1.0.0
  name: my-deploy
  namespace: default
spec:
  replicas: 1
  selector:
    matchLabels:
      app: test
  template:
    spec:
      containers:
      - image: test-image:v1.0.0
        imagePullPolicy: Always
---
apiVersion: apps/v1
kind: Pod
metadata:
  annotations:
    helm.sh/hook: test
  name: name-template
`)
}

func copyValuesFilesTestChartsIntoHarness(t *testing.T, th *kusttest_test.HarnessEnhanced) {
	t.Helper()

	thDir := filepath.Join(th.GetRoot(), "charts")
	chartDir := "testdata/helmcharts"

	fs := th.GetFSys()
	require.NoError(t, fs.MkdirAll(filepath.Join(thDir, "templates")))
	require.NoError(t, copyutil.CopyDir(th.GetFSys(), chartDir, thDir))
}
