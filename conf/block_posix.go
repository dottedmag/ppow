//go:build !windows

package conf

import (
	"fmt"
	"os"
	"strings"
	"syscall"
)

var strSignals = map[string]os.Signal{
	"sighup":   syscall.SIGHUP,
	"sigterm":  syscall.SIGTERM,
	"sigint":   syscall.SIGINT,
	"sigkill":  syscall.SIGKILL,
	"sigquit":  syscall.SIGQUIT,
	"sigusr1":  syscall.SIGUSR1,
	"sigusr2":  syscall.SIGUSR2,
	"sigwinch": syscall.SIGWINCH,
}

func (b *Block) addDaemon(command string, options []string) error {
	if b.Daemons == nil {
		b.Daemons = []Daemon{}
	}
	d := Daemon{
		Command:       command,
		RestartSignal: syscall.SIGHUP,
	}
	for _, v := range options {
		v = strings.TrimPrefix(v, "+")
		if strings.Contains(v, "->") {
			strFrom, strTo, ok := strings.Cut(v, "->")
			if !ok {
				return fmt.Errorf("unknown signal mapping: %s", v)
			}
			from := strSignals[strFrom]
			if from == nil {
				return fmt.Errorf("unknown signal: %s", strFrom)
			}
			to := strSignals[strTo]
			if to == nil {
				return fmt.Errorf("unknown signal: %s", strTo)
			}
			if d.SignalMapping == nil {
				d.SignalMapping = map[os.Signal]os.Signal{}
			}
			d.SignalMapping[from] = to
		} else {
			sig := strSignals[v]
			if sig == nil {
				return fmt.Errorf("unknown signal: %s", v)
			}
			d.RestartSignal = sig
		}
	}
	b.Daemons = append(b.Daemons, d)
	return nil
}
