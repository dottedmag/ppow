package conf

import (
	"fmt"
	"maps"
	"os"
	"sort"

	expmaps "golang.org/x/exp/maps"
)

type Signal struct {
	os.Signal
}

func (s *Signal) UnmarshalText(text []byte) error {
	if sig, ok := knownSignals[string(text)]; ok {
		s.Signal = sig
		return nil
	}
	return fmt.Errorf("unknown signal: %s", text)
}

// A Daemon is a persistent process that is kept running
type Daemon struct {
	Command       string `toml:"cmd"` // expand, @confdir, @mods, @dirmods
	RestartSignal Signal `toml:"signal"`
}

// A Prep runs and terminates
type Prep struct {
	Command  string `toml:"cmd"`      // expand, @confdir, @mods, @dirmods
	Onchange bool   `toml:"onchange"` // Should prep skip initial run
}

// Block is a match pattern and a set of specifications
type Block struct {
	Include        []string `toml:"include"`
	Exclude        []string `toml:"exclude"`
	NoCommonFilter bool     `toml:"noignore"`
	InDir          string   `toml:"indir"`

	Daemons []Daemon `toml:"daemon"`
	Preps   []Prep   `toml:"prep"`
}

// Config represents a complete configuration
type Config struct {
	Blocks    []Block           `toml:"block"`
	Variables map[string]string `toml:"variables"`
}

// Variables returns a copy of the variables map in config
func Variables(c *Config) map[string]string {
	if c.Variables == nil {
		return map[string]string{}
	}
	return maps.Clone(c.Variables)
}

func IncludePatterns(c *Config) []string {
	paths := map[string]bool{}

	for _, b := range c.Blocks {
		for _, p := range b.Include {
			paths[p] = true
		}
	}

	out := expmaps.Keys(paths)
	sort.Strings(out)
	return out
}
