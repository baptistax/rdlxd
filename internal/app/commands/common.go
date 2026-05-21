package commands

import (
	"errors"
	"flag"
	"strings"
)

var ErrUsage = errors.New("invalid command usage")

func newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(noopWriter{})
	return fs
}

func parseFlags(fs *flag.FlagSet, args []string) error {
	ordered := make([]string, 0, len(args))
	positionals := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			positionals = append(positionals, args[i+1:]...)
			break
		}
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			positionals = append(positionals, arg)
			continue
		}
		ordered = append(ordered, arg)
		if flagTakesValue(fs, arg) && !strings.Contains(arg, "=") && i+1 < len(args) {
			i++
			ordered = append(ordered, args[i])
		}
	}
	ordered = append(ordered, positionals...)
	return fs.Parse(ordered)
}

func flagTakesValue(fs *flag.FlagSet, arg string) bool {
	name := strings.TrimLeft(arg, "-")
	if index := strings.Index(name, "="); index >= 0 {
		name = name[:index]
	}
	flagValue := fs.Lookup(name)
	if flagValue == nil {
		return false
	}
	type boolFlag interface {
		IsBoolFlag() bool
	}
	if value, ok := flagValue.Value.(boolFlag); ok {
		return !value.IsBoolFlag()
	}
	return true
}

type noopWriter struct{}

func (noopWriter) Write(p []byte) (int, error) {
	return len(p), nil
}
