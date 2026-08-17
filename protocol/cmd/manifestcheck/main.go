// Command manifestcheck validates release manifests during local development.
// Development artifact placeholders are replaced only in-memory; the runtime
// registry still accepts exclusively real SHA-256 digests.
package main

import (
	"fmt"
	"os"
	"regexp"

	"github.com/liveshop-platform/contracts/modulemanifest"
)

var developmentDigest = regexp.MustCompile(`sha256:dev-[a-z0-9-]+`)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: manifestcheck <module.json> [...]")
		os.Exit(2)
	}
	for _, path := range os.Args[1:] {
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		data = developmentDigest.ReplaceAll(data, []byte("sha256:0000000000000000000000000000000000000000000000000000000000000000"))
		manifest, err := modulemanifest.Decode(data)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", path, err)
			os.Exit(1)
		}
		fmt.Printf("%s: %s@%s capability contract is valid\n", path, manifest.Metadata.ID, manifest.Metadata.Version)
	}
}
