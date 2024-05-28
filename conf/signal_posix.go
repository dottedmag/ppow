//go:build !windows
// +build !windows

package conf

import (
	"os"
	"syscall"
)

var knownSignals = map[string]os.Signal{
	"sighup":   syscall.SIGHUP,
	"sigterm":  syscall.SIGTERM,
	"sigint":   syscall.SIGINT,
	"sigkill":  syscall.SIGKILL,
	"sigquit":  syscall.SIGQUIT,
	"sigusr1":  syscall.SIGUSR1,
	"sigusr2":  syscall.SIGUSR2,
	"sigwinch": syscall.SIGWINCH,
}
