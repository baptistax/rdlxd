package app

import (
	"errors"
	"fmt"
	"io"

	"github.com/baptistax/rdlxd/internal/app/commands"
)

type App struct {
	stdout io.Writer
	stderr io.Writer
}

func New(stdout, stderr io.Writer) *App {
	return &App{stdout: stdout, stderr: stderr}
}

func (a *App) Run(args []string) error {
	if len(args) == 0 {
		return a.printUsage()
	}

	switch args[0] {
	case "auth":
		return commands.RunAuth(args[1:], a.stdout)
	case "download":
		return commands.RunDownload(args[1:], a.stdout)
	case "status":
		return commands.RunStatus(args[1:], a.stdout)
	case "failed":
		return commands.RunFailed(args[1:], a.stdout)
	case "retry":
		return commands.RunRetry(args[1:], a.stdout)
	case "help", "-h", "--help":
		return a.printUsage()
	default:
		return commands.RunSmart(args, a.stdout)
	}
}

func (a *App) PrintError(err error) {
	if err == nil {
		return
	}
	if errors.Is(err, commands.ErrUsage) {
		fmt.Fprintln(a.stderr, err)
		return
	}
	fmt.Fprintf(a.stderr, "Error: %v\n", err)
}

func (a *App) printUsage() error {
	_, err := fmt.Fprintln(a.stdout, `Usage:
  rdlxd auth
  rdlxd download <source> [--out ./output] [--limit 100] [--include-nsfw] [--verbose]
  rdlxd status <output-folder>
  rdlxd failed <output-folder>
  rdlxd retry <output-folder>
  rdlxd <source> [--out ./output] [--limit 100] [--include-nsfw] [--verbose]
  rdlxd <output-folder> [--verbose]`)
	return err
}
