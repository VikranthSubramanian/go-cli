package cli

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

// Cmd is an executable CLI command.
type Cmd interface {
	Info() *Info              // Get command information
	Main(args []string) error // Run command with positional arguments in args
}

// Helper is an optional command interface for providing help information.
type Helper interface {
	Cmd
	Help(w *Writer) // Write command help to w
}

// nameSep is the command name separator.
const nameSep = '|'

// Info contains command metadata. Commands are typically defined by assigning
// the value returned from Info.Add() to a global variable, which is then
// returned from Cmd.Info():
//
//	var info = cli.Main.Add(&cli.Info{
//		Name: "cmd",
//		New:  func() cli.Cmd { return &cmd{Opt: "default-value"} },
//	})
//
//	type cmd struct{ Opt string `flag:"Option description"` }
//
//	func (*cmd) Info() *cli.Info          { return info }
//	func (*cmd) Main(args []string) error { return nil }
type Info struct {
	Name    string     // '|'-separated command name and optional aliases
	Usage   string     // Option and argument syntax
	Summary string     // One-line description without trailing period
	MinArgs int        // Minimum number of positional arguments
	MaxArgs int        // Maximum number of positional arguments
	Hide    bool       // Hide from command list
	New     func() Cmd // Constructor (optional for parent commands)

	parent *Info            // Parent command
	cmds   map[string]*Info // Sub-commands
}

// Add registers command ci with parent p.
func (p *Info) Add(ci *Info) *Info {
	if ci.New == nil {
		ci.New = func() Cmd { return (*nilCmd)(ci) }
	}
	if ci.parent != nil {
		panic("cli: command already added to a parent: " + ci.Name)
	}
	if ci.parent = p; p.cmds == nil {
		p.cmds = make(map[string]*Info)
	}
	for _, name := range strings.Split(ci.Name, string(nameSep)) {
		if name == "" {
			panic("cli: missing command name")
		}
		if _, dup := p.cmds[name]; dup {
			panic("cli: duplicate command name: " + name)
		}
		p.cmds[name] = ci
	}
	return ci
}

// PrimaryName returns the first entry in ci.Name.
func (ci *Info) PrimaryName() string {
	if i := strings.IndexByte(ci.Name, nameSep); i != -1 {
		return ci.Name[:i]
	}
	return ci.Name
}

// Parse instantiates the requested command and parses the arguments. It returns
// the command, positional arguments, and any UsageError or flag.ErrHelp.
func (ci *Info) Parse(args []string) (Cmd, []string, error) {
	// Find sub-command
	var err error
	for len(args) > 0 && ci.cmds != nil {
		if isHelp(args[0]) {
			err = flag.ErrHelp
		} else if sub := ci.cmds[args[0]]; sub != nil {
			ci = sub
		} else {
			err = Errorf("unknown command %q", args[0])
			break
		}
		args = args[1:]
	}
	if err == nil && len(args) > 0 && isHelp(args[0]) {
		err = flag.ErrHelp
	}

	// Parse options
	cmd := ci.New()
	if err == nil && len(args) > 0 {
		fs := NewFlagSet(cmd)
		if err = fs.Parse(args); err != nil && err != flag.ErrHelp {
			err = UsageError(err.Error())
		}
		args = fs.Args()
	}

	// Check positional argument count
	if err != nil {
		args = nil
	} else if ci.MinArgs == ci.MaxArgs && len(args) != ci.MinArgs {
		if ci.MinArgs <= 0 {
			err = Error("command does not accept any arguments")
		} else {
			err = Errorf("command requires %d argument(s)", ci.MinArgs)
		}
	} else if len(args) < ci.MinArgs {
		err = Errorf("command requires at least %d argument(s)", ci.MinArgs)
	} else if ci.MinArgs < ci.MaxArgs && ci.MaxArgs < len(args) {
		err = Errorf("command accepts at most %d argument(s)", ci.MaxArgs)
	}
	return cmd, args, err
}

// Run parses the arguments, runs the requested commands, and terminates the
// process via Exit.
func (ci *Info) Run(args []string) {
	cmd, args, err := ci.Parse(args)
	if err == nil {
		if err = cmd.Main(args); err == nil {
			Exit(0)
			return
		}
	}
	if ci = cmd.Info(); err == flag.ErrHelp {
		w := newWriter(ci)
		defer w.done(os.Stderr, 0)
		w.help()
	} else if e, ok := err.(UsageError); ok {
		w := newWriter(ci)
		defer w.done(os.Stderr, 2)
		w.error(string(e))
	} else {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		Exit(1)
	}
}

// fullName returns the fully qualified command name consisting of the prefix,
// primary names of all parents, the primary command name, and any aliases.
func (ci *Info) fullName(prefix string) string {
	var buf [64]byte
	var walk func(ci *Info)
	b := append(buf[:0], prefix...)
	walk = func(ci *Info) {
		if ci != nil && ci.Name != "" {
			walk(ci.parent)
			b = append(append(b, ' '), ci.PrimaryName()...)
		}
	}
	if walk(ci.parent); ci.Name != "" {
		if b = append(b, ' '); strings.IndexByte(ci.Name, nameSep) == -1 {
			b = append(b, ci.Name...)
		} else {
			b = append(append(append(b, '{'), ci.Name...), '}')
		}
	}
	return strings.TrimSpace(string(b))
}