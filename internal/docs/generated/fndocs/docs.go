// Code generated by "mdtogo"; DO NOT EDIT.
package fndocs

var FnShort = `Transform and validate packages using containerized functions.`
var FnLong = `
The ` + "`" + `fn` + "`" + ` command group contains subcommands for transforming and validating ` + "`" + `kpt` + "`" + ` packages
using containerized functions.
`

var DocShort = `Display the documentation for a function`
var DocLong = `
` + "`" + `kpt fn doc` + "`" + ` invokes the function container with ` + "`" + `--help` + "`" + ` flag.

  kpt fn doc --image=IMAGE

--image is a required flag.
If the function supports --help, it will print the documentation to STDOUT.
Otherwise, it will exit with non-zero exit code and print the error message to
STDERR.
`
var DocExamples = `
  # diplay the documentation for image gcr.io/kpt-fn/set-namespace:v0.1.1
  kpt fn doc --image gcr.io/kpt-fn/set-namespace:v0.1.1
`

var EvalShort = `Execute function on resources`
var EvalLong = `
  kpt fn eval [DIR|-] [flags]

Args:

  DIR|-:
    Path to the local directory containing resources. Defaults to the current
    working directory. Using '-' as the directory path will cause ` + "`" + `eval` + "`" + ` to
    read resources from ` + "`" + `stdin` + "`" + ` and write the output to ` + "`" + `stdout` + "`" + `.

Flags:

  --image:
    Container image of the function to execute e.g. ` + "`" + `gcr.io/kpt-fn/set-namespace:v0.1` + "`" + `
  
  --exec-path:
    Path to the local executable binary to execute as a function.
    
  --fn-config:
    Path to the file containing ` + "`" + `functionConfig` + "`" + ` for the function.
  
  --include-meta-resources:
    If enabled, meta resources (i.e. ` + "`" + `Kptfile` + "`" + ` and ` + "`" + `functionConfig` + "`" + `) are included
    in the input to the function. By default it is disabled.
  
  --network:
    If enabled, container functions are allowed to access network.
    By default is it disabled.
  
  --mount:
    List of storage options to enable reading from the local filesytem. By default,
    container functions can not access the local filesystem. It accepts the same options
    as specified on the [Docker Volumes] for ` + "`" + `docker run` + "`" + `. All volumes are mounted
    readonly by default. Specify ` + "`" + `rw=true` + "`" + ` to mount volumes in read-write mode.
  
  --env:
    List of local environment variables to be exported to the container function.
    By default, none of local environment variables are made available to the
    container running the function. The value can be in ` + "`" + `key=value` + "`" + ` format or only
    the key of an already exported environment variable.
  
  --as-current-user:
    Use the ` + "`" + `uid` + "`" + ` and ` + "`" + `gid` + "`" + ` of the kpt process for container function execution.
  
  --results-dir:
    Path to a directory to write structured results. Directory must exist.
    Structured results emitted by the functions are aggregated and saved
    to ` + "`" + `results.yaml` + "`" + ` file in the specified directory.
    If not specified, no result files are written to the local filesystem.
    
  --dry-run:
    If enabled, the resources are not written to local filesystem, instead they
    are written to stdout. By defaults it is disabled.
    
`
var EvalExamples = `
  # execute container my-fn on the resources in DIR directory and
  # write output back to DIR
  $ kpt fn --image gcr.io/example.com/my-fn eval DIR

  # execute container my-fn on the resources in DIR directory with
  # ` + "`" + `functionConfig` + "`" + ` my-fn-config
  $ kpt fn --image gcr.io/example.com/my-fn --fn-config my-fn-config eval DIR

  # execute container my-fn with an input ConfigMap containing ` + "`" + `data: {foo: bar}` + "`" + `
  $ kpt fn --image gcr.io/example.com/my-fn:v1.0.0 eval DIR -- foo=bar

  # execute executable my-fn on the resources in DIR directory and
  # write output back to DIR
  $ kpt fn --exec-path ./my-fn eval DIR

  # execute container my-fn on the resources in DIR directory,
  # save structured results in /tmp/my-results dir and write output back to DIR
  $ kpt fn --image gcr.io/example.com/my-fn --results-dir /tmp/my-results-dir eval DIR

  # execute container my-fn on the resources in DIR directory with network access enabled,
  # and write output back to DIR
  $ kpt fn --image gcr.io/example.com/my-fn --network eval DIR

  # execute container my-fn on the resource in DIR and export KUBECONFIG
  # and foo environment variable
  $ kpt fn --image gcr.io/example.com/my-fn --env KUBECONFIG -e foo=bar eval DIR
`

var ExportShort = `Auto-generating function pipelines for different workflow orchestrators`
var ExportLong = `
  kpt fn export DIR/ [--fn-path FUNCTIONS_DIR/] --workflow ORCHESTRATOR [--output OUTPUT_FILENAME]
  
  DIR:
    Path to a package directory.
  FUNCTIONS_DIR:
    Read functions from the directory instead of the DIR/.
  ORCHESTRATOR:
    Supported orchestrators are:
      - github-actions
      - cloud-build
      - gitlab-ci
      - jenkins
      - tekton
      - circleci
  OUTPUT_FILENAME:
    Specifies the filename of the generated pipeline. If omitted, the default
    output is stdout
`
var ExportExamples = `
  # read functions from DIR, run them against it as one step.
  # write the generated GitHub Actions pipeline to main.yaml.
  kpt fn export DIR/ --output main.yaml --workflow github-actions

  # discover functions in FUNCTIONS_DIR and run them against resource in DIR.
  # write the generated Cloud Build pipeline to stdout.
  kpt fn export DIR/ --fn-path FUNCTIONS_DIR/ --workflow cloud-build
`

var RenderShort = `Render a package.`
var RenderLong = `
  kpt fn render [PKG_PATH] [flags]

Args:

  PKG_PATH:
    Local package path to render. Directory must exist and contain a Kptfile
    to be updated. Defaults to the current working directory.

Flags:

  --results-dir:
    Path to a directory to write structured results. Directory must exist.
    Structured results emitted by the functions are aggregated and saved
    to ` + "`" + `results.yaml` + "`" + ` file in the specified directory.
    If not specified, no result files are written to the local filesystem.
`
var RenderExamples = `
  # Render the package in current directory
  $ kpt fn render

  # Render the package in current directory and save results in my-results-dir
  $ kpt fn --results-dir my-results-dir render

  # Render my-package-dir
  $ kpt fn render my-package-dir
`

var SinkShort = `Specify a directory as an output sink package`
var SinkLong = `
  kpt fn sink [DIR]
  
  DIR:
    Path to a package directory.  Defaults to stdout if unspecified.
`
var SinkExamples = `
  # run a function using explicit sources and sinks
  kpt fn source DIR/ |
    kpt fn run --image gcr.io/example.com/my-fn |
    kpt fn sink DIR/
`

var SourceShort = `Specify a directory as an input source package`
var SourceLong = `
  kpt fn source [DIR...]
  
  DIR:
    Path to a package directory.  Defaults to stdin if unspecified.
`
var SourceExamples = `
  # print to stdout configuration from DIR/ formatted as an input source
  kpt fn source DIR/

  # run a function using explicit sources and sinks
  kpt fn source DIR/ |
    kpt fn run --image gcr.io/example.com/my-fn |
    kpt fn sink DIR/
`
