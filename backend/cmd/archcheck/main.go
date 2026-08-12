// Command archcheck enforces repository architecture rules that coding agents
// and human contributors must satisfy before a change can be integrated.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/liveshop-platform/contracts/modulemanifest"
)

var metaPattern = regexp.MustCompile(`path:"([^"]+)"\s+method:"([^"]+)"`)
var pathParameterPattern = regexp.MustCompile(`\{[^/{}]+\}`)
var innerHTMLPattern = regexp.MustCompile("(?s)innerHTML\\s*=\\s*`([^`]*)`")
var serverInterpolationPattern = regexp.MustCompile(`\$\{\s*(item|release|account|event|document)\.`)

type checker struct {
	root     string
	failures []string
	api      map[string]struct{}
}

func main() {
	root := flag.String("root", "..", "repository root")
	flag.Parse()
	abs, err := filepath.Abs(*root)
	if err != nil {
		fatal(err)
	}
	c := &checker{root: abs, api: map[string]struct{}{}}
	c.requiredFiles()
	c.goImports()
	c.contracts()
	c.frontendSafety()
	if len(c.failures) > 0 {
		sort.Strings(c.failures)
		for _, failure := range c.failures {
			fmt.Fprintln(os.Stderr, "ARCH:", failure)
		}
		os.Exit(1)
	}
	fmt.Println("Architecture checks passed.")
}

func (c *checker) requiredFiles() {
	for _, relative := range []string{
		"AGENTS.md",
		"backend/AGENTS.md",
		"backend/docs/AGENT-BUSINESS-DEVELOPMENT.md",
		"backend/docs/domain/FACTS.md",
		"backend/docs/domain/INVARIANTS.md",
		"backend/docs/domain/STATE-MACHINE.md",
		"backend/docs/domain/TRANSACTIONS.md",
		"backend/docs/domain/EXTERNAL-CONTRACTS.md",
	} {
		if _, err := os.Stat(filepath.Join(c.root, filepath.FromSlash(relative))); err != nil {
			c.add("required engineering contract is missing: " + relative)
		}
	}
}

func (c *checker) goImports() {
	internal := filepath.Join(c.root, "backend", "internal")
	_ = filepath.WalkDir(internal, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			c.add(walkErr.Error())
			return nil
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		relative, _ := filepath.Rel(c.root, path)
		slashed := filepath.ToSlash(relative)
		parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ParseComments)
		if err != nil {
			c.add(fmt.Sprintf("%s: parse: %v", slashed, err))
			return nil
		}
		sourceModule := internalModule(slashed)
		for _, imported := range parsed.Imports {
			value, _ := strconv.Unquote(imported.Path.Value)
			c.checkImport(slashed, sourceModule, value)
		}
		ast.Inspect(parsed, func(node ast.Node) bool {
			field, ok := node.(*ast.Field)
			if !ok || field.Tag == nil {
				return true
			}
			tag, _ := strconv.Unquote(field.Tag.Value)
			if match := metaPattern.FindStringSubmatch(tag); len(match) == 3 {
				c.api[strings.ToUpper(match[2])+" "+match[1]] = struct{}{}
			}
			return true
		})
		return nil
	})
}

func (c *checker) checkImport(file, sourceModule, imported string) {
	if strings.Contains(file, "/domain/") {
		for _, forbidden := range []string{"github.com/gogf/", "google.golang.org/grpc", "database/sql", "net/http", "/infrastructure/", "/transport/"} {
			if strings.Contains(imported, forbidden) {
				c.add(fmt.Sprintf("%s: domain imports %s", file, imported))
			}
		}
	}
	if strings.Contains(file, "/transport/") || strings.Contains(file, "/controller/") {
		for _, forbidden := range []string{"database/sql", "github.com/jackc/pgx", "/infrastructure/postgres", "/registry/"} {
			if strings.Contains(imported, forbidden) {
				c.add(fmt.Sprintf("%s: transport/controller bypasses application boundary through %s", file, imported))
			}
		}
	}
	if strings.Contains(file, "/logic/") {
		for _, forbidden := range []string{"database/sql", "github.com/jackc/pgx", "google.golang.org/grpc", "github.com/gogf/gf/v2/net/ghttp", "/controller/"} {
			if strings.Contains(imported, forbidden) {
				c.add(fmt.Sprintf("%s: application logic imports %s", file, imported))
			}
		}
	}
	marker := "/internal/"
	if index := strings.Index(imported, marker); index >= 0 {
		target := strings.Split(strings.TrimPrefix(imported[index+len(marker):], "/"), "/")[0]
		if sourceModule != "" && target != "" && target != sourceModule {
			c.add(fmt.Sprintf("%s: module %s imports another module internal package %s", file, sourceModule, imported))
		}
	}
}

func (c *checker) contracts() {
	manifestPath := filepath.Join(c.root, "module.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		c.add("cannot read module.json: " + err.Error())
		return
	}
	var manifest modulemanifest.Manifest
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		c.add("invalid module.json: " + err.Error())
		return
	}
	validated := manifest
	for index := range validated.Spec.Contributions {
		if strings.HasPrefix(validated.Spec.Contributions[index].Artifact.Integrity, "sha256:dev-") {
			validated.Spec.Contributions[index].Artifact.Integrity = "sha256:" + strings.Repeat("0", 64)
		}
	}
	if err := validated.Validate(); err != nil {
		c.add("invalid module.json: " + err.Error())
		return
	}
	for _, route := range manifest.Spec.Backend.HTTPRoutes {
		for _, operation := range route.Operations {
			if !c.hasHTTPBinding(operation.Method, operation.Path) {
				c.add(fmt.Sprintf("manifest operation %s has no matching GoFrame g.Meta binding (%s %s)", operation.ID, operation.Method, operation.Path))
			}
		}
	}
	for i, left := range manifest.Spec.Backend.HTTPRoutes {
		for _, right := range manifest.Spec.Backend.HTTPRoutes[i+1:] {
			if left.Surface == right.Surface && prefixesOverlap(left.Prefix, right.Prefix) {
				c.add(fmt.Sprintf("manifest routes overlap on %s: %s and %s", left.Surface, left.Prefix, right.Prefix))
			}
		}
	}
	if manifest.Spec.Backend.GRPC != nil {
		generated, _ := os.ReadFile(filepath.Join(c.root, "backend", "contracts", "gen", "go", "platform", "v1", "platform_grpc.pb.go"))
		for _, method := range manifest.Spec.Backend.GRPC.Methods {
			if !strings.Contains(string(generated), strconv.Quote(method.FullMethod)) {
				c.add("manifest gRPC method is absent from generated contract: " + method.FullMethod)
			}
		}
	}
}

func (c *checker) hasHTTPBinding(method, fullPath string) bool {
	method = strings.ToUpper(method)
	for binding := range c.api {
		parts := strings.SplitN(binding, " ", 2)
		if len(parts) == 2 && parts[0] == method && pathSuffix(fullPath, parts[1]) {
			return true
		}
	}
	return false
}

func (c *checker) frontendSafety() {
	root := filepath.Join(c.root, "frontend-admin", "src")
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".ts") {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		for _, assignment := range innerHTMLPattern.FindAllSubmatch(data, -1) {
			if serverInterpolationPattern.Match(assignment[1]) {
				relative, _ := filepath.Rel(c.root, path)
				c.add(filepath.ToSlash(relative) + ": server-derived data must use textContent/value instead of innerHTML")
				break
			}
		}
		return nil
	})
}

func internalModule(path string) string {
	parts := strings.Split(filepath.ToSlash(path), "/")
	for i, part := range parts {
		if part == "internal" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

func pathSuffix(full, relative string) bool {
	full = pathParameterPattern.ReplaceAllString(strings.TrimRight(full, "/"), "{}")
	relative = pathParameterPattern.ReplaceAllString(strings.TrimRight(relative, "/"), "{}")
	return full == relative || strings.HasSuffix(full, relative)
}

func prefixesOverlap(left, right string) bool {
	left, right = strings.TrimRight(left, "/"), strings.TrimRight(right, "/")
	return left == right || strings.HasPrefix(left, right+"/") || strings.HasPrefix(right, left+"/")
}

func (c *checker) add(message string) { c.failures = append(c.failures, message) }

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
