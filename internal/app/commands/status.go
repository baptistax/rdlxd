package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/baptistax/rdlxd/internal/download"
	"github.com/baptistax/rdlxd/internal/media"
	"github.com/baptistax/rdlxd/internal/output"
	"github.com/baptistax/rdlxd/internal/post"
	"github.com/baptistax/rdlxd/internal/storage"
)

func RunStatus(args []string, stdout io.Writer) error {
	fs := newFlagSet("status")
	if err := parseFlags(fs, args); err != nil {
		return ErrUsage
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("%w: status requires one output folder", ErrUsage)
	}
	layout := storage.LayoutFromSourceDir(fs.Arg(0))
	state, err := storage.OpenSQLiteState(layout.StatePath)
	if err != nil {
		return err
	}
	defer state.Close()
	summary, err := state.GetSummaryCounts()
	if err != nil {
		return err
	}
	fmt.Fprint(stdout, output.FormatStatusSummary(summary))
	return nil
}

func RunFailed(args []string, stdout io.Writer) error {
	fs := newFlagSet("failed")
	if err := parseFlags(fs, args); err != nil {
		return ErrUsage
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("%w: failed requires one output folder", ErrUsage)
	}
	layout := storage.LayoutFromSourceDir(fs.Arg(0))
	state, err := storage.OpenSQLiteState(layout.StatePath)
	if err != nil {
		return err
	}
	defer state.Close()
	rows, err := state.ListIncompletePosts()
	if err != nil {
		return err
	}
	fmt.Fprint(stdout, output.FormatFailedRows(rows))
	return nil
}

func RunRetry(args []string, stdout io.Writer) error {
	fs := newFlagSet("retry")
	if err := parseFlags(fs, args); err != nil {
		return ErrUsage
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("%w: retry requires one output folder", ErrUsage)
	}
	layout := storage.LayoutFromSourceDir(fs.Arg(0))
	state, err := storage.OpenSQLiteState(layout.StatePath)
	if err != nil {
		return err
	}
	defer state.Close()
	logger, err := storage.OpenLog(layout.LogsPath)
	if err != nil {
		return err
	}
	defer logger.Close()
	runID := storage.NewRunID()
	if err := state.CreateRun(runID, "retry"); err != nil {
		return err
	}
	rows, err := state.ListIncompletePosts()
	if err != nil {
		return err
	}
	processor := postProcessor{
		layout:     layout,
		state:      state,
		logger:     logger,
		runID:      runID,
		pipeline:   buildMediaPipeline(),
		downloader: download.NewDownloader(),
	}
	retried := 0
	skipped := 0
	for _, row := range rows {
		if !row.Retryable || row.Status == string(media.StatusUnsupported) {
			skipped++
			continue
		}
		current, err := readPostSidecar(layout, row.PostID)
		if err != nil {
			skipped++
			continue
		}
		if err := processor.processPost(context.Background(), current, postProcessOptions{IncludeNSFW: true}); err != nil {
			_ = logger.Error(storage.LogEvent{RunID: runID, PostID: row.PostID, Event: "retry_failed", ErrorCode: "retry_failed", Retryable: true})
			skipped++
			continue
		}
		retried++
	}
	summary, err := state.GetSummaryCounts()
	if err != nil {
		return err
	}
	if err := storage.WriteManifest(layout.ManifestPath, storage.Manifest{Version: "0.1", RunID: runID, Source: "retry", Summary: summary}); err != nil {
		return err
	}
	failedRows, err := state.ListIncompletePosts()
	if err != nil {
		return err
	}
	if err := storage.WriteFailedJSON(layout.FailedPath, failedRows); err != nil {
		return err
	}
	if err := state.FinishRun(runID, "finished"); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Retried: %d\nSkipped: %d\n\n", retried, skipped)
	fmt.Fprint(stdout, output.FormatStatusSummary(summary))
	return nil
}

func readPostSidecar(layout storage.Layout, postID string) (post.Post, error) {
	var zero post.Post
	postDir, err := storage.PostDir(layout, postID)
	if err != nil {
		return zero, err
	}
	data, err := os.ReadFile(filepath.Join(postDir, "post.json"))
	if err != nil {
		return zero, err
	}
	var sidecar postSidecar
	if err := json.Unmarshal(data, &sidecar); err == nil && sidecar.Post.Name != "" {
		return post.Normalize(sidecar.Post), nil
	}
	var direct post.Post
	if err := json.Unmarshal(data, &direct); err != nil {
		return zero, err
	}
	return post.Normalize(direct), nil
}
