// Copyright 2021 Google LLC
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

package pipeline

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/pkg"
	kptfilev1alpha2 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1alpha2"
	"k8s.io/klog"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// Executor hydrates a given pkg.
type Executor struct {
	PkgPath string
}

// Execute runs a pipeline.
func (e *Executor) Execute() error {
	rootPkg, err := newPkgNode(e.PkgPath, nil)
	if err != nil {
		return err
	}

	// initialize hydration context
	hctx := &hydrationContext{
		root: rootPkg,
		pkgs: map[string]*pkgNode{},
	}

	resources, err := hydrate(rootPkg, hctx)
	if err != nil {
		return fmt.Errorf("failed to run pipeline in package %q %w", rootPkg.Path(), err)
	}

	pkgWriter := &kio.LocalPackageWriter{PackagePath: rootPkg.Path()}
	err = pkgWriter.Write(resources)
	if err != nil {
		klog.Errorf("failed to update package: %v", err)
		return fmt.Errorf("failed to update package: %w", err)
	}
	return nil
}

// helper function to log debug information about a resource.
func debugResource(r *yaml.RNode) {
	meta, _ := r.GetMeta()
	klog.Infof("resource %s annotations: %v", meta.Name, meta.Annotations)
}

//
// hydrationContext contains bits to track state of a package hydration.
// This is sort of global state that is available to hydration step at
// each pkg along the hydration walk.
type hydrationContext struct {
	// root points to the root of hydration graph where we bagan our hydration journey
	root *pkgNode

	// pkgs refers to the packages undergoing hydration. pkgs are key'd by their
	// unique paths.
	pkgs map[string]*pkgNode
}

//
// pkgNode represents a package being hydrated. Think of it as a node in the hydration DAG.
//
type pkgNode struct {
	pkg *pkg.Pkg

	// state indicates if the pkg is being hydrated or done.
	state hydrationState

	// KRM resources that we have gathered post hydration for this package.
	// These inludes resources at this pkg as well all it's children.
	resources []*yaml.RNode
}

// newPkgNode returns a pkgNode instance given a path or pkg.
func newPkgNode(path string, p *pkg.Pkg) (pn *pkgNode, err error) {
	if path == "" && p == nil {
		return pn, fmt.Errorf("missing package path %s or package", path)
	}
	if path != "" {
		p, err = pkg.New(path)
		if err != nil {
			return pn, fmt.Errorf("failed to read package %w", err)
		}
	}
	// Note: Ensuring the presence of Kptfile can probably be moved
	// to the lower level pkg abstraction, but not sure if that
	// is desired in all the cases. So revisit this.
	if _, err = p.Kptfile(); err != nil {
		return pn, fmt.Errorf("failed to read kptfile for package %s %w", p, err)
	}
	pn = &pkgNode{
		pkg:   p,
		state: Dry, // package starts in dry state
	}
	return pn, nil
}

func (pn *pkgNode) Path() string {
	return string(pn.pkg.UniquePath)
}

// hydrationState represent hydration state of a pkg.
type hydrationState int

// constants for all the hydration states
const (
	Dry hydrationState = iota
	Hydrating
	Wet
)

func (s hydrationState) String() string {
	return []string{"Dry", "Hydrating", "Wet"}[s]
}

// hydrate hydrates given pkg and returns wet resources.
func hydrate(pn *pkgNode, hctx *hydrationContext) (resources []*yaml.RNode, err error) {
	currPkg, found := hctx.pkgs[pn.Path()]
	if found {
		switch currPkg.state {
		case Hydrating:
			// we detected a cycle
			err = fmt.Errorf("found cycle in dependencies for package %s", currPkg.Path())
			return resources, err
		case Wet:
			resources = currPkg.resources
			return resources, err
		default:
			return resources, fmt.Errorf("package %s detected in invalid state", currPkg.pkg)
		}
	} else {
		// add it to the discovered package list
		hctx.pkgs[pn.Path()] = pn
		currPkg = pn
	}
	// mark the pkg in hydrating
	currPkg.state = Hydrating
	var input []*yaml.RNode

	// determine sub packages to be hydrated
	subpkgs, err := currPkg.pkg.SubPackages()
	if err != nil {
		return resources, err
	}
	// hydrate recursively and gather hydated transitive resources.
	for _, subpkg := range subpkgs {
		var transitiveResources []*yaml.RNode
		var subPkgNode *pkgNode

		if subPkgNode, err = newPkgNode("", subpkg); err != nil {
			return resources, err
		}

		transitiveResources, err = hydrate(subPkgNode, hctx)
		if err != nil {
			err = fmt.Errorf("failed to run pipeline on subpackage %s %w", subpkg, err)
			return resources, err
		}

		input = append(input, transitiveResources...)
	}

	// hydrate current package
	currPkgResources, err := currPkg.pkg.LocalResources(false)
	if err != nil {
		return resources, err
	}
	// include current package's resources in the input resource list
	input = append(input, currPkgResources...)

	resources, err = currPkg.runPipeline(input, hctx)
	if err != nil {
		return resources, err
	}

	if hctx.root != currPkg {
		// Resources are read from local filesystem or generated at a package level, so the
		// path annotation in each resource points to path relative to that package.
		// But the resources are written to the file system at the root package level, so
		// the path annotation in each resources needs to be adjusted to be relative to the rootPkg.
		relPath, err := filepath.Rel(hctx.root.Path(), currPkg.Path())
		if err != nil {
			return nil, err
		}
		resources, err = adjustRelPath(resources, relPath)
		if err != nil {
			return nil, fmt.Errorf("failed to adjust relative path %w", err)
		}
	}

	// pkg is hydrated, mark the pkg as wet and update the resources
	currPkg.state = Wet
	currPkg.resources = resources

	return resources, err
}

// runPipeline runs the pipeline defined at current pkgNode on given input resources.
func (pn *pkgNode) runPipeline(input []*yaml.RNode, hctx *hydrationContext) ([]*yaml.RNode, error) {
	if len(input) == 0 {
		return nil, nil
	}

	output := &kio.PackageBuffer{}

	pl, err := pn.pkg.Pipeline()
	if err != nil {
		return nil, fmt.Errorf("failed to read pipeline for package %s %w", pn.pkg, err)
	}

	// empty pipeline
	if len(pl.Mutators) == 0 && len(pl.Validators) == 0 {
		return input, nil
	}

	filters, err := fnFilters(pl, pn.Path())
	if err != nil {
		return nil, fmt.Errorf("failed to get function filters: %w", err)
	}
	// create a kio pipeline from kyaml library to execute the function chains
	kioPipeline := kio.Pipeline{
		Inputs: []kio.Reader{
			&kio.PackageBuffer{Nodes: input},
		},
		Filters: filters,
		Outputs: []kio.Writer{output},
	}
	err = kioPipeline.Execute()
	if err != nil {
		err = fmt.Errorf("failed to run pipeline for pkg %v %w", pn.Path(), err)
		return nil, err
	}
	return output.Nodes, nil

}

// adjustRelPath updates the resources with given relative path.
func adjustRelPath(resources []*yaml.RNode, relPath string) ([]*yaml.RNode, error) {
	for _, r := range resources {
		meta, err := r.GetMeta()
		if err != nil {
			return resources, err
		}
		// TODO(droot): revisit this. There is an edge case where if relativePath(root, curr)
		// is same as relativePath(curr, local-resource)
		if relPath != "" && !strings.HasPrefix(meta.Annotations[kioutil.PathAnnotation], relPath) {
			newPath := path.Join(relPath, meta.Annotations[kioutil.PathAnnotation])
			err = r.PipeE(yaml.SetAnnotation(kioutil.PathAnnotation, newPath))
			if err != nil {
				return resources, err
			}
		}
	}
	return resources, nil
}

// fnFilters returns chain of functions that are applicable
// to a given pipeline.
func fnFilters(pl *kptfilev1alpha2.Pipeline, pkgPath string) ([]kio.Filter, error) {
	filters, err := fnChain(pl, pkgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get function chain: %w", err)
	}
	return filters, nil
}
