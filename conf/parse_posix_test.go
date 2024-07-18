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
		"{\ndaemon +sigusr1: c\n}",
		&Config{Blocks: []Block{{Daemons: []Daemon{{"c", syscall.SIGUSR1, nil}}}}},
	},
	{
		"",
		"{\ndaemon +sigusr2: c\n}",
		&Config{Blocks: []Block{{Daemons: []Daemon{{"c", syscall.SIGUSR2, nil}}}}},
	},
	{
		"",
		"{\ndaemon +sigwinch: c\n}",
		&Config{Blocks: []Block{{Daemons: []Daemon{{"c", syscall.SIGWINCH, nil}}}}},
	},
}

func init() {
	parseTests = append(parseTests, parsePosixTests...)
}
