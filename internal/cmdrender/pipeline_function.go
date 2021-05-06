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

// Package pipeline provides struct definitions for Pipeline and utility
// methods to read and write a pipeline resource.
package cmdrender

import (
	"context"
	goerrors "errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/GoogleContainerTools/kpt/thirdparty/kyaml/fn/runtime/runtimeutil"

	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/fnruntime"
	"github.com/GoogleContainerTools/kpt/internal/printer"
	"github.com/GoogleContainerTools/kpt/internal/types"
	"github.com/GoogleContainerTools/kpt/pkg/api/fnresult/v1alpha2"
	kptfilev1alpha2 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1alpha2"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// newFunctionRunner returns a kio.Filter given a specification of a function
// and it's config.
func newFunctionRunner(ctx context.Context, hctx *hydrationContext, f *kptfilev1alpha2.Function, pkgPath types.UniquePath) (kio.Filter, error) {
	config, err := newFnConfig(f, pkgPath)
	if err != nil {
		return nil, err
	}

	cfn := &fnruntime.ContainerFn{
		Path:  pkgPath,
		Image: f.Image,
		Ctx:   ctx,
	}

	return &functionRunner{
		ctx:             ctx,
		containerRunner: cfn,
		fnResults:       hctx.fnResults,
		filter: &runtimeutil.FunctionFilter{
			Run:            cfn.Run,
			FunctionConfig: config,
		},
	}, nil
}

// functionRunner wraps FunctionFilter and implements kio.Filter interface.
type functionRunner struct {
	ctx             context.Context
	fnResults       *v1alpha2.ResultList
	containerRunner *fnruntime.ContainerFn
	filter          *runtimeutil.FunctionFilter
}

func (fr *functionRunner) Filter(input []*yaml.RNode) (output []*yaml.RNode, err error) {
	pr := printer.FromContextOrDie(fr.containerRunner.Ctx)
	printOpt := printer.NewOpt()
	pr.OptPrintf(printOpt, "[RUNNING] %q\n", fr.containerRunner.Image)
	output, err = fr.do(input)
	if err != nil {
		pr.OptPrintf(printOpt, "[FAIL] %q\n", fr.containerRunner.Image)
		return output, err
	}
	// capture the result from running the function
	pr.OptPrintf(printOpt, "[PASS] %q\n", fr.containerRunner.Image)
	return output, nil
}

func (fr *functionRunner) do(input []*yaml.RNode) (output []*yaml.RNode, err error) {
	var results []framework.ResultItem

	output, err = fr.filter.Filter(input)

	_ = toResultItems(fr.filter.Result(), &results)
	fnResult := v1alpha2.Result{
		Image:   fr.containerRunner.Image,
		Results: results,
	}
	fr.fnResults.Items = append(fr.fnResults.Items, fnResult)
	if err != nil {
		var fnErr *errors.FnExecError
		if goerrors.As(err, &fnErr) {
			fnResult.ExitCode = fnErr.ExitCode
			fnResult.Stderr = fnErr.Stderr
		}
		return output, err
	}
	fnResult.ExitCode = 0
	return output, nil
}

func toResultItems(yml *yaml.RNode, results *[]framework.ResultItem) error {
	err := yaml.Unmarshal([]byte(yml.MustString()), results)
	if err != nil {
		return err
	}
	return nil
}

func newFnConfig(f *kptfilev1alpha2.Function, pkgPath types.UniquePath) (*yaml.RNode, error) {
	const op errors.Op = "fn.readConfig"
	var fn errors.Fn = errors.Fn(f.Image)

	var node *yaml.RNode
	switch {
	case f.ConfigPath != "":
		path := filepath.Join(string(pkgPath), f.ConfigPath)
		file, err := os.Open(path)
		if err != nil {
			return nil, errors.E(op, fn,
				fmt.Errorf("missing function config %q", f.ConfigPath))
		}
		b, err := ioutil.ReadAll(file)
		if err != nil {
			return nil, errors.E(op, fn, err)
		}
		node, err = yaml.Parse(string(b))
		if err != nil {
			return nil, errors.E(op, fn, fmt.Errorf("invalid function config %q %w", f.ConfigPath, err))
		}
		// directly use the config from file
		return node, nil
	case !kptfilev1alpha2.IsNodeZero(&f.Config):
		// directly use the inline config
		return yaml.NewRNode(&f.Config), nil
	case len(f.ConfigMap) != 0:
		node = yaml.NewMapRNode(&f.ConfigMap)
		if node == nil {
			return nil, nil
		}
		// create a ConfigMap only for configMap config
		configNode := yaml.MustParse(`
apiVersion: v1
kind: ConfigMap
metadata:
  name: function-input
data: {}
`)
		err := configNode.PipeE(yaml.SetField("data", node))
		if err != nil {
			return nil, errors.E(op, fn, err)
		}
		return configNode, nil
	}
	// no need to return ConfigMap if no config given
	return nil, nil
}
