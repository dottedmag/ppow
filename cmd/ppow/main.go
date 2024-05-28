package main

import (
	"fmt"
	"os"

	"github.com/dottedmag/ppow"
	"github.com/dottedmag/ppow/notify"
	"github.com/dottedmag/termlog"
	"github.com/spf13/pflag"
)

func main() {
	file := pflag.StringP("file", "f", "ppow.toml", "Path to modfile")
	noConf := pflag.BoolP("noconf", "c", false, "Don't watch our own config file")
	beep := pflag.BoolP("bell", "b", false, "Ring terminal bell if any command returns an error")
	ignores := pflag.BoolP("ignores", "i", false, "List default ignore patterns and exit")
	doNotify := pflag.BoolP("notifiy", "n", false, "Send stderr to system notification if commands error")
	prep := pflag.BoolP("prep", "p", false, "Run prep commands and exit")
	debug := pflag.Bool("debug", false, "Debugging for ppow development")
	version := pflag.Bool("version", false, "Show application version")

	pflag.Parse()

	if *version {
		fmt.Println(ppow.Version)
		os.Exit(0)
	}

	if *ignores {
		for _, patt := range ppow.CommonExcludes {
			fmt.Println(patt)
		}
		os.Exit(0)
	}

	log := termlog.NewLog()
	if *debug {
		log.Enable("debug")
	}

	notifiers := []notify.Notifier{}
	if *doNotify {
		n := notify.PlatformNotifier()
		if n == nil {
			log.Shout("Could not find a desktop notifier")
		} else {
			notifiers = append(notifiers, n)
		}
	}
	if *beep {
		notifiers = append(notifiers, &notify.BeepNotifier{})
	}

	mr, err := ppow.NewModRunner(*file, log, notifiers, !(*noConf))
	if err != nil {
		log.Shout("%s", err)
		return
	}

	if *prep {
		err := mr.PrepOnly(true)
		if err != nil {
			log.Shout("%s", err)
		}
	} else {
		err = mr.Run()
		if err != nil {
			log.Shout("%s", err)
		}
	}
}
