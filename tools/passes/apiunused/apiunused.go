package apiunused

import (
	"encoding/json"
	"io"
	"os"
	"reflect"

	"golang.org/x/tools/go/analysis"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/tools/passes"
	clientgo "gitlab.com/gitlab-org/terraform-provider-gitlab/tools/passes/clientgo"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/tools/passes/usage"
)

var Output io.Writer = io.Discard

var Analyzer = &analysis.Analyzer{
	Doc:        "Estimate the unused pieces of the client-go package",
	Name:       "apiunused",
	ResultType: reflect.TypeOf((*Result)(nil)),
	Requires:   []*analysis.Analyzer{clientgo.Analyzer, usage.Analyzer},
	Run:        run,
}

type Result struct {
	UnusedByFile map[string]Unused
}

type Unused struct {
	Types   []string `json:"types,omitempty"`
	Funcs   []string `json:"funcs,omitempty"`
	Methods []string `json:"methods,omitempty"`
	Fields  []string `json:"fields,omitempty"`
}

func run(pass *analysis.Pass) (interface{}, error) {
	clientGo := pass.ResultOf[clientgo.Analyzer].(*clientgo.Result)
	usage := pass.ResultOf[usage.Analyzer].(*usage.Result)

	result := &Result{
		UnusedByFile: make(map[string]Unused),
	}

	processUnused(result, clientGo.TypeToFilenames, usage.Types,
		func(item Unused, name string) Unused {
			item.Types = append(item.Types, name)
			return item
		})

	processUnused(result, clientGo.FuncToFilenames, usage.Funcs,
		func(item Unused, name string) Unused {
			item.Funcs = append(item.Funcs, name)
			return item
		})

	processUnused(result, clientGo.MethodToFilenames, usage.Methods,
		func(item Unused, name string) Unused {
			item.Methods = append(item.Methods, name)
			return item
		})

	processUnused(result, clientGo.FieldToFilenames, usage.Fields,
		func(item Unused, name string) Unused {
			item.Fields = append(item.Fields, name)
			return item
		})

	if !passes.IsTestPackage(pass) {
		writeOutput(result)
	}

	return result, nil
}

func processUnused(result *Result, nameToFilenames clientgo.MultiMap, seen usage.Set, mutFn func(Unused, string) Unused) {
	for name, filenames := range nameToFilenames {
		if !seen[name] {
			for _, filename := range filenames {
				result.UnusedByFile[filename] = mutFn(result.UnusedByFile[filename], name)
			}
		}
	}
}

func writeOutput(result *Result) {
	e := json.NewEncoder(os.Stdout)
	e.SetIndent("", "  ")

	_ = e.Encode(result.UnusedByFile)
}
