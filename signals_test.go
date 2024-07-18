package ppow

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/dottedmag/must"
)

var testFatalSignals = map[string]syscall.Signal{
	"abrt":   syscall.SIGABRT,
	"fpe":    syscall.SIGFPE,
	"hup":    syscall.SIGHUP,
	"int":    syscall.SIGINT,
	"io":     syscall.SIGIO,
	"iot":    syscall.SIGIOT,
	"quit":   syscall.SIGQUIT,
	"sys":    syscall.SIGSYS,
	"term":   syscall.SIGTERM,
	"trap":   syscall.SIGTRAP,
	"vtalrm": syscall.SIGVTALRM,
	"xcpu":   syscall.SIGXCPU,
	"xfsz":   syscall.SIGXFSZ,
}

var testNonFatalSignals = map[string]syscall.Signal{
	"cont":  syscall.SIGCONT,
	"tstp":  syscall.SIGTSTP,
	"ttin":  syscall.SIGTTIN,
	"ttou":  syscall.SIGTTOU,
	"usr1":  syscall.SIGUSR1,
	"usr2":  syscall.SIGUSR2,
	"winch": syscall.SIGWINCH,
}

func TestSignals(t *testing.T) {
	d := t.TempDir()

	ppowBin := filepath.Join(d, "ppow")
	testbinBin := filepath.Join(d, "testbin")
	conf := filepath.Join(d, "ppow.conf")
	stdoutFile := filepath.Join(d, "stdout")

	cmdCompilePpow := exec.Command("go", "build", "-o", ppowBin, "./cmd/ppow")
	cmdCompilePpow.Stdout = os.Stdout
	cmdCompilePpow.Stderr = os.Stderr
	must.OK(cmdCompilePpow.Run())

	cmdCompileTestBin := exec.Command("go", "build", "-o", testbinBin, "./testbin")
	cmdCompileTestBin.Stdout = os.Stdout
	cmdCompileTestBin.Stderr = os.Stderr
	must.OK(cmdCompileTestBin.Run())

	for strSig, sig := range testFatalSignals {
		t.Run(strSig, func(t *testing.T) {
			must.OK(os.WriteFile(conf, []byte(`**.go {
    daemon: ./testbin
}`), 0o644))

			cmdRunPpow := exec.Command("./ppow")
			cmdRunPpow.Dir = d

			cmdRunPpow.Stdout = must.OK1(os.Create(stdoutFile))
			cmdRunPpow.Stderr = os.Stderr

			must.OK(cmdRunPpow.Start())
			defer cmdRunPpow.Process.Signal(syscall.SIGKILL)

			// Give ppow time to start itself and subprocess
			time.Sleep(100 * time.Millisecond)
			must.OK(cmdRunPpow.Process.Signal(sig))

			cmdRunPpow.Wait()

			// Give subprocesses time to exit
			time.Sleep(100 * time.Millisecond)

			stdout := string(must.OK1(os.ReadFile(stdoutFile)))

			if !strings.Contains(stdout, fmt.Sprintf("default handler %s", sig)) {
				fmt.Println(stdout)
				t.Fail()
			}
		})
	}

	for strSig, sig := range testFatalSignals {
		t.Run("ignore/"+strSig, func(t *testing.T) {
			must.OK(os.WriteFile(conf, []byte(`**.go {
    daemon: ./testbin `+strSig+`
}`), 0o644))

			cmdRunPpow := exec.Command("./ppow")
			cmdRunPpow.Dir = d

			cmdRunPpow.Stdout = must.OK1(os.Create(stdoutFile))
			cmdRunPpow.Stderr = os.Stderr

			must.OK(cmdRunPpow.Start())
			defer cmdRunPpow.Process.Signal(syscall.SIGKILL)

			// Give ppow time to start itself and subprocess
			time.Sleep(100 * time.Millisecond)
			must.OK(cmdRunPpow.Process.Signal(sig))

			// Force-kill subprocess that ignored fatal signal
			time.Sleep(100 * time.Millisecond)
			must.OK(cmdRunPpow.Process.Signal(syscall.SIGINT))

			cmdRunPpow.Wait()

			// Give subprocesses time to exit
			time.Sleep(100 * time.Millisecond)

			stdout := string(must.OK1(os.ReadFile(stdoutFile)))

			if !strings.Contains(stdout, fmt.Sprintf("ignoring %s", sig)) {
				fmt.Println(stdout)
				t.Fail()
			}

			if !strings.Contains(stdout, fmt.Sprintf("stopping via signal killed")) {
				fmt.Println(stdout)
				t.Fail()
			}
		})
	}

	for strSig, sig := range testNonFatalSignals {
		t.Run(strSig, func(t *testing.T) {
			must.OK(os.WriteFile(conf, []byte(`**.go {
    daemon: ./testbin
}`), 0o644))

			cmdRunPpow := exec.Command("./ppow")
			cmdRunPpow.Dir = d

			cmdRunPpow.Stdout = must.OK1(os.Create(stdoutFile))
			cmdRunPpow.Stderr = os.Stderr

			must.OK(cmdRunPpow.Start())
			defer cmdRunPpow.Process.Signal(syscall.SIGKILL)

			// Give ppow time to start itself and subprocess
			time.Sleep(100 * time.Millisecond)
			must.OK(cmdRunPpow.Process.Signal(sig))
			time.Sleep(100 * time.Millisecond)
			must.OK(cmdRunPpow.Process.Signal(sig))
			time.Sleep(100 * time.Millisecond)
			must.OK(cmdRunPpow.Process.Signal(syscall.SIGTERM))

			cmdRunPpow.Wait()

			// Give subprocesses time to exit
			time.Sleep(100 * time.Millisecond)

			stdout := string(must.OK1(os.ReadFile(stdoutFile)))

			if strings.Count(stdout, fmt.Sprintf("default handler %s", sig)) != 2 {
				fmt.Println(stdout)
				t.Fail()
			}
		})
		t.Run("ignore/"+strSig, func(t *testing.T) {
			must.OK(os.WriteFile(conf, []byte(`**.go {
    daemon: ./testbin `+strSig+`
}`), 0o644))

			cmdRunPpow := exec.Command("./ppow")
			cmdRunPpow.Dir = d

			cmdRunPpow.Stdout = must.OK1(os.Create(stdoutFile))
			cmdRunPpow.Stderr = os.Stderr

			must.OK(cmdRunPpow.Start())
			defer cmdRunPpow.Process.Signal(syscall.SIGKILL)

			// Give ppow time to start itself and subprocess
			time.Sleep(100 * time.Millisecond)
			must.OK(cmdRunPpow.Process.Signal(sig))
			time.Sleep(100 * time.Millisecond)
			must.OK(cmdRunPpow.Process.Signal(syscall.SIGTERM))

			cmdRunPpow.Wait()

			// Give subprocesses time to exit
			time.Sleep(100 * time.Millisecond)

			stdout := string(must.OK1(os.ReadFile(stdoutFile)))

			if !strings.Contains(stdout, fmt.Sprintf("ignoring %s", sig)) {
				fmt.Println(stdout)
				t.Fail()
			}
		})
	}

	t.Run("sigint->sigusr1", func(t *testing.T) {
		must.OK(os.WriteFile(conf, []byte(`**.go {
    daemon +sigusr1->sigusr2: ./testbin
}`), 0o644))

		cmdRunPpow := exec.Command("./ppow")
		cmdRunPpow.Dir = d

		cmdRunPpow.Stdout = must.OK1(os.Create(stdoutFile))
		cmdRunPpow.Stderr = os.Stderr

		must.OK(cmdRunPpow.Start())
		defer cmdRunPpow.Process.Signal(syscall.SIGKILL)

		// Give ppow time to start itself and subprocess
		time.Sleep(100 * time.Millisecond)
		must.OK(cmdRunPpow.Process.Signal(syscall.SIGUSR1))
		time.Sleep(100 * time.Millisecond)
		must.OK(cmdRunPpow.Process.Signal(syscall.SIGUSR1))
		time.Sleep(100 * time.Millisecond)
		must.OK(cmdRunPpow.Process.Signal(syscall.SIGTERM))

		cmdRunPpow.Wait()

		// Give subprocesses time to exit
		time.Sleep(100 * time.Millisecond)

		stdout := string(must.OK1(os.ReadFile(stdoutFile)))

		if strings.Count(stdout, fmt.Sprintf("default handler %s", syscall.SIGUSR2)) != 2 {
			fmt.Println(stdout)
			t.Fail()
		}
	})
}
