package ppow

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/dottedmag/termlog"
)

type cmdTest struct {
	name    string
	cmd     string
	bufferr bool

	shells []string

	logHas  string
	buffHas string
	err     bool
	procerr bool
	kill    bool
}

func testCmd(t *testing.T, shell string, ct cmdTest) {
	if ct.shells != nil {
		issh := func() bool {
			for _, v := range ct.shells {
				if v == shell {
					return true
				}
			}
			return false
		}()
		if !issh {
			t.Skip("skipping")
			return
		}
	}

	lt := termlog.NewLogTest()
	exec, err := NewExecutor(shell, ct.cmd, "")
	if err != nil {
		t.Error(err)
		return
	}
	type result struct {
		err    error
		pstate *ExecState
	}

	ch := make(chan result)
	go func() {
		err, pstate := exec.Run(lt.Log.Stream(""), ct.bufferr)
		ch <- result{err: err, pstate: pstate}
	}()

	// Wait for the first output to make sure process is running
	for {
		time.Sleep(100 * time.Millisecond)
		if lt.String() != "" {
			break
		}
	}

	if ct.kill {
		err := exec.Signal(syscall.SIGKILL)
		if err != nil {
			t.Errorf("Error stopping: %s", err)
			return
		}
		time.Sleep(1 * time.Second)
	}

	res := <-ch
	if (res.err != nil) != ct.err {
		t.Errorf("Unexpected invocation error: %s", err)
	}
	if (res.pstate.Error != nil) != ct.procerr {
		t.Errorf("Unexpected process error: %s, %s", res.pstate.Error, res.pstate.ErrOutput)
	}
	if ct.buffHas != "" && !strings.Contains(res.pstate.ErrOutput, ct.buffHas) {
		t.Errorf("Unexpected buffer return: %s", res.pstate.ErrOutput)
	}
	if ct.logHas != "" && !strings.Contains(lt.String(), ct.logHas) {
		t.Errorf("Unexpected log return: %s", lt.String())
	}
}

var shellTests = []cmdTest{
	{
		name:   "echosuccess",
		cmd:    "echo ppowtest; true",
		logHas: "ppowtest",
	},
	{
		name:    "echofail",
		cmd:     "echo ppowtest; false",
		logHas:  "ppowtest",
		procerr: true,
	},
	{
		name:    "unknowncmd",
		cmd:     "definitelynosuchcommand",
		procerr: true,
	},
	{
		name:    "stderr-posix",
		cmd:     "echo ppowstderr >&2",
		bufferr: true,
		buffHas: "ppowstderr",
		shells:  []string{"ppow", "sh", "bash"},
	},
	{
		name:    "stderr-powershell",
		cmd:     "Write-Error \"ppowstderr\"",
		bufferr: true,
		procerr: true,
		buffHas: "ppowstderr",
		shells:  []string{"powershell"},
	},
	{
		name:    "kill",
		cmd:     "echo ppowtest; echo; sleep 999999",
		logHas:  "ppowtest",
		kill:    true,
		procerr: true,
	},
}

func TestShells(t *testing.T) {
	shellTesting = true

	var shells []string
	if runtime.GOOS == "windows" {
		shells = []string{
			"powershell",
		}
	} else {
		shells = []string{
			"sh",
			"bash",
			"powershell",
		}
	}
	for _, sh := range shells {
		for _, tc := range shellTests {
			t.Run(
				fmt.Sprintf("%s/%s", sh, tc.name),
				func(t *testing.T) {
					if _, err := CheckShell(sh); err != nil {
						t.Skipf("skipping - %s", err)
						return
					}
					testCmd(t, sh, tc)
				},
			)
		}
	}
}

func TestCaseInsensitivePath(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("skipping - only windows has case insensitive PATH")
	}

	oldpath := os.Getenv("PATH")
	fixpath := func() {
		os.Unsetenv("Path")
		os.Setenv("PATH", oldpath)
	}
	defer fixpath()
	os.Unsetenv("PATH")
	os.Setenv("Path", fmt.Sprintf("%s%ctrigger-text", oldpath, os.PathListSeparator))

	shellTesting = true

	pathTest := cmdTest{
		name:   "path-test",
		cmd:    "echo $PATH",
		logHas: "trigger-text",
	}
	sh := "ppow"
	t.Run(
		"ppow/path-capitalization",
		func(t *testing.T) {
			if _, err := CheckShell(sh); err != nil {
				t.Skipf("skipping - %s", err)
			}
			testCmd(t, sh, pathTest)
		},
	)
}
