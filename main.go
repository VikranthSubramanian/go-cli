// Package cli provides tools for defining command line interfaces.
package cli

import (
	"fmt"
	"os"
	"path/filepath"
)

// Bin is the name of the executing binary.
var Bin = filepath.Base(os.Args[0])

// Exit is called by Info.Run() to terminate the process.
var Exit = os.Exit

// Main may be used as the common root of all commands in a CLI program. It is
// normally called from main() as follows:
//
//	func main() {
//		cli.Main.Run(os.Args[1:])
//	}
var Main Info

func init() { Main.New = func() Cmd { return (*nilCmd)(&Main) } }

// UsageError reports a problem with command options or positional arguments.
type UsageError string

// Error implements error interface.
func (e UsageError) Error() string { return string(e) }

// Error returns a new UsageError, formatting arguments via fmt.Sprintln.
func Error(v ...interface{}) UsageError {
	if len(v) == 1 {
		if s, ok := v[0].(string); ok {
			return UsageError(s)
		}
	}
	s := fmt.Sprintln(v...)
	return UsageError(s[:len(s)-1])
}

// Errorf returns a new UsageError, formatting arguments via fmt.Sprintf.
func Errorf(format string, v ...interface{}) UsageError {
	return UsageError(fmt.Sprintf(format, v...))
}

// nilCmd implements Cmd interface for commands without their own constructor.
type nilCmd Info

func (cmd *nilCmd) Info() *Info { return (*Info)(cmd) }

func (cmd *nilCmd) Main(args []string) error {
	w := newWriter(cmd.Info())
	defer w.done(os.Stderr, 2)
	if cmd.cmds == nil {
		w.WriteString("Command not implemented\n")
	} else {
		w.WriteString("Specify command:\n")
		w.commands()
	}
	return nil
}
