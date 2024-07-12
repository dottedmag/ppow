package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// This binary prints all the signals it receives and can be configured
// to react to signals in various ways

var sigCh = make(chan os.Signal, 1)

func invokeDefault(sig os.Signal) {
	signal.Reset(sig)
	defer func() {
		signal.Notify(sigCh, sig)
	}()

	syscall.Kill(syscall.Getpid(), sig.(syscall.Signal))

	// Give it time to be handled before re-instating signal handler
	// Signals are asynchronous, so without the delay the signal is
	// delivered but not handled when defer() reinstates signal handler
	time.Sleep(1 * time.Millisecond)
}

var ignoredSignals = map[os.Signal]bool{}

var allSignals = map[string]os.Signal{
	"abrt":   syscall.SIGABRT,
	"fpe":    syscall.SIGFPE,
	"hup":    syscall.SIGHUP,
	"int":    syscall.SIGINT,
	"io":     syscall.SIGIO,
	"iot":    syscall.SIGIOT,
	"quit":   syscall.SIGQUIT,
	"segv":   syscall.SIGSEGV,
	"sys":    syscall.SIGSYS,
	"term":   syscall.SIGTERM,
	"trap":   syscall.SIGTRAP,
	"tstp":   syscall.SIGTSTP,
	"ttin":   syscall.SIGTTIN,
	"ttou":   syscall.SIGTTOU,
	"usr1":   syscall.SIGUSR1,
	"usr2":   syscall.SIGUSR2,
	"vtalrm": syscall.SIGVTALRM,
	"winch":  syscall.SIGWINCH,
	"xcpu":   syscall.SIGXCPU,
	"xfsz":   syscall.SIGXFSZ,
}

func main() {
	for _, strSig := range os.Args[1:] {
		ignoredSignals[allSignals[strSig]] = true
	}

	for _, sig := range allSignals {
		signal.Notify(sigCh, sig)
	}

	fmt.Printf("ready %d\n", syscall.Getpid())

	for sig := range sigCh {
		if ignoredSignals[sig] {
			fmt.Printf("ignoring %s\n", sig)
			continue
		}
		fmt.Printf("default handler %s\n", sig)
		invokeDefault(sig)
	}
}
