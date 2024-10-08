package main

import (
	"fmt"
	"os"

	"github.com/dottedmag/ppow"
	"github.com/dottedmag/termlog"
	"github.com/spf13/pflag"
)

func main() {
	file := pflag.StringP("file", "f", "", "Path to modfile (defaults to ppow.conf with fallback to mmod.conf if not specified)")
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

	notifiers := []ppow.Notifier{}
	if *doNotify {
		n := ppow.PlatformNotifier()
		if n == nil {
			log.Shout("Could not find a desktop notifier")
		} else {
			notifiers = append(notifiers, n)
		}
	}
	if *beep {
		notifiers = append(notifiers, &ppow.BeepNotifier{})
	}

	if *file == "" && fileExists("ppow.conf") {
		*file = "ppow.conf"
	}
	if *file == "" && fileExists("modd.conf") {
		*file = "modd.conf"
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

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
