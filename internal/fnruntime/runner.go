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

package fnruntime

import (
	"context"
	goerrors "errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/printer"
	"github.com/GoogleContainerTools/kpt/internal/types"
	fnresult "github.com/GoogleContainerTools/kpt/pkg/api/fnresult/v1alpha2"
	kptfilev1alpha2 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1alpha2"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/fn/runtime/runtimeutil"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// NewContainerRunner returns a kio.Filter given a specification of a container function
// and it's config.
func NewContainerRunner(ctx context.Context, f *kptfilev1alpha2.Function, pkgPath types.UniquePath, fnResults *fnresult.ResultList) (kio.Filter, error) {
	config, err := newFnConfig(f, pkgPath)
	if err != nil {
		return nil, err
	}
	cfn := &ContainerFn{
		Path:  pkgPath,
		Image: f.Image,
		Ctx:   ctx,
	}
	fltr := &runtimeutil.FunctionFilter{
		Run:            cfn.Run,
		FunctionConfig: config,
	}
	fnResult := &fnresult.Result{
		Image: f.Image,
		// TODO(droot): This is required for making structured results subpackage aware.
		// Enable this once test harness supports filepath based assertions.
		// Pkg: string(pkgPath),
	}
	return NewFunctionRunner(ctx, fltr, false, fnResult, fnResults)
}

// NewFunctionRunner returns a kio.Filter given a specification of a function
// and it's config.
func NewFunctionRunner(ctx context.Context,
	fltr *runtimeutil.FunctionFilter,
	disableOutput bool,
	fnResult *fnresult.Result,
	fnResults *fnresult.ResultList) (kio.Filter, error) {
	name := fnResult.Image
	if name == "" {
		name = fnResult.ExecPath
	}
	return &FunctionRunner{
		ctx:           ctx,
		name:          name,
		filter:        fltr,
		disableOutput: disableOutput,
		fnResult:      fnResult,
		fnResults:     fnResults,
	}, nil
}

// FunctionRunner wraps FunctionFilter and implements kio.Filter interface.
type FunctionRunner struct {
	ctx           context.Context
	name          string
	disableOutput bool
	filter        *runtimeutil.FunctionFilter
	fnResult      *fnresult.Result
	fnResults     *fnresult.ResultList
}

func (fr *FunctionRunner) Filter(input []*yaml.RNode) (output []*yaml.RNode, err error) {
	if fr.disableOutput {
		return fr.do(input)
	}
	pr := printer.FromContextOrDie(fr.ctx)
	printOpt := printer.NewOpt()

	pr.OptPrintf(printOpt, "[RUNNING] %q\n", fr.name)
	output, err = fr.do(input)
	if err != nil {
		pr.OptPrintf(printOpt, "[FAIL] %q\n", fr.name)
		printFnResult(fr.ctx, fr.fnResult)
		var fnErr *ExecError
		if goerrors.As(err, &fnErr) {
			printFnExecErr(fr.ctx, fnErr)
			return nil, errors.ErrAlreadyHandled
		}
		return nil, err
	}
	pr.OptPrintf(printOpt, "[PASS] %q\n", fr.name)
	printFnResult(fr.ctx, fr.fnResult)
	return output, err
}

// do executes the kpt function and returns the modified resources.
// fnResult is updated with the function results returned by the kpt function.
func (fr *FunctionRunner) do(input []*yaml.RNode) (output []*yaml.RNode, err error) {
	fnResult := fr.fnResult

	output, err = fr.filter.Filter(input)

	// parse the results irrespective of the success/failure of fn exec
	resultErr := parseStructuredResult(fr.filter.Results, fnResult)
	if resultErr != nil {
		// Not sure if it's a good idea. This may mask the original
		// function exec error. Revisit this if this turns out to be true.
		return output, resultErr
	}
	if err != nil {
		var fnErr *ExecError
		if goerrors.As(err, &fnErr) {
			fnResult.ExitCode = fnErr.ExitCode
			fnResult.Stderr = fnErr.Stderr
			fnErr.FnResult = fnResult
			fr.fnResults.ExitCode = 1
		}
		// accumulate the results
		fr.fnResults.Items = append(fr.fnResults.Items, *fnResult)
		return output, err
	}
	fnResult.ExitCode = 0
	fr.fnResults.Items = append(fr.fnResults.Items, *fnResult)
	return output, nil
}

func parseStructuredResult(yml *yaml.RNode, fnResult *fnresult.Result) error {
	if yml.IsNilOrEmpty() {
		return nil
	}
	// Note: TS SDK and Go SDK implements two different formats for the
	// result. Go SDK wraps result items while TS SDK doesn't. So examine
	// if items are wrapped or not to support both the formats for now.
	// Refer to https://github.com/GoogleContainerTools/kpt/pull/1923#discussion_r628604165
	// for some more details.
	if yml.YNode().Kind == yaml.MappingNode {
		// check if legacy structured result wraps ResultItems
		itemsNode, err := yml.Pipe(yaml.Lookup("items"))
		if err != nil {
			return err
		}
		if !itemsNode.IsNilOrEmpty() {
			// if legacy structured result, uplift the items
			yml = itemsNode
		}
	}
	err := yaml.Unmarshal([]byte(yml.MustString()), &fnResult.Results)
	if err != nil {
		return err
	}
	return nil
}

// printFnResult prints given function result in a user friendly
// format on kpt CLI.
func printFnResult(ctx context.Context, fnResult *fnresult.Result) {
	pr := printer.FromContextOrDie(ctx)
	printOpt := printer.NewOpt()
	if len(fnResult.Results) > 0 {
		// function returned structured results
		pr.OptPrintf(printOpt, "  Results:\n")
		var lines []string
		for _, item := range fnResult.Results {
			lines = append(lines, resultToString(item))
		}
		ri := &multiLineFormatter{
			Lines:          lines,
			TruncateOutput: printer.TruncateOutput,
		}
		pr.OptPrintf(printOpt, "%s", ri.String())
	}
}

// printFnExecErr prints given ExecError in a user friendly format
// on kpt CLI.
func printFnExecErr(ctx context.Context, fnErr *ExecError) {
	pr := printer.FromContextOrDie(ctx)
	printOpt := printer.NewOpt()
	if len(fnErr.Stderr) > 0 {
		pr.OptPrintf(printOpt.Stderr(), "  Stderr:\n")
		errLines := &multiLineFormatter{
			Lines:          strings.Split(fnErr.Stderr, "\n"),
			UseQuote:       true,
			TruncateOutput: printer.TruncateOutput,
		}
		pr.OptPrintf(printOpt.Stderr(), "%s", errLines.String())
	}
	pr.OptPrintf(printOpt.Stderr(), "  Exit code: %d\n\n", fnErr.ExitCode)
}

// multiLineFormatter knows how to format multiple lines in pretty format
// that can be displayed to an end user.
type multiLineFormatter struct {
	// Lines to be printed on the CLI.
	Lines []string

	// TruncateOuput determines if output needs to be truncated or not.
	TruncateOutput bool

	// MaxLines to be printed if truncation is enabled.
	MaxLines int

	// UseQuote determines if line needs to be quoted or not
	UseQuote bool
}

// String returns multiline string.
func (ri *multiLineFormatter) String() string {
	ri.MaxLines = FnExecErrorTruncateLines
	strInterpolator := "%s"
	if ri.UseQuote {
		strInterpolator = "%q"
	}

	var b strings.Builder
	lineIndent := strings.Repeat(" ", FnExecErrorIndentation+2)
	if !ri.TruncateOutput {
		// stderr string should have indentations
		for _, s := range ri.Lines {
			// suppress newlines to avoid poor formatting
			s = strings.ReplaceAll(s, "\n", " ")
			b.WriteString(fmt.Sprintf(lineIndent+strInterpolator+"\n", s))
		}
		return b.String()
	}
	printedLines := 0
	for i, s := range ri.Lines {
		if i >= ri.MaxLines {
			break
		}
		// suppress newlines to avoid poor formatting
		s = strings.ReplaceAll(s, "\n", " ")
		b.WriteString(fmt.Sprintf(lineIndent+strInterpolator+"\n", s))
		printedLines++
	}
	truncatedLines := len(ri.Lines) - printedLines
	if truncatedLines > 0 {
		b.WriteString(fmt.Sprintf(lineIndent+"...(%d line(s) truncated, use '--truncate-output=false' to disable)\n", truncatedLines))
	}
	return b.String()
}

// resultToString converts given structured result item to string format.
func resultToString(result framework.ResultItem) string {
	// TODO: Go SDK should implement Stringer method
	// for framework.ResultItem. This is a temporary
	// wrapper that will eventually be moved to Go SDK.

	defaultSeverity := "info"

	s := strings.Builder{}

	severity := defaultSeverity

	if string(result.Severity) != "" {
		severity = string(result.Severity)
	}
	s.WriteString(fmt.Sprintf("[%s] %s", strings.ToUpper(severity), result.Message))

	if result.ResourceRef.Kind != "" {
		// if an object is involved
		s.WriteString(" in object ")
		if result.ResourceRef.APIVersion != "" {
			s.WriteString(fmt.Sprintf("%s/", result.ResourceRef.APIVersion))
		}
		if result.ResourceRef.Kind != "" {
			s.WriteString(fmt.Sprintf("%s/", result.ResourceRef.Kind))
		}
		if result.ResourceRef.Namespace != "" {
			s.WriteString(fmt.Sprintf("%s/", result.ResourceRef.Namespace))
		}
		if result.ResourceRef.Name != "" {
			s.WriteString(result.ResourceRef.Name)
		}
	}

	if result.File.Path != "" {
		s.WriteString(fmt.Sprintf(" in file %q", result.File.Path))
	}

	if result.Field.Path != "" {
		s.WriteString(fmt.Sprintf(" in field %q", result.Field.Path))
	}

	return s.String()
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
