ppow is a developer tool that triggers commands and manages daemons in response
to filesystem changes. It is a fork of excellent but unmaintained [modd](https://github.com/cortesi/modd).

If you use ppow, you should also look at
[devd](https://github.com/cortesi/devd), a compact HTTP daemon for developers.
Devd integrates with ppow, allowing you to trigger in-browser livereload with
ppow.

The repo contains a set of example *ppow.toml* files that you can look at for a
quick idea of what ppow can do:

Example                                      | Description
-------------------------------------------- | -------
[frontend.ppow.toml](./examples/frontend.ppow.toml)    | A front-end project with React + Browserify + Babel. ppow and devd replace many functions of Gulp/Grunt.
[go.ppow.toml](./examples/go.ppow.toml)                | Live unit tests for Go.
[python.ppow.toml](./examples/python.ppow.toml)        | Python + Redis, with devd managing livereload.



# Install

ppow is a single binary with no external dependencies, released for OSX,
Windows, Linux, FreeBSD, NetBSD and OpenBSD. Go to the [releases
page](https://github.com/dottedmag/ppow/releases/latest), download the package for
your OS, and copy the binary to somewhere on your PATH.

Alternatively, with Go 1.17+ installed, you can install `ppow` directly using `go install`. Please note that CGO is required, so if you happen to have it disabled you will need to prepend the `CGO_ENABLED=1` environment variable.

    $ go install github.com/dottedmag/ppow/cmd/ppow@latest

# Quick start

Put this in a file called *ppow.toml*:

```
[[block]]
include = ["**/*.go"]
[[block.prep]]
cmd = "go test @dirmods"
```

Now run ppow like so:

```
$ ppow
```

The first time ppow is run, it will run the tests of all Go modules. Whenever
any file with the .go extension is modified, the "go test" command will be run
only on the enclosing module.


# Details

On startup, ppow looks for a file called *ppow.toml* in the current directory.
This file is TOML, consisting of one or more blocks of commands,
each of which can be triggered on changes to files matching a set of file
patterns. The *ppow.toml* file is meant to be portable, and can safely be
checked into source repositories. Functionality that users will want to
customize (like desktop notifications) is controlled through command-line
flags.

Commands have two flavors: **prep** commands that run and terminate (e.g.
compiling, running test suites or running linters), and **daemon** commands that
run and keep running (e.g databases or webservers). Daemons are sent a SIGHUP
(by default) when their block is triggered, and are restarted if they ever exit.

Prep commands are run in order of occurrence. If any prep command exits with an
error, execution of the current block is stopped immediately. If all prep
commands succeed, any daemons in the block are restarted, also in order of
occurrence. If multiple blocks are triggered by the same set of changes, they
too run in order, from top to bottom.

Here's a an example *ppow.toml* file. It runs the test suite whenever a .go
file changes, builds devd whenever a non-test file is changed, and keeps a
test instance running throughout.

```
[[block]]
include = ["**/*.go"]
[[block.prep]]
cmd = "go test @dirmods"

# Exclude all test files of the form *_test.go
[[block]]
include = ["**/.go"]
exclude = ["**/*_test.go"]
[[block.prep]]
cmd = "go install ./cmd/devd"
[[block.daemon]]
signal = "sigterm"
cmd = "devd -m ./tmp"
```

The **@dirmods** variable expands to a properly escaped list of all directories
containing changed files. When ppow is first run, this includes all directories
containing matching files. So, this means that ppow will run all tests on
startup, and then subsequently run the tests only for the affected module
whenever there's a change. There's a corresponding **@mods** variable that
contains all changed files.

Note the `signal = "sigterm"` option. When devd receives a SIGHUP
(the default signal sent by ppow), it triggers a browser livereload, rather
than exiting. This is what you want when devd is being used to serve a web
project you're hacking on, but when developing devd _itself_, we actually want
it to exit and restart to pick up changes. We therefore tell ppow to send a
SIGTERM to the daemon instead, which causes devd to exit and be restarted by
ppow.

By default ppow interprets commands using `sh`. Some other external shells
are also supported, and can be used by setting `shell` variable in your
"ppow.toml" file.


# File watch patterns

ppow batches up changes until there is a lull in filesystem activity - this
means that coherent processes like compilation and rendering that touch many
files are likely to trigger commands only once. Patterns therefore match on a
batch of changed files - when the first match in a batch is seen, the block is
triggered.

Patterns and the paths they match against are always in slash-delimited form,
even on Windows. Paths are cleaned and normalised being matched, with redundant
components removed. If the path is within the current working directory, the
normalised path is relative to the current working directory, otherwise it is
absolute. One subtlety is that this means that a pattern like `./*.js` will
never match, because inbound paths will not have a leading `./` component - just
use `*.js` instead.

## Exclusions

Exclusions are applied after all includes - that is, ppow collects all
files matching the include patterns, then removes files matching the exclude
patterns.

## Default ignore list

Common nuisance files like VCS directories, swap files, and so forth are
ignored by default. You can list the set of ignored patterns using the **-i**
flag to the ppow command. The default ignore patterns can be disabled using the
special **+noignore** flag, like so:

```
[[block]]
include = [".git/config"]
noignore = true
[[block.prep]]
cmd = "echo \"git config changed\""
```

## Empty match pattern

If no include pattern is specified, prep commands run once only at startup, and
daemons are restarted if they exit, but won't ever be explicitly signalled to
restart by ppow.

```
[[block]]
[[block.prep]]
cmd = "echo hello"
```

## Symlinks

ppow does not implicitly traverse symlinks. To monitor a symlink, split the path
specification and the matching pattern, like this:

```
[[block]]
include = ["mydir/symlinkdir", "foo.*"]
[[block.prep]]
cmd = "echo changed"
```

Behind the scenes, we resolve the symlinked directory as if it was specified
directly by the user. This means that if the symlink destination lies outside of
the current working directory, the resulting paths for matches, exclusions and
commands will be absolute.


## Syntax

File patterns support the following syntax:

Term          | Meaning
------------- | -------
`*`           | any sequence of non-path-separators
`**`          | any sequence of characters, including path separators
`?`           | any single non-path-separator character
`[class]`     | any single non-path-separator character against a class of characters
`{alt1,...}`  | any of the comma-separated alternatives - to avoid conflict with the block specification, patterns with curly-braces should be enclosed in quotes

Any character with a special meaning can be escaped with a backslash (`\`).
Character classes support the following:

Class      | Meaning
---------- | -------
`[abc]`    | any character within the set
`[a-z]`    | any character in the range
`[^class]` | any character which does *not* match the class


# Blocks

Each block contains optional match patterns, commands and block-scoped options.

```
[[block]]
[[block.prep]]
cmd = """echo "I'm now rebuilding" | tee /tmp/output"""
```

Within commands, the `@` character is treated specially, since it is the marker
for variable replacement. You can include a verbatim `@` symbol b escaping it
with a backslash, and backslashes preceding the `@` symbol can themselves be
escaped recursively.

```
[variables]
foo = "bar"

[[block]]
[[block.prep]]
cmd = "echo @foo" # bar

[[block]]
[[block.prep]]
cmd = "echo \@foo" # @foo

[[block]]
[[block.prep]]
cmd = "echo \\@foo" # \bar
```

## Prep commands

All prep commands in a block are run in order before any daemons are restarted.
If any prep command exits with an error, execution stops.

The following variables are automatically generated for prep commands

Variable      | Meaning
------------- | -------
@mods         | On first run, all files matching the block patterns. On subsequent change, a list of all modified files.
@confdir      | The absolute path of the directory that contains the current ppow config file.
@dirmods      | On first run, all directories containing files matching the block patterns. On subsequent change, a list of all directories containing modified files.

All file names in variables are relative to the current directory, and
shell-escaped for safety. All paths are in slash-delimited form on all
platforms.

Given a config file like this, ppow will run *eslint* on all .js files when
started, and then after that only run *eslint* on files if they change:

```
[[block]]
include = ["**/*.js"]
[[block.prep]]
cmd = "eslint @mods"
```

By default, prep commands are executed on the initial run of ppow. The
`onchange` option can be used to skip the initial run, and only execute when
there is a detected change.

```
[[block]]
include = ["**/*.go"]
[[block.prep]]
onchange = true
cmd = "go test"
```


## Daemon commands

Daemons are executed on startup, and are restarted by ppow whenever they exit.
When a block containing a daemon command is triggered, ppow sends a signal to
the daemon process group. If the signal causes the daemon to exit, it is
immediately restarted by ppow - however, it's also common for daemons to do
other useful things like reloading configuration in response to signals.

The default signal used is SIGHUP, but the signal can be controlled using
modifier flags, like so:

```
[[block]]
[[block.daemon]]
signal = "sigterm"
cmd = "mydaemon --config ./foo.config"
```

The following signals are supported: **sighup**, **sigterm**, **sigint**,
**sigkill**, **sigquit**, **sigusr1**, **sigusr2**, **sigwinch**.

Support for signals on Windows is limited. The signal type is ignored, and all
daemons are stopped and restarted when a signal would normally be sent.

The following variables are automatically generated for prep commands

Variable      | Meaning
------------- | -------
@confdir      | The absolute path of the directory that contains the current ppow config file.


## Controlling log headers

ppow outputs a short header on the terminal to show which command is responsible
for output. This header is calculated from the first non-whitespace line of the
command - backslash escapes are removed from the end of the line, comment
characters are removed from the beginning, and whitespace is stripped. Using the
fact that the shell itself permits comments, you can completely control the log
display name.

```
[[block]]

# This will show as "prep: mycommand"
[[block.prep]]
cmd = """
mycommand \
  --longoption 1 \
  --longoption 2
"""
# This will show as "prep: daemon 1"
[[block.prep]]
cmd = """
# daemon 1
mycommand \
  --longoption 1 \
  --longoption 2
"""
```

## Options

The only block option at the moment is **indir**, which controls the execution
directory of a block. ppow will change to this directory before executing
commands and daemons, and change back to the previous directory afterwards.

The directory specification follows the same conventions as commands.

```
[[block]]
indir = "./my/directory"
[[block.prep]]
cmd = "ls"
```


# Variables

Variables are declared as follows:

```
[variables]
variable = "value"
```

All values are strings and follow the same semantics as commands - that is,
they can have escaped line endings, or be quoted strings. Variables are read
once at startup, and it is an error to re-declare a variable that already exists.

You can use variables in commands like so:

```
[variables]
dst = "./build/dst"

[[block]]
include = ["**"]
[[block.prep]]
cmd = "ls @dst"
```

There is a special "shell" variable that determines which shell is used to
execute commands. Valid values are `sh` (the default), `bash` and
`powershell`. This variable is set as follows:

```
[variables]
shell = "bash"
```

# Desktop Notifications

When the **-n** flag is specified, ppow sends anything sent to *stderr* from any
prep command that exits abnormally to a desktop notifier. Since ppow commands
are shell scripts, you can redirect or manipulate output to entirely customise
what gets sent to notifiers as needed.

At the moment, we support [Growl](http://growl.info/) on OSX, and
[libnotify](https://launchpad.net/ubuntu/+source/libnotify) on Linux and other
Unix systems.

## Growl

For Growl to work, you will need Growl itself to be running, and have the
**growlnotify** command installed. Growlnotify is an additional tool that you
can download from the official [Growl
website](http://growl.info/downloads.php).


## Libnotify

Libnotify is a general notification framework available on most Unix-like
systems. ppow uses the **notify-send** command to send notifications using
libnotify. You'll need to use your system package manager to install
**libnotify**.


# Colour output in process logs

Some programs that have colourised output when run on the command-line don't
emit colour when run under ppow. Users might assume that ppow is stripping the
colour from the command output, but that is not the case. Well-behaved terminal
programs check whether they are connected to a terminal, and if not, disable
colour codes in their own output. It is possible to trick a program into
believing that a terminal is present through pseudo-terminal emulation, but this
is complex and platform dependent and is not a good fit for a simple, reliable
tool like ppow.

This leaves users with two options:

- Many tools that produce colour output also have a flag to force colour when no
  terminal is detected, and many logging libraries with human-friendly output do
  the same. The simplest solution is to work out how to force output and
  explicitly specify this in your ppow configuration.
- There are platform-specific tools you can interpose between ppow and the
  subprocess to emulate a terminal. One example is
  [unbuffer](https://linux.die.net/man/1/unbuffer) on Linux.


# Development

The scripts used to build this package for distribution can be found
[here](https://github.com/cortesi/godist).
