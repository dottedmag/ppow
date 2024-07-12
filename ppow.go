package ppow

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/cortesi/moddwatch"
	"github.com/dottedmag/ppow/conf"
	"github.com/dottedmag/termlog"
)

// Version is the ppow release version
const Version = "0.9-pre"

const lullTime = time.Millisecond * 100

const shellVarName = "@shell"

// CommonExcludes is a list of commonly excluded files suitable for passing in
// the excludes parameter to Watch - includes repo directories, temporary
// files, and so forth.
var CommonExcludes = []string{
	// VCS
	"**/.git/**",
	"**/.hg/**",
	"**/.svn/**",
	"**/.bzr/**",

	// OSX
	"**/.DS_Store/**",

	// Temporary files
	"**.tmp",
	"**~",
	"**#",
	"**.bak",
	"**.swp",
	"**.___jb_old___",
	"**.___jb_bak___",
	"**mage_output_file.go",

	// Python
	"**.py[cod]",

	// Node
	"**/node_modules/**",
}

// ModRunner coordinates running the ppow command
type ModRunner struct {
	Log        termlog.TermLog
	Config     *conf.Config
	ConfPath   string
	ConfReload bool
	Notifiers  []Notifier
	signalled  bool
}

// NewModRunner constructs a new ModRunner
func NewModRunner(confPath string, log termlog.TermLog, notifiers []Notifier, confreload bool) (*ModRunner, error) {
	mr := &ModRunner{
		Log:        log,
		ConfPath:   confPath,
		ConfReload: confreload,
		Notifiers:  notifiers,
	}
	err := mr.ReadConfig()
	if err != nil {
		return nil, err
	}
	return mr, nil
}

// ReadConfig parses the configuration file in ConfPath
func (mr *ModRunner) ReadConfig() error {
	ret, err := os.ReadFile(mr.ConfPath)
	if err != nil {
		return fmt.Errorf("Error reading config file %s: %s", mr.ConfPath, err)
	}
	newcnf, err := conf.Parse(mr.ConfPath, string(ret))
	if err != nil {
		return fmt.Errorf("Error reading config file %s: %s", mr.ConfPath, err)
	}

	if _, err := GetShellName(newcnf.GetVariables()[shellVarName]); err != nil {
		return err
	}

	newcnf.CommonExcludes(CommonExcludes)
	mr.Config = newcnf
	return nil
}

// PrepOnly runs all prep functions and exits
func (mr *ModRunner) PrepOnly(initial bool) error {
	for _, b := range mr.Config.Blocks {
		err := RunPreps(b, mr.Config.GetVariables(), nil, mr.Log, mr.Notifiers, initial)
		if err != nil {
			return err
		}
	}
	return nil
}

func (mr *ModRunner) runBlock(b conf.Block, mod *moddwatch.Mod, dpen *DaemonPen) {
	if b.InDir != "" {
		currentDir, err := os.Getwd()
		if err != nil {
			mr.Log.Shout("Error getting current working directory: %s", err)
			return
		}
		err = os.Chdir(b.InDir)
		if err != nil {
			mr.Log.Shout(
				"Error changing to indir directory \"%s\": %s",
				b.InDir,
				err,
			)
			return
		}
		defer func() {
			err := os.Chdir(currentDir)
			if err != nil {
				mr.Log.Shout("Error returning to original directory: %s", err)
			}
		}()
	}
	err := RunPreps(
		b,
		mr.Config.GetVariables(),
		mod, mr.Log,
		mr.Notifiers,
		mod == nil,
	)
	if err != nil {
		if _, ok := err.(ProcError); !ok {
			mr.Log.Shout("Error running prep: %s", err)
		}
		return
	}
	dpen.Restart()
}

func (mr *ModRunner) trigger(root string, mod *moddwatch.Mod, dworld *DaemonWorld) {
	for i, b := range mr.Config.Blocks {
		lmod := mod
		if lmod != nil {
			var err error
			lmod, err = mod.Filter(root, b.Include, b.Exclude)
			if err != nil {
				mr.Log.Shout("Error filtering events: %s", err)
				continue
			}
			if lmod.Empty() {
				continue
			}
		}
		mr.runBlock(b, lmod, dworld.DaemonPens[i])
	}
}

//
// - prep
// - daemon
//
// Signals are passed, processes are not restarted.
// SIGTERM is special: if invoked twice then second time it's a KILL.
//

var sentinel = &moddwatch.Mod{}

var fatalSignals = []os.Signal{
	syscall.SIGABRT,
	syscall.SIGFPE,
	syscall.SIGHUP,
	syscall.SIGINT,
	syscall.SIGIO,
	syscall.SIGIOT,
	syscall.SIGQUIT,
	syscall.SIGSYS,
	syscall.SIGTERM,
	syscall.SIGTRAP,
	syscall.SIGVTALRM,
	syscall.SIGXCPU,
	syscall.SIGXFSZ,
}

var nonFatalSignals = []os.Signal{
	syscall.SIGCONT,
	syscall.SIGTSTP,
	syscall.SIGTTIN,
	syscall.SIGTTOU,
	syscall.SIGUSR1,
	syscall.SIGUSR2,
	syscall.SIGWINCH,
}

func nonFatalSignal(sig os.Signal) bool {
	for _, s := range nonFatalSignals {
		if s == sig {
			return true
		}
	}
	return false
}

// Gives control of chan to caller
func (mr *ModRunner) runOnChan(modchan chan *moddwatch.Mod, readyCallback func()) error {
	dworld, err := NewDaemonWorld(mr.Config, mr.Log)
	if err != nil {
		return err
	}
	defer dworld.Shutdown(os.Kill)

	c := make(chan os.Signal, 1)

	for _, sig := range fatalSignals {
		signal.Notify(c, sig)
	}
	for _, sig := range nonFatalSignals {
		signal.Notify(c, sig)
	}
	defer signal.Reset()

	go func() {
		for {
			sig := <-c

			if nonFatalSignal(sig) {
				mr.Log.Notice("Received signal %s, passing to running processes (if any)...", sig)
				dworld.Signal(sig)
				continue
			}

			if sig == syscall.SIGINT && mr.signalled {
				mr.Log.Notice("Received SIGINT after another signal, force-killing remaining processes")
				modchan <- sentinel
				return
			}

			mr.Log.Notice("Received signal %s, passing to running processes (if any)...", sig)
			mr.Log.Notice("(Hint: if any processes are stuck, send SIGINT for force-killing them)")
			mr.signalled = true
			dworld.Shutdown(sig)
			// Give the subprocesses time to exit
			// TODO (misha): Catch the exit code and propagate
			time.Sleep(100 * time.Millisecond)
			modchan <- sentinel
			return
		}
	}()

	ipatts := mr.Config.IncludePatterns()
	if mr.ConfReload {
		ipatts = append(ipatts, filepath.Dir(mr.ConfPath))
	}

	currentDir, err := os.Getwd()
	if err != nil {
		return err
	}
	// FIXME: This takes a long time. We could start it in parallel with the
	// first process run in a goroutine
	watcher, err := moddwatch.Watch(currentDir, ipatts, []string{}, lullTime, modchan)

	if err != nil {
		return fmt.Errorf("Error watching: %s", err)
	}
	defer watcher.Stop()

	mr.trigger(currentDir, nil, dworld)
	go readyCallback()
	for mod := range modchan {
		if mod == nil {
			break
		}
		if mod == sentinel {
			return fmt.Errorf("shutdown")
		}
		if mr.ConfReload && mod.Has(mr.ConfPath) {
			mr.Log.Notice("Reloading config %s", mr.ConfPath)
			err := mr.ReadConfig()
			if err != nil {
				mr.Log.Warn("%s", err)
				continue
			} else {
				return nil
			}
		}
		mr.Log.SayAs("debug", "Delta: \n%s", mod.String())
		mr.trigger(currentDir, mod, dworld)
	}
	return nil
}

// Run is the top-level runner for ppow
func (mr *ModRunner) Run() error {
	for {
		modchan := make(chan *moddwatch.Mod, 1024)
		err := mr.runOnChan(modchan, func() {})
		if err != nil {
			return err
		}
	}
}
