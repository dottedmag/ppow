package conf

// The base parser is from the text/template package in Go.

// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

import (
	"fmt"
	"sort"

	"github.com/BurntSushi/toml"
	"golang.org/x/exp/maps"
)

const confVarName = "@confdir"

// = path.Dir(p.name)

//func expandVariables(c *Config) error {
//}

// Parse parses a string, and returns a completed Config
func Parse(name string, text string) (*Config, error) {
	var c Config

	meta, err := toml.Decode(text, &c)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config %q: %w", name, err)
	}
	undecodedKeys := map[string]bool{}
	for _, k := range meta.Undecoded() {
		undecodedKeys[k.String()] = true
	}
	if len(undecodedKeys) > 0 {
		ks := maps.Keys(undecodedKeys)
		sort.Strings(ks)
		return nil, fmt.Errorf("unexpected keys in config %q: %v", name, ks)
	}
	for name := range c.Variables {
		switch name {
		case "confdir", "mods", "dirmods":
			return nil, fmt.Errorf("%q is a built-in variable, may not be overriden", name)
		}
	}
	// ...expand variables?... add @ to keys
	// ...SIGHUP by default...
	// absolutize paths
	return &c, nil
}
