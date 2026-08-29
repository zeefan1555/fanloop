package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/cloudwego/thriftgo/parser"
)

const generatedPackageRoot = "github.com/zeefan1555/commonloop/internal/idl"

type command struct {
	id, summary, pkg, request, response, risk, scope string
	dryRun                                           bool
	errors                                           []string
}

func main() {
	input := flag.String("input", "idl/cli.thrift", "aggregate Thrift file")
	output := flag.String("output", "internal/idl/commands_gen.go", "generated Go file")
	flag.Parse()
	root, err := parser.ParseFile(*input, []string{filepath.Dir(*input)}, true)
	if err != nil {
		fatal(err)
	}
	commands, err := collect(root, errorCodes(root))
	if err != nil {
		fatal(err)
	}
	content, err := render(commands)
	if err != nil {
		fatal(err)
	}
	if err := os.WriteFile(*output, content, 0o644); err != nil {
		fatal(err)
	}
}

func collect(root *parser.Thrift, knownErrors map[string]bool) ([]command, error) {
	seenFiles := map[string]bool{}
	seenCommands := map[string]bool{}
	var commands []command
	var walk func(*parser.Thrift) error
	walk = func(current *parser.Thrift) error {
		if current == nil || seenFiles[current.Filename] {
			return nil
		}
		seenFiles[current.Filename] = true
		pkg := namespace(current)
		for _, service := range current.Services {
			for _, function := range service.Functions {
				annotations, err := annotationMap(function.Annotations)
				if err != nil {
					return fmt.Errorf("%s.%s: %w", service.Name, function.Name, err)
				}
				item, err := commandFrom(pkg, function, annotations, knownErrors)
				if err != nil {
					return fmt.Errorf("%s.%s: %w", service.Name, function.Name, err)
				}
				if seenCommands[item.id] {
					return fmt.Errorf("duplicate cli.id %q", item.id)
				}
				seenCommands[item.id] = true
				commands = append(commands, item)
			}
		}
		for _, include := range current.Includes {
			if err := walk(include.Reference); err != nil {
				return err
			}
		}
		return nil
	}
	if err := walk(root); err != nil {
		return nil, err
	}
	if len(commands) != 11 {
		return nil, fmt.Errorf("found %d public commands, want 11", len(commands))
	}
	return commands, nil
}

func commandFrom(pkg string, function *parser.Function, annotations map[string]string, knownErrors map[string]bool) (command, error) {
	for _, key := range []string{"cli.id", "cli.summary", "cli.risk", "cli.requirement_scope", "cli.supports_dry_run", "cli.errors"} {
		if annotations[key] == "" {
			return command{}, fmt.Errorf("missing %s annotation", key)
		}
	}
	risk := annotations["cli.risk"]
	if risk != "read" && risk != "local_write" && risk != "external_write" {
		return command{}, fmt.Errorf("invalid cli.risk %q", risk)
	}
	scope := annotations["cli.requirement_scope"]
	if scope != "none" && scope != "new" && scope != "existing" && scope != "optional" {
		return command{}, fmt.Errorf("invalid cli.requirement_scope %q", scope)
	}
	dryRun, err := strconv.ParseBool(annotations["cli.supports_dry_run"])
	if err != nil {
		return command{}, fmt.Errorf("invalid cli.supports_dry_run: %w", err)
	}
	request := argument(function, "request")
	root := argument(function, "requirement_root")
	dry := argument(function, "dry_run")
	if request == nil || request.Type == nil {
		return command{}, fmt.Errorf("request argument is required")
	}
	if (scope == "none") != (root == nil) {
		return command{}, fmt.Errorf("requirement_root contradicts scope %q", scope)
	}
	if dryRun != (dry != nil) {
		return command{}, fmt.Errorf("dry_run argument contradicts supports_dry_run=%t", dryRun)
	}
	if function.FunctionType == nil || function.FunctionType.Name == "" {
		return command{}, fmt.Errorf("response type is required")
	}
	if len(function.Throws) != 1 || function.Throws[0].Type == nil || function.Throws[0].Type.Name != "error.PublicError" {
		return command{}, fmt.Errorf("method must throw error.PublicError")
	}
	errors := splitCSV(annotations["cli.errors"])
	for _, code := range errors {
		if !knownErrors[code] {
			return command{}, fmt.Errorf("unknown error code %q", code)
		}
	}
	return command{
		id: annotations["cli.id"], summary: annotations["cli.summary"], pkg: pkg,
		request: request.Type.Name, response: function.FunctionType.Name,
		risk: risk, scope: scope, dryRun: dryRun, errors: errors,
	}, nil
}

func errorCodes(root *parser.Thrift) map[string]bool {
	result := map[string]bool{}
	seen := map[string]bool{}
	var walk func(*parser.Thrift)
	walk = func(current *parser.Thrift) {
		if current == nil || seen[current.Filename] {
			return
		}
		seen[current.Filename] = true
		for _, enum := range current.Enums {
			if enum.Name == "ErrorCode" {
				for _, value := range enum.Values {
					if value.Name != "unspecified" {
						result[value.Name] = true
					}
				}
			}
		}
		for _, include := range current.Includes {
			walk(include.Reference)
		}
	}
	walk(root)
	return result
}

func render(commands []command) ([]byte, error) {
	packages := map[string]bool{"commonidl": true, "erroridl": true}
	for _, item := range commands {
		packages[item.pkg] = true
	}
	aliases := make([]string, 0, len(packages))
	for alias := range packages {
		aliases = append(aliases, alias)
	}
	sort.Strings(aliases)
	var out bytes.Buffer
	fmt.Fprintln(&out, "// Code generated by tools/command-spec-gen. DO NOT EDIT.")
	fmt.Fprintln(&out, "package idl")
	fmt.Fprintln(&out, "import (")
	fmt.Fprintln(&out, "\t\"reflect\"")
	for _, alias := range aliases {
		fmt.Fprintf(&out, "\t%q\n", generatedPackageRoot+"/"+alias)
	}
	fmt.Fprintln(&out, ")")
	fmt.Fprintln(&out, "var commandSpecs = []CommandSpec{")
	for _, item := range commands {
		fmt.Fprintf(&out, "\t{ID: %q, Summary: %q, RequestType: reflect.TypeFor[%s.%s](), ResponseType: reflect.TypeFor[%s.%s](), Risk: commonidl.CommandRisk_%s, RequirementScope: commonidl.RequirementScope_%s, SupportsDryRun: %t, Errors: errorSpecs(", item.id, item.summary, item.pkg, item.request, item.pkg, item.response, item.risk, item.scope, item.dryRun)
		for index, code := range item.errors {
			if index > 0 {
				out.WriteString(", ")
			}
			fmt.Fprintf(&out, "erroridl.ErrorCode_%s", code)
		}
		fmt.Fprintln(&out, ")},")
	}
	fmt.Fprintln(&out, "}")
	return format.Source(out.Bytes())
}

func annotationMap(values parser.Annotations) (map[string]string, error) {
	result := map[string]string{}
	for _, value := range values {
		if len(value.Values) != 1 {
			return nil, fmt.Errorf("annotation %q must have exactly one value", value.Key)
		}
		if _, exists := result[value.Key]; exists {
			return nil, fmt.Errorf("annotation %q is duplicated", value.Key)
		}
		result[value.Key] = value.Values[0]
	}
	return result, nil
}

func namespace(current *parser.Thrift) string {
	parts := strings.Split(current.GetNamespaceOrReferenceName("go"), ".")
	return strings.ToLower(parts[len(parts)-1])
}

func argument(function *parser.Function, name string) *parser.Field {
	for _, value := range function.Arguments {
		if value.Name == name {
			return value
		}
	}
	return nil
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, item := range parts {
		if item = strings.TrimSpace(item); item != "" {
			result = append(result, item)
		}
	}
	return result
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
