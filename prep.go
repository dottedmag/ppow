package ppow

import (
	"path/filepath"
	"time"

	"github.com/cortesi/moddwatch"
	"github.com/cortesi/termlog"
	"github.com/dottedmag/ppow/conf"
	"github.com/dottedmag/ppow/notify"
	"github.com/dottedmag/ppow/shell"
)

// ProcError is a process error, possibly containing command output
type ProcError struct {
	shorttext string
	Output    string
}

func (p ProcError) Error() string {
	return p.shorttext
}

// RunProc runs a process to completion, sending output to log
func RunProc(cmd string, shellMethod string, dir string, log termlog.Stream) error {
	log.Header()
	ex, err := shell.NewExecutor(shellMethod, cmd, dir)
	if err != nil {
		return err
	}
	start := time.Now()
	err, estate := ex.Run(log, true)
	if err != nil {
		return err
	} else if estate.Error != nil {
		log.Shout("%s", estate.Error)
		return ProcError{estate.Error.Error(), estate.ErrOutput}
	}
	log.Notice(">> done (%s)", time.Since(start))
	return nil
}

// RunPreps runs all commands in sequence. Stops if any command returns an error.
func RunPreps(
	b conf.Block,
	cnf *conf.Config,
	confPath string,
	mod *moddwatch.Mod,
	log termlog.TermLog,
	notifiers []notify.Notifier,
	initial bool,
) error {
	sh, err := globalEval("@shell", confPath, cnf)
	if err != nil {
		return err
	}

	var modified []string
	if mod != nil {
		modified = mod.All()
	}

	if modified == nil {
		includes, excludes, err := evalIncludesExcludes(&b, confPath, cnf)
		if err != nil {
			return err
		}
		modified, err = moddwatch.List(".", includes, excludes)
		if err != nil {
			return err
		}
	}

	for _, p := range b.Preps {
		cmd, err := blockEval(p.Command, confPath, cnf, modified)
		if initial && p.Onchange {
			log.Say(niceHeader("skipping prep: ", cmd))
			continue
		}
		if err != nil {
			return err
		}
		dir, err := globalEval(b.InDir, confPath, cnf)
		if err != nil {
			return err
		}
		dir, err = filepath.Abs(dir)
		if err != nil {
			return err
		}
		err = RunProc(cmd, sh, dir, log.Stream(niceHeader("prep: ", cmd)))
		if err != nil {
			if pe, ok := err.(ProcError); ok {
				for _, n := range notifiers {
					n.Push("ppow error", pe.Output, "")
				}
			}
			return err
		}
	}
	return nil
}
