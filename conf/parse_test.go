package conf

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func mustAbs(s string) string {
	f, err := filepath.Abs(s)
	if err != nil {
		panic(err)
	}
	return f
}

var parseTests = []struct {
	path     string
	input    string
	expected *Config
}{
	{
		"",
		"",
		&Config{},
	},
	{
		"",
		"[[block]]",
		&Config{
			Blocks: []Block{
				{},
			},
		},
	},
	{
		"",
		`[[block]]
include=["foo"]
`,
		&Config{
			Blocks: []Block{
				{
					Include: []string{"foo"},
				},
			},
		},
	},
	{
		"",
		`[[block]]
include=["foo", "bar"]`,
		&Config{
			Blocks: []Block{
				{
					Include: []string{"foo", "bar"},
				},
			},
		},
	},
	{
		"",
		`[[block]]
exclude=["foo"]`,
		&Config{
			Blocks: []Block{
				{
					Exclude: []string{"foo"},
				},
			},
		},
	},
	{
		"",
		`[[block]]
exclude=["foo", "bar", "voing"]`,
		&Config{
			Blocks: []Block{
				{Exclude: []string{"foo", "bar", "voing"}},
			},
		},
	},
	{
		"",
		`[[block]]
include=["foo"]
noignore = true`,
		&Config{
			Blocks: []Block{
				{
					Include:        []string{"foo"},
					NoCommonFilter: true,
				},
			},
		},
	},
	{
		"",
		`[[block]]
include=["foo"]
[[block.daemon]]
cmd = "command"`,
		&Config{
			Blocks: []Block{
				{
					Include: []string{"foo"},
					Daemons: []Daemon{{"command", Signal{}}},
				},
			},
		},
	},
	{
		"",
		`[[block]]
[[block.daemon]]
cmd = "c"
signal = "sighup"`,
		&Config{
			Blocks: []Block{
				{Daemons: []Daemon{{"c", Signal{syscall.SIGHUP}}}},
			},
		},
	},
	{
		"",
		`[[block]]
[[block.daemon]]
cmd = "c"
signal = "sigterm"`,
		&Config{Blocks: []Block{{Daemons: []Daemon{{"c", Signal{syscall.SIGTERM}}}}}},
	},
	{
		"",
		`[[block]]
[[block.daemon]]
cmd = "c"
signal = "sigint"`,
		&Config{Blocks: []Block{{Daemons: []Daemon{{"c", Signal{syscall.SIGINT}}}}}},
	},
	{
		"",
		`[[block]]
[[block.daemon]]
cmd = "c"
signal = "sigkill"`,
		&Config{Blocks: []Block{{Daemons: []Daemon{{"c", Signal{syscall.SIGKILL}}}}}},
	},
	{
		"",
		`[[block]]
[[block.daemon]]
cmd = "c"
signal = "sigquit"`,
		&Config{Blocks: []Block{{Daemons: []Daemon{{"c", Signal{syscall.SIGQUIT}}}}}},
	},
	{
		"",
		`[[block]]
include=["foo"]
[[block.prep]]
cmd = "command"`,
		&Config{
			Blocks: []Block{
				{
					Include: []string{"foo"},
					Preps:   []Prep{{Command: "command"}},
				},
			},
		},
	},
	{
		"",
		`[[block]]
include = ["foo"]
[[block.prep]]
onchange = true
cmd = "command"`,
		&Config{
			Blocks: []Block{
				{
					Include: []string{"foo"},
					Preps:   []Prep{{Command: "command", Onchange: true}},
				},
			},
		},
	},
	{
		"",
		`[[block]]
include = ["foo"]
[[block.prep]]
cmd = "command\n-one\n-two"`,
		&Config{
			Blocks: []Block{
				{
					Include: []string{"foo"},
					Preps:   []Prep{{Command: "command\n-one\n-two"}},
				},
			},
		},
	},
	{
		"",
		`[variables]
var="bar"
[[block]]
include = ["foo"]`,
		&Config{
			Blocks: []Block{
				{
					Include: []string{"foo"},
				},
			},
			Variables: map[string]string{
				"var": "bar",
			},
		},
	},
	{
		"",
		`[[block]]
indir = "foo"`,
		&Config{
			Blocks: []Block{
				{InDir: "foo"},
			},
		},
	},
	{
		"./path/to/ppow.toml",
		"",
		&Config{},
	},
	{
		"./path/to/ppow.toml",
		`[[block]]
indir = "@confdir/foo"`,
		&Config{
			Blocks: []Block{
				{InDir: "@confdir/foo"},
			},
		},
	},
}

var parseCmpOptions = []cmp.Option{
	cmp.AllowUnexported(Config{}),
}

func TestParse(t *testing.T) {
	for i, tt := range parseTests {
		t.Run(tt.input, func(t *testing.T) {
			ret, err := Parse(tt.path, tt.input)
			if err != nil {
				t.Errorf("%q - %s", tt.input, err)
			}

			if diff := cmp.Diff(ret, tt.expected, parseCmpOptions...); diff != "" {
				t.Errorf("%d %s", i, diff)
			}
		})
	}
}

func findAllConfigs(root string) []string {
	var configs []string
	filepath.WalkDir(root, func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if filepath.Ext(name) == ".toml" {
			configs = append(configs, name)
		}
		return nil
	})
	return configs
}

func TestParseExt(t *testing.T) {
	for _, tt := range findAllConfigs("..") {
		t.Run(tt, func(t *testing.T) {
			contents, err := os.ReadFile(tt)
			if err != nil {
				t.Errorf("failed to read config: %v", err)
				return
			}
			if _, err := Parse(tt, string(contents)); err != nil {
				t.Errorf("failed to parse config %s: %v", tt, err)
			}
		})
	}
}

var parseErrorTests = []struct {
	input string
	err   string
}{
	{`[[block]]
[[block.daemon]]
signal = "foobar"`, "unknown signal"},
}

func TestErrorsParse(t *testing.T) {
	for i, tt := range parseErrorTests {
		v, err := Parse("test", tt.input)
		if err == nil {
			t.Errorf("%d: Expected error, got %#v", i, v)
		}
		if err != nil && !strings.Contains(err.Error(), tt.err) {
			t.Errorf("Expected\n%q\ngot\n%q", tt.err, err.Error())
		}
	}
}
