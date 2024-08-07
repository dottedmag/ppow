package conf

import (
	"io/fs"
	"os"
	"path/filepath"
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
		"{}",
		&Config{
			Blocks: []Block{
				{},
			},
		},
	},
	{
		"",
		"foo {}",
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
		"foo bar {}",
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
		"!foo {}",
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
		`!"foo" {}`,
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
		`!"foo" !'bar' !voing {}`,
		&Config{
			Blocks: []Block{
				{Exclude: []string{"foo", "bar", "voing"}},
			},
		},
	},
	{
		"",
		`foo +noignore {}`,
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
		"'foo bar' voing {}",
		&Config{
			Blocks: []Block{
				{
					Include: []string{"foo bar", "voing"},
				},
			},
		},
	},
	{
		"",
		"foo {\ndaemon: command\n}",
		&Config{
			Blocks: []Block{
				{
					Include: []string{"foo"},
					Daemons: []Daemon{{"command", syscall.SIGHUP, nil}},
				},
			},
		},
	},
	{
		"",
		"{\ndaemon +sighup: c\n}",
		&Config{
			Blocks: []Block{
				{Daemons: []Daemon{{"c", syscall.SIGHUP, nil}}},
			},
		},
	},
	{
		"",
		"{\ndaemon +sigterm: c\n}",
		&Config{Blocks: []Block{{Daemons: []Daemon{{"c", syscall.SIGTERM, nil}}}}},
	},
	{
		"",
		"{\ndaemon +sigint: c\n}",
		&Config{Blocks: []Block{{Daemons: []Daemon{{"c", syscall.SIGINT, nil}}}}},
	},
	{
		"",
		"{\ndaemon +sigkill: c\n}",
		&Config{Blocks: []Block{{Daemons: []Daemon{{"c", syscall.SIGKILL, nil}}}}},
	},
	{
		"",
		"{\ndaemon +sigquit: c\n}",
		&Config{Blocks: []Block{{Daemons: []Daemon{{"c", syscall.SIGQUIT, nil}}}}},
	},
	{
		"",
		"{\ndaemon +sigquit->sigterm +sigterm->sigusr1: c\n}",
		&Config{Blocks: []Block{{Daemons: []Daemon{{"c", syscall.SIGHUP, map[os.Signal]os.Signal{syscall.SIGQUIT: syscall.SIGTERM, syscall.SIGTERM: syscall.SIGUSR1}}}}}},
	},
	{
		"",
		"foo {\nprep: command\n}",
		&Config{
			Blocks: []Block{
				{
					Include: []string{"foo"},
					Preps:   []Prep{Prep{Command: "command"}},
				},
			},
		},
	},
	{
		"",
		"foo {\nprep +onchange: command\n}",
		&Config{
			Blocks: []Block{
				{
					Include: []string{"foo"},
					Preps:   []Prep{Prep{Command: "command", Onchange: true}},
				},
			},
		},
	},
	{
		"",
		"foo {\nprep: 'command\n-one\n-two'}",
		&Config{
			Blocks: []Block{
				{
					Include: []string{"foo"},
					Preps:   []Prep{Prep{Command: "command\n-one\n-two"}},
				},
			},
		},
	},
	{
		"",
		"foo #comment\nbar\n#comment\n{\n#comment\nprep: command\n}",
		&Config{
			Blocks: []Block{
				{
					Include: []string{"foo", "bar"},
					Preps:   []Prep{Prep{Command: "command"}},
				},
			},
		},
	},
	{
		"",
		"foo #comment\n#comment\nbar { #comment \nprep: command\n}",
		&Config{
			Blocks: []Block{
				{
					Include: []string{"foo", "bar"},
					Preps:   []Prep{{"command", false}},
				},
			},
		},
	},
	{
		"",
		"@var=bar\nfoo {}",
		&Config{
			Blocks: []Block{
				{
					Include: []string{"foo"},
				},
			},
			variables: map[string]string{
				"@var": "bar",
			},
		},
	},
	{
		"",
		"@var='bar\nvoing'\nfoo {}",
		&Config{
			Blocks: []Block{
				{
					Include: []string{"foo"},
				},
			},
			variables: map[string]string{
				"@var": "bar\nvoing",
			},
		},
	},
	{
		"",
		"foo {}\n@var=bar\n",
		&Config{
			Blocks: []Block{
				{
					Include: []string{"foo"},
				},
			},
			variables: map[string]string{
				"@var": "bar",
			},
		},
	},
	{
		"",
		"@oink=foo\nfoo {}\n@var=bar\n",
		&Config{
			Blocks: []Block{
				{
					Include: []string{"foo"},
				},
			},
			variables: map[string]string{
				"@var":  "bar",
				"@oink": "foo",
			},
		},
	},
	{
		"",
		"{ indir: foo\n }",
		&Config{
			Blocks: []Block{
				{InDir: mustAbs("foo")},
			},
		},
	},
	{
		"./path/to/ppow.conf",
		"",
		&Config{
			variables: map[string]string{
				"@confdir": "path/to",
			},
		},
	},
	{
		"./path/to/ppow.conf",
		"{ indir: @confdir/foo\n }",
		&Config{
			Blocks: []Block{
				{InDir: mustAbs("path/to/foo")},
			},
			variables: map[string]string{
				"@confdir": "path/to",
			},
		},
	},
}

var parseCmpOptions = []cmp.Option{
	cmp.AllowUnexported(Config{}),
}

func TestParse(t *testing.T) {
	for i, tt := range parseTests {
		ret, err := Parse(tt.path, tt.input)
		if err != nil {
			t.Fatalf("%q - %s", tt.input, err)
		}

		if diff := cmp.Diff(ret, tt.expected, parseCmpOptions...); diff != "" {
			t.Errorf("%d %s", i, diff)
		}
	}
}

func findAllConfigs(root string) []string {
	var configs []string
	filepath.WalkDir(root, func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if filepath.Ext(name) == ".conf" {
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
	{"{", "test:1: unterminated block"},
	{"a", "test:1: expected block open parentheses, got \"\""},
	{`foo { "bar": "bar" }`, "test:1: invalid input"},
	{"foo { daemon: \n }", "test:1: empty command specification"},
	{"foo { daemon: \" }", "test:1: unterminated quoted string"},
	{"foo { daemon *: foo }", "test:1: invalid syntax"},
	{"foo { daemon +invalid: foo }", "test:1: unknown signal: invalid"},
	{"foo { prep +invalid: foo }", "test:1: unknown signal: +invalid"},
	{"foo { prep +sigterm->sigbaa: foo }", "test:1: unknown signal: +sigterm->sigbaa"},
	{"foo { prep +sigboo->sigusr1: foo }", "test:1: unknown signal: +sigboo->sigusr1"},
	{"@foo bar {}", "test:1: Expected ="},
	{"@foo =", "test:1: unterminated variable assignment"},
	{"@foo=bar\n@foo=bar {}", "test:2: variable @foo shadows previous declaration"},
	{"{indir +foo: bar\n}", "test:1: indir takes no options"},
	{"{indir: bar\nindir: voing\n}", "test:2: indir can only be used once per block"},
}

func TestErrorsParse(t *testing.T) {
	for i, tt := range parseErrorTests {
		v, err := Parse("test", tt.input)
		if err == nil {
			t.Fatalf("%d: Expected error, got %#v", i, v)
		}
		if err.Error() != tt.err {
			t.Errorf("Expected\n%q\ngot\n%q", tt.err, err.Error())
		}
	}
}
