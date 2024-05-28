package ppow

import (
	"fmt"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/cortesi/moddwatch"
	"github.com/cortesi/termlog"
	"github.com/dottedmag/ppow/conf"
	"github.com/dottedmag/ppow/notify"
	"github.com/dottedmag/ppow/shell"
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
	Notifiers  []notify.Notifier
}

// NewModRunner constructs a new ModRunner
func NewModRunner(confPath string, log termlog.TermLog, notifiers []notify.Notifier, confreload bool) (*ModRunner, error) {
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

func addCommonExcludes(c *conf.Config) {
	for i, b := range c.Blocks {
		if !b.NoCommonFilter {
			b.Exclude = append(b.Exclude, CommonExcludes...)
			c.Blocks[i] = b
		}
	}

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

	if _, err := shell.GetShellName(conf.Variables(newcnf)[shellVarName]); err != nil {
		return err
	}

	addCommonExcludes(newcnf)
	mr.Config = newcnf
	return nil
}

// PrepOnly runs all prep functions and exits
func (mr *ModRunner) PrepOnly(initial bool) error {
	for _, b := range mr.Config.Blocks {
		err := RunPreps(b, mr.Config, mr.ConfPath, nil, mr.Log, mr.Notifiers, initial)
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
		dir, err := globalEval(b.InDir, mr.ConfPath, mr.Config)
		if err != nil {
			mr.Log.Shout("Unable to evaluate indir directory: %s", err)
			return
		}
		dir, err = filepath.Abs(dir)
		if err != nil {
			mr.Log.Shout("Unable to absolutize indir directory: %s", err)
			return
		}
		err = os.Chdir(dir)
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
		mr.Config,
		mr.ConfPath,
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
blocks:
	for i, b := range mr.Config.Blocks {
		lmod := mod
		if lmod != nil {
			var err error
			includes, excludes, err := evalIncludesExcludes(&b, mr.ConfPath, mr.Config)
			if err != nil {
				mr.Log.Shout("%s", err)
				continue blocks
			}
			lmod, err = mod.Filter(root, includes, excludes)
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

func globalEval(s string, confPath string, cnf *conf.Config) (string, error) {
	return varEval(s, conf.Variables(cnf), map[string]string{
		"shell":    "sh",
		"confpath": filepath.Dir(confPath),
	})
}

func globalEvalList(ss []string, confPath string, cnf *conf.Config) ([]string, error) {
	var out []string
	for _, val := range ss {
		evaled, err := globalEval(val, confPath, cnf)
		if err != nil {
			return nil, err
		}
		out = append(out, evaled)
	}
	return out, nil
}

func evalIncludesExcludes(b *conf.Block, confPath string, cnf *conf.Config) (retIncludes, retExcludes []string, _ error) {
	includes, err := globalEvalList(b.Include, confPath, cnf)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to evaluate includes: %w", err)
	}
	excludes, err := globalEvalList(b.Exclude, confPath, cnf)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to evaluate excludes: %w", err)
	}
	return includes, excludes, nil
}

func getDirs(paths []string) []string {
	m := map[string]bool{}
	for _, p := range paths {
		p := path.Dir(p)
		m[p] = true
	}
	keys := []string{}
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// quotePath quotes a path for use on the command-line. The path must be in
// slash-delimited format, and the quoted path will use the native OS separator.
// FIXME: This is actually dependent on the shell used.
func quotePath(path string) string {
	path = strings.Replace(path, "\"", "\\\"", -1)
	return "\"" + path + "\""
}

// The paths we receive from Go's path manipulation functions are "cleaned",
// which removes redundancy, but also removes the leading "./" needed by many
// command-line tools. This function turns cleaned paths into "really relative"
// paths.
func realRel(p string) string {
	// They should already be clean, but let's make sure.
	p = path.Clean(p)
	if path.IsAbs(p) {
		return p
	} else if p == "." {
		return "./"
	}
	return "./" + p
}

// mkArgs prepares a list of paths for the command line
func mkArgs(paths []string) string {
	escaped := make([]string, len(paths))
	for i, s := range paths {
		escaped[i] = quotePath(realRel(s))
	}
	return strings.Join(escaped, " ")
}

func blockEval(s string, confPath string, cnf *conf.Config, modified []string) (string, error) {
	return varEval(s, conf.Variables(cnf), map[string]string{
		"shell":    "sh",
		"confpath": filepath.Dir(confPath),
		"mods":     mkArgs(modified),
		"dirmods":  mkArgs(getDirs(modified)),
	})
}

var varNameRx = regexp.MustCompile(`(\\*)@(\w+)`)

func varEval(s string, vars map[string]string, terminalVars map[string]string) (string, error) {
	var doVarEval func(string, map[string]bool) (string, error)

	doVarEval = func(s string, seenVars map[string]bool) (string, error) {
		mm := varNameRx.FindAllStringSubmatchIndex(s, -1)
		if len(mm) == 0 {
			return s, nil
		}

		out := s[:mm[0][0]]
		for i := 0; i < len(mm); i++ {
			if i > 0 {
				out += s[mm[i-1][1]:mm[i][0]]
			}
			nSlashes := mm[i][3] - mm[i][2]
			varName := s[mm[i][4]:mm[i][5]]

			out += strings.Repeat(`\`, nSlashes/2)
			if nSlashes%2 == 0 {
				// @ not escaped, do eval
				if seenVars[varName] {
					return "", fmt.Errorf("infinite recursion of variable @%s", varName)
				}
				if _, ok := vars[varName]; ok {
					seenVars[varName] = true
					expanded, err := doVarEval(vars[varName], seenVars)
					if err != nil {
						return "", err
					}
					delete(seenVars, varName)
					out += expanded
				} else if _, ok := terminalVars[varName]; ok {
					out += terminalVars[varName]
				} else {
					return "", fmt.Errorf("variable @%s is not defined", varName)
				}
			} else {
				// @ escaped
				out += "@" + varName
			}
		}
		out += s[mm[len(mm)-1][1]:]
		return out, nil
	}

	return doVarEval(s, map[string]bool{})
}

// Gives control of chan to caller
func (mr *ModRunner) runOnChan(modchan chan *moddwatch.Mod, readyCallback func()) error {
	dworld, err := NewDaemonWorld(mr.Config, mr.ConfPath, mr.Log)
	if err != nil {
		return err
	}
	defer dworld.Shutdown(os.Kill)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	defer signal.Reset(os.Interrupt, os.Kill)
	go func() {
		dworld.Shutdown(<-c)
		os.Exit(0)
	}()

	ipatts := conf.IncludePatterns(mr.Config)
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
