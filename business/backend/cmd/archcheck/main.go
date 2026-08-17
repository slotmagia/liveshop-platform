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

// surfacePattern captures the owning surface of a file or import. Page-backing
// surfaces live under application/, Platform's own control plane under
// controlplane/; neither may reach into the other.
var surfacePattern = regexp.MustCompile(`/internal/(?:application|controlplane)/([^/]+)/`)
var modulePathPattern = regexp.MustCompile(`(?m)^module\s+(\S+)`)

type checker struct {
	root     string
	protocol string
	module   string
	failures []string
	api      map[string]struct{}
}

func main() {
	root := flag.String("root", "..", "business module root")
	protocol := flag.String("protocol", "", "wire contract module root; defaults to <root>/../protocol")
	flag.Parse()
	abs, err := filepath.Abs(*root)
	if err != nil {
		fatal(err)
	}
	contracts := *protocol
	if contracts == "" {
		contracts = filepath.Join(abs, "..", "protocol")
	}
	contractsAbs, err := filepath.Abs(contracts)
	if err != nil {
		fatal(err)
	}
	c := &checker{root: abs, protocol: contractsAbs, api: map[string]struct{}{}}
	c.module = c.goModulePath()
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
		"backend/docs/编码Agent业务开发.md",
		"backend/docs/domain/事实.md",
		"backend/docs/domain/不变量.md",
		"backend/docs/domain/状态机.md",
		"backend/docs/domain/事务.md",
		"backend/docs/domain/外部契约.md",
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
		for _, imported := range parsed.Imports {
			value, _ := strconv.Unquote(imported.Path.Value)
			c.checkImport(slashed, value)
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

// layerRule forbids a set of imports for every file under one path fragment.
// Together the rules keep each directory a change boundary rather than a
// grouping of files.
type layerRule struct {
	scope string
	// contains matches anywhere in the import path; suffix matches the whole
	// package, so a parent package can be forbidden while a child is allowed.
	contains []string
	suffix   []string
	except   []string
	message  string
}

var layerRules = []layerRule{
	{
		scope:    "backend/internal/biz/",
		contains: []string{"github.com/gogf/", "google.golang.org/grpc", "database/sql", "net/http", "/internal/data/", "/internal/application/", "/internal/controlplane/", "/internal/common/"},
		message:  "domain layer imports an outer layer",
	},
	{
		scope:    "backend/internal/data/",
		contains: []string{"github.com/gogf/", "google.golang.org/grpc", "net/http", "/internal/application/", "/internal/controlplane/", "/internal/common/"},
		except:   nil,
		message:  "data layer imports a transport or application package",
	},
	{
		scope:    "/api/http/",
		contains: []string{"/internal/biz", "/internal/data", "database/sql"},
		message:  "wire contract imports an internal layer",
	},
	{
		scope:    "backend/internal/application/",
		contains: []string{"/internal/controlplane/"},
		message:  "a page-backing surface reaches into the control plane",
	},
	{
		scope:    "/logic/",
		contains: []string{"database/sql", "github.com/go-sql-driver/mysql", "github.com/jackc/pgx", "google.golang.org/grpc", "github.com/gogf/", "/controller/", "/common/web"},
		message:  "application logic imports a transport",
	},
	{
		scope:    "/service/",
		contains: []string{"database/sql", "github.com/gogf/", "google.golang.org/grpc", "/api/http/", "/controller/"},
		message:  "application boundary imports a transport",
	},
	{
		scope:    "/controller/http/",
		contains: []string{"google.golang.org/grpc", "database/sql", "github.com/go-sql-driver/mysql", "github.com/jackc/pgx", "/internal/data/"},
		suffix:   []string{"/internal/biz"},
		message:  "HTTP controller imports another transport, storage or a use case",
	},
	{
		scope:    "/controller/grpc/",
		contains: []string{"github.com/gogf/", "/common/web", "/api/http/", "database/sql", "github.com/go-sql-driver/mysql", "github.com/jackc/pgx", "/internal/data/"},
		suffix:   []string{"/internal/biz"},
		message:  "gRPC controller imports the HTTP contract, storage or a use case",
	},
	{
		scope:    "/router/",
		contains: []string{"database/sql", "github.com/go-sql-driver/mysql", "github.com/jackc/pgx", "/internal/data/"},
		suffix:   []string{"/internal/biz"},
		message:  "router reaches past the application boundary",
	},
	{
		scope:    "backend/internal/common/",
		contains: []string{"/internal/application/", "/internal/controlplane/"},
		message:  "shared transport code depends on a surface",
	},
}

func (c *checker) checkImport(file, imported string) {
	for _, rule := range layerRules {
		if !strings.Contains(file, rule.scope) || containsAny(file, rule.except) {
			continue
		}
		for _, forbidden := range rule.contains {
			if strings.Contains(imported, forbidden) {
				c.add(fmt.Sprintf("%s: %s (%s)", file, rule.message, imported))
			}
		}
		for _, forbidden := range rule.suffix {
			if strings.HasSuffix(imported, forbidden) {
				c.add(fmt.Sprintf("%s: %s (%s)", file, rule.message, imported))
			}
		}
	}
	c.checkSurfaceIsolation(file, imported)
	if strings.Contains(imported, "/internal/") && c.module != "" && !strings.HasPrefix(imported, c.module+"/internal/") {
		c.add(fmt.Sprintf("%s: imports another module's internal package %s", file, imported))
	}
}

// checkSurfaceIsolation keeps admin, auth, provisioning and runtime from
// reusing each other's private contracts and logic.
func (c *checker) checkSurfaceIsolation(file, imported string) {
	source := surfacePattern.FindStringSubmatch("/" + file)
	target := surfacePattern.FindStringSubmatch(imported)
	if len(source) == 2 && len(target) == 2 && source[1] != target[1] {
		c.add(fmt.Sprintf("%s: surface %s imports surface %s (%s)", file, source[1], target[1], imported))
	}
}

func (c *checker) goModulePath() string {
	data, err := os.ReadFile(filepath.Join(c.root, "backend", "go.mod"))
	if err != nil {
		c.add("cannot read backend/go.mod: " + err.Error())
		return ""
	}
	if match := modulePathPattern.FindSubmatch(data); len(match) == 2 {
		return string(match[1])
	}
	c.add("backend/go.mod does not declare a module path")
	return ""
}

func containsAny(value string, fragments []string) bool {
	for _, fragment := range fragments {
		if strings.Contains(value, fragment) {
			return true
		}
	}
	return false
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
		generated, _ := os.ReadFile(filepath.Join(c.protocol, "gen", "go", "platform", "v1", "platform_grpc.pb.go"))
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
