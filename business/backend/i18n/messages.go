package i18n

import (
	"embed"
	"strings"
	"sync"

	"github.com/BurntSushi/toml"
)

//go:embed *.toml
var files embed.FS

type file struct {
	Errors map[string]string `toml:"errors"`
}

var (
	once sync.Once
	data map[string]map[string]string
)

func Lookup(locale, reason string) string {
	once.Do(func() {
		data = map[string]map[string]string{}
		for _, name := range []string{"zh-CN.toml", "en-US.toml"} {
			raw, err := files.ReadFile(name)
			if err != nil {
				continue
			}
			var parsed file
			if _, err := toml.Decode(string(raw), &parsed); err != nil {
				continue
			}
			data[strings.TrimSuffix(name, ".toml")] = parsed.Errors
		}
	})
	if table := data[locale]; table != nil {
		return strings.TrimSpace(table[reason])
	}
	return ""
}
