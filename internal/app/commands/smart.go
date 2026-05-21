package commands

import (
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/baptistax/rdlxd/internal/output"
	"github.com/baptistax/rdlxd/internal/storage"
)

func RunSmart(args []string, stdout io.Writer) error {
	fs := newFlagSet("rdlxd")
	outDir := fs.String("out", "./output", "output directory")
	limit := fs.Int("limit", 100, "maximum posts to collect")
	includeNSFW := fs.Bool("include-nsfw", false, "include posts marked NSFW")
	excludeNSFW := fs.Bool("exclude-nsfw", false, "exclude posts marked NSFW")
	verbose := fs.Bool("verbose", false, "show more console details")
	if err := parseFlags(fs, args); err != nil {
		return ErrUsage
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("%w: rdlxd requires exactly one source or output folder", ErrUsage)
	}

	target := fs.Arg(0)
	if isOutputFolder(target) {
		return runOutputFolder(target, stdout)
	}
	if isExistingDirectory(target) {
		return fmt.Errorf("folder does not contain rdlxd state: %s", target)
	}

	downloadArgs := []string{
		target,
		"--out", *outDir,
		"--limit", strconv.Itoa(*limit),
	}
	if *includeNSFW {
		downloadArgs = append(downloadArgs, "--include-nsfw")
	}
	if *excludeNSFW {
		downloadArgs = append(downloadArgs, "--exclude-nsfw")
	}
	if *verbose {
		downloadArgs = append(downloadArgs, "--verbose")
	}
	return RunDownload(downloadArgs, stdout)
}

func isOutputFolder(path string) bool {
	layout := storage.LayoutFromSourceDir(path)
	info, err := os.Stat(layout.StatePath)
	return err == nil && !info.IsDir()
}

func isExistingDirectory(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func runOutputFolder(path string, stdout io.Writer) error {
	layout := storage.LayoutFromSourceDir(path)
	state, err := storage.OpenSQLiteState(layout.StatePath)
	if err != nil {
		return err
	}
	defer state.Close()

	summary, err := state.GetSummaryCounts()
	if err != nil {
		return err
	}
	rows, err := state.ListIncompletePosts()
	if err != nil {
		return err
	}
	fmt.Fprint(stdout, output.FormatStatusSummary(summary))
	fmt.Fprintln(stdout)
	fmt.Fprint(stdout, output.FormatFailedRows(rows))
	return nil
}
