// Command manifestcompose splits and composes module manifests so independent
// contributors do not need to edit one shared JSON document.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/liveshop-platform/contracts/modulemanifest"
)

var safeName = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

func main() {
	mode := flag.String("mode", "check", "split, compose, or check")
	input := flag.String("input", "../../module.json", "manifest JSON to split or compare")
	source := flag.String("source", "manifest/platform", "fragment directory")
	output := flag.String("output", "../../module.json", "composed manifest output")
	flag.Parse()

	var err error
	switch *mode {
	case "split":
		err = split(*input, *source)
	case "compose":
		err = composeFile(*source, *output)
	case "check":
		err = check(*source, *input)
	default:
		err = fmt.Errorf("unsupported mode %q", *mode)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "manifestcompose:", err)
		os.Exit(1)
	}
}

func split(input, source string) error {
	data, err := os.ReadFile(input)
	if err != nil {
		return err
	}
	var manifest modulemanifest.Manifest
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(source, "routes"), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(source, "contributions"), 0o755); err != nil {
		return err
	}
	routes := manifest.Spec.Backend.HTTPRoutes
	permissions := manifest.Spec.Permissions
	contributions := manifest.Spec.Contributions
	manifest.Spec.Backend.HTTPRoutes = nil
	manifest.Spec.Permissions = nil
	manifest.Spec.Contributions = nil
	if err := writeJSON(filepath.Join(source, "base.json"), manifest); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(source, "permissions.json"), permissions); err != nil {
		return err
	}
	for index, route := range routes {
		name := fmt.Sprintf("%03d-%s-%s.json", index+1, route.Surface, slug(route.Prefix))
		if err := writeJSON(filepath.Join(source, "routes", name), route); err != nil {
			return err
		}
	}
	for _, contribution := range contributions {
		if err := writeJSON(filepath.Join(source, "contributions", slug(contribution.ID)+".json"), contribution); err != nil {
			return err
		}
	}
	return nil
}

func compose(source string) (modulemanifest.Manifest, []byte, error) {
	var manifest modulemanifest.Manifest
	if err := readJSON(filepath.Join(source, "base.json"), &manifest); err != nil {
		return manifest, nil, err
	}
	if err := readJSON(filepath.Join(source, "permissions.json"), &manifest.Spec.Permissions); err != nil {
		return manifest, nil, err
	}
	routeFiles, err := filepath.Glob(filepath.Join(source, "routes", "*.json"))
	if err != nil {
		return manifest, nil, err
	}
	sort.Strings(routeFiles)
	for _, path := range routeFiles {
		var route modulemanifest.HTTPRoute
		if err := readJSON(path, &route); err != nil {
			return manifest, nil, err
		}
		manifest.Spec.Backend.HTTPRoutes = append(manifest.Spec.Backend.HTTPRoutes, route)
	}
	contributionFiles, err := filepath.Glob(filepath.Join(source, "contributions", "*.json"))
	if err != nil {
		return manifest, nil, err
	}
	sort.Strings(contributionFiles)
	for _, path := range contributionFiles {
		var contribution modulemanifest.Contribution
		if err := readJSON(path, &contribution); err != nil {
			return manifest, nil, err
		}
		manifest.Spec.Contributions = append(manifest.Spec.Contributions, contribution)
	}
	validated := manifest
	for index := range validated.Spec.Contributions {
		if strings.HasPrefix(validated.Spec.Contributions[index].Artifact.Integrity, "sha256:dev-") {
			validated.Spec.Contributions[index].Artifact.Integrity = "sha256:" + strings.Repeat("0", 64)
		}
	}
	if err := validated.Validate(); err != nil {
		return manifest, nil, err
	}
	encoded, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return manifest, nil, err
	}
	return manifest, append(encoded, '\n'), nil
}

func composeFile(source, output string) error {
	_, encoded, err := compose(source)
	if err != nil {
		return err
	}
	return os.WriteFile(output, encoded, 0o644)
}

func check(source, input string) error {
	_, expected, err := compose(source)
	if err != nil {
		return err
	}
	actual, err := os.ReadFile(input)
	if err != nil {
		return err
	}
	var compactExpected, compactActual bytes.Buffer
	if err := json.Compact(&compactExpected, expected); err != nil {
		return err
	}
	if err := json.Compact(&compactActual, actual); err != nil {
		return err
	}
	if !bytes.Equal(compactExpected.Bytes(), compactActual.Bytes()) {
		return fmt.Errorf("%s differs from composed fragments in %s; run manifestcompose -mode compose", input, source)
	}
	fmt.Println("Module manifest fragments match module.json.")
	return nil
}

func readJSON(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}

func writeJSON(path string, value any) error {
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(encoded, '\n'), 0o644)
}

func slug(value string) string {
	value = strings.Trim(safeName.ReplaceAllString(value, "-"), "-.")
	if value == "" {
		return "root"
	}
	return strings.ToLower(value)
}
