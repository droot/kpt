// Copyright 2022 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package builtins

import (
	"fmt"
	"io"
	"path"

	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
	"sigs.k8s.io/kustomize/kyaml/resid"
	"sigs.k8s.io/kustomize/kyaml/yaml"

	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
)

const (
	PkgContextFile = "package-context.yaml"
	pkgContextName = "kptfile.kpt.dev"
)

var (
	configMapGVK = resid.NewGvk("", "v1", "ConfigMap")
	kptfileGVK   = resid.NewGvk(kptfilev1.KptFileGroup, kptfilev1.KptFileVersion, kptfilev1.KptFileKind)
)

// PackageContextGenerator is a built-in KRM function that generates
// a KRM object that contains package context information that can be
// used by functions such as `set-namespace` to customize package with
// minimal configuration.
type PackageContextGenerator struct{}

// Run function reads the function input `resourceList` from a given reader `r`
// and writes the function output to the provided writer `w`.
// Run implements the function signature defined in
// sigs.k8s.io/kustomize/kyaml/fn/runtime/runtimeutil/FunctionFilter.Run.
func (pc *PackageContextGenerator) Run(r io.Reader, w io.Writer) error {
	rw := &kio.ByteReadWriter{
		Reader:                r,
		Writer:                w,
		KeepReaderAnnotations: true,
	}
	return framework.Execute(pc, rw)
}

// Process implements framework.ResourceListProcessor interface.
func (pc *PackageContextGenerator) Process(resourceList *framework.ResourceList) error {
	var contextResources, updatedResources []*yaml.RNode

	// This loop does the following:
	// - Filters out package context resources from the input resources
	// - Generates a package context resource for each kpt package (i.e Kptfile)
	for _, resource := range resourceList.Items {
		gvk := resid.GvkFromNode(resource)
		if gvk.Equals(configMapGVK) && resource.GetName() == pkgContextName {
			// drop existing package context resources
			continue
		}
		updatedResources = append(updatedResources, resource)
		if gvk.Equals(kptfileGVK) {
			// it's a Kptfile, generate a corresponding package context
			pkgContext, err := pkgContextResource(resource)
			if err != nil {
				resourceList.Results = framework.Results{
					&framework.Result{
						Message:  err.Error(),
						Severity: framework.Error,
					},
				}
				return resourceList.Results
			}
			contextResources = append(contextResources, pkgContext)
		}
	}

	for _, resource := range contextResources {
		updatedResources = append(updatedResources, resource)
		resourcePath, _, _ := kioutil.GetFileAnnotations(resource)
		resourceList.Results = append(resourceList.Results, &framework.Result{
			Message:  "generated package context",
			Severity: framework.Info,
			File:     &framework.File{Path: resourcePath, Index: 0},
		})
	}
	resourceList.Items = updatedResources
	return nil
}

// pkgContextResource generates package context resource from a given
// Kptfile. The resource is generated adjacent to the Kptfile of the package.
func pkgContextResource(kf *yaml.RNode) (*yaml.RNode, error) {
	cm := yaml.MustParse(fmt.Sprintf(`
apiVersion: v1
kind: ConfigMap
metadata:
  name: %s
  annotations:
    config.kubernetes.io/local-config: "true"
data: {}
`, pkgContextName))

	kptfilePath, _, err := kioutil.GetFileAnnotations(kf)
	if err != nil {
		return nil, err
	}
	annotations := map[string]string{
		kioutil.PathAnnotation: path.Join(path.Dir(kptfilePath), PkgContextFile),
	}

	for k, v := range annotations {
		if _, err := cm.Pipe(yaml.SetAnnotation(k, v)); err != nil {
			return nil, err
		}
	}
	cm.SetDataMap(map[string]string{
		"name": kf.GetName(),
	})
	return cm, nil
}

// DummyPkgContext returns content for package context that contains
// placeholder value for package name. This will be used to create
// abstract blueprints.
func DummyPkgContext() string {
	return fmt.Sprintf(`apiVersion: v1
kind: ConfigMap
metadata:
  name: %s
  annotations:
    config.kubernetes.io/local-config: "true"
data:
  name: example
`, pkgContextName)
}
