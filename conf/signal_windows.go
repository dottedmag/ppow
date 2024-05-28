//go:build windows
// +build windows

package conf

import (
	"os"
	"syscall"
)

var knownSignals = map[string]os.Signal{
	"sighup":  syscall.SIGHUP,
	"sigterm": syscall.SIGTERM,
	"sigint":  syscall.SIGINT,
	"sigkill": syscall.SIGKILL,
	"sigquit": syscall.SIGQUIT,
}
