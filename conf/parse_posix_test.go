//go:build !windows
// +build !windows

package conf

import (
	"syscall"
)

var parsePosixTests = []struct {
	path     string
	input    string
	expected *Config
}{
	{
		"",
		`[[block]]
[[block.daemon]]
signal = "sigusr1"
cmd = "c"`,
		&Config{Blocks: []Block{{Daemons: []Daemon{{"c", Signal{syscall.SIGUSR1}}}}}},
	},
	{
		"",
		`[[block]]
[[block.daemon]]
signal = "sigusr2"
cmd = "c"`,
		&Config{Blocks: []Block{{Daemons: []Daemon{{"c", Signal{syscall.SIGUSR2}}}}}},
	},
	{
		"",
		`[[block]]
[[block.daemon]]
signal = "sigwinch"
cmd = "c"`,
		&Config{Blocks: []Block{{Daemons: []Daemon{{"c", Signal{syscall.SIGWINCH}}}}}},
	},
}

func init() {
	parseTests = append(parseTests, parsePosixTests...)
}
