package commands

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/baptistax/rdlxd/internal/config"
	"github.com/baptistax/rdlxd/internal/download"
	"github.com/baptistax/rdlxd/internal/media"
	"github.com/baptistax/rdlxd/internal/media/resolvers"
	"github.com/baptistax/rdlxd/internal/output"
	"github.com/baptistax/rdlxd/internal/post"
	"github.com/baptistax/rdlxd/internal/reddit"
	"github.com/baptistax/rdlxd/internal/storage"
)

func RunDownload(args []string, stdout io.Writer) error {
	fs := newFlagSet("download")
	outDir := fs.String("out", "./output", "output directory")
	limit := fs.Int("limit", 100, "maximum posts to collect")
	includeNSFW := fs.Bool("include-nsfw", false, "include posts marked NSFW")
	excludeNSFW := fs.Bool("exclude-nsfw", false, "exclude posts marked NSFW")
	verbose := fs.Bool("verbose", false, "show more console details")
	if err := parseFlags(fs, args); err != nil {
		return ErrUsage
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("%w: rdlxd requires exactly one source", ErrUsage)
	}
	if *limit < 0 {
		return fmt.Errorf("limit must be zero or greater")
	}

	source, err := reddit.ParseSource(fs.Arg(0), *limit)
	if err != nil {
		return err
	}
	if *excludeNSFW {
		*includeNSFW = false
	}

	layout, err := storage.InitializeLayout(*outDir, storage.SourceSlug(source))
	if err != nil {
		return err
	}

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
	if err := state.CreateRun(runID, source.RawInput); err != nil {
		return err
	}
	if err := logger.Info(storage.LogEvent{
		RunID:   runID,
		Event:   "run_started",
		Message: "download run started",
	}); err != nil {
		return err
	}
	_ = logger.Info(storage.LogEvent{RunID: runID, Event: "source_parsed", Message: source.RawInput})

	ctx := context.Background()
	client, err := createRedditClient(ctx)
	if err != nil {
		_ = state.FinishRun(runID, "failed")
		return err
	}
	pipeline := buildMediaPipeline()
	if *verbose {
		fmt.Fprintf(stdout, "Pipeline resolvers: %s\n", pipeline.ResolverNames())
		fmt.Fprintf(stdout, "Auth mode: %s\n", authModeLabel(!client.PublicMode))
		fmt.Fprintf(stdout, "Include NSFW: %t\n", *includeNSFW)
	}
	client.Observer = func(event reddit.RequestEvent) {
		message := event.URL
		if event.Event == "rate_limit_observed" && event.RateLimit != nil {
			message = fmt.Sprintf("used=%.0f remaining=%.0f reset=%.0f", event.RateLimit.Used, event.RateLimit.Remaining, event.RateLimit.Reset)
		}
		logEvent := storage.LogEvent{RunID: runID, Event: event.Event, Message: message}
		if event.Err != nil {
			logEvent.ErrorCode = "reddit_request_error"
			_ = logger.Error(logEvent)
			return
		}
		_ = logger.Info(logEvent)
	}
	fetcher := reddit.ClientListingFetcher{Client: client}
	posts, err := fetcher.FetchPosts(ctx, *source)
	if err != nil {
		_ = state.FinishRun(runID, "failed")
		return friendlyRedditError(err)
	}

	processor := postProcessor{
		layout:     layout,
		state:      state,
		logger:     logger,
		runID:      runID,
		pipeline:   pipeline,
		downloader: download.NewDownloader(),
	}
	for _, redditPost := range posts {
		postID := post.PostFolderName(redditPost)
		_ = logger.Info(storage.LogEvent{RunID: runID, PostID: postID, Event: "post_discovered", Message: redditPost.Title})
		if err := processor.processPost(ctx, redditPost, postProcessOptions{IncludeNSFW: *includeNSFW}); err != nil {
			_ = logger.Error(storage.LogEvent{RunID: runID, PostID: postID, Event: "post_processing_failed", ErrorCode: "post_processing_failed", Retryable: true})
			_ = state.UpsertPost(storage.PostRecord{
				PostID:    postID,
				Name:      redditPost.Name,
				Permalink: redditPost.Permalink,
				Title:     redditPost.Title,
				Status:    string(media.StatusFailed),
				Substatus: "post_processing_failed",
				Reason:    err.Error(),
				Retryable: true,
			})
		}
	}

	summary, err := state.GetSummaryCounts()
	if err != nil {
		return err
	}

	if err := storage.WriteManifest(layout.ManifestPath, storage.Manifest{
		Version: "0.1",
		RunID:   runID,
		Source:  source.RawInput,
		Summary: summary,
	}); err != nil {
		return err
	}
	_ = logger.Info(storage.LogEvent{RunID: runID, Event: "manifest_written", Message: "manifest written"})
	failedRows, err := state.ListIncompletePosts()
	if err != nil {
		return err
	}
	if err := storage.WriteFailedJSON(layout.FailedPath, failedRows); err != nil {
		return err
	}
	_ = logger.Info(storage.LogEvent{RunID: runID, Event: "failed_list_written", Message: "failed list written"})
	if err := state.FinishRun(runID, "finished"); err != nil {
		return err
	}
	_ = logger.Info(storage.LogEvent{RunID: runID, Event: "run_finished", Message: "download run finished"})

	fmt.Fprint(stdout, output.FormatDownloadSummary(source.RawInput, summary, displayOutputPath(*outDir, storage.SourceSlug(source))))
	if len(failedRows) > 0 {
		fmt.Fprintln(stdout)
		fmt.Fprint(stdout, output.FormatFailedRows(failedRows))
	}
	return nil
}

func buildMediaPipeline() *media.Pipeline {
	currentPostResolvers := []media.MediaResolver{
		resolvers.RedditGalleryResolver{},
		resolvers.RedditVideoResolver{},
		resolvers.RedditImageURLResolver{},
		resolvers.RedditPreviewResolver{},
	}
	allResolvers := append([]media.MediaResolver{}, currentPostResolvers...)
	allResolvers = append(allResolvers, resolvers.NewCrosspostParentResolver(currentPostResolvers))
	allResolvers = append(allResolvers, resolvers.UnsupportedPostResolver{})
	return media.NewPipeline(allResolvers...)
}

func authModeLabel(useAuth bool) string {
	if useAuth {
		return "oauth"
	}
	return "best-effort"
}

func displayOutputPath(outDir, slug string) string {
	display := filepath.ToSlash(filepath.Join(outDir, slug))
	if strings.HasPrefix(outDir, "./") && !strings.HasPrefix(display, "./") {
		return "./" + display
	}
	return display
}

func createRedditClient(ctx context.Context) (*reddit.Client, error) {
	cfg := config.Default()
	client := reddit.NewClient(cfg.UserAgent)
	if base := strings.TrimSpace(getenv("RDLXD_REDDIT_BASE_URL")); base != "" {
		client.OAuthBaseURL = base
		client.PublicBaseURL = base
	}
	if base := strings.TrimSpace(getenv("REDDITDOWNLOADER_REDDIT_BASE_URL")); base != "" {
		client.OAuthBaseURL = base
		client.PublicBaseURL = base
	}
	store := config.FileTokenStore{Path: cfg.TokenPath}
	token, err := store.Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not load OAuth token: %w", err)
	}
	if token == nil || (token.AccessToken == "" && token.RefreshToken == "") {
		client.PublicMode = true
		return client, nil
	}
	flow := reddit.InstalledAppFlow{Config: reddit.OAuthConfig{
		ClientID:    cfg.ClientID,
		RedirectURI: defaultRedirectURI,
		Scopes:      cfg.Scopes,
		UserAgent:   cfg.UserAgent,
	}}
	refresh := func(ctx context.Context) (string, error) {
		if token.RefreshToken == "" {
			return "", reddit.AuthError{Message: "reddit OAuth refresh token not found; run: rdlxd auth"}
		}
		if cfg.ClientID == "" {
			return "", reddit.AuthError{Message: "reddit client id not configured; set REDDITDOWNLOADER_CLIENT_ID and run rdlxd auth"}
		}
		refreshed, err := flow.Refresh(ctx, token.RefreshToken)
		if err != nil {
			return "", err
		}
		if refreshed.RefreshToken == "" {
			refreshed.RefreshToken = token.RefreshToken
		}
		if len(refreshed.Scopes) == 0 {
			refreshed.Scopes = token.Scopes
		}
		if err := store.Save(ctx, *refreshed); err != nil {
			return "", err
		}
		token = refreshed
		return refreshed.AccessToken, nil
	}
	if token.AccessToken == "" || (!token.ExpiresAt.IsZero() && time.Now().UTC().After(token.ExpiresAt.Add(-60*time.Second))) {
		if _, err := refresh(ctx); err != nil {
			client.PublicMode = true
			return client, nil
		}
	}
	client.AccessToken = token.AccessToken
	client.RefreshToken = refresh
	return client, nil
}

func friendlyRedditError(err error) error {
	var authErr reddit.AuthError
	if errors.As(err, &authErr) {
		return authErr
	}
	var responseErr reddit.ResponseError
	if errors.As(err, &responseErr) {
		switch responseErr.Code {
		case "auth_required":
			return reddit.AuthError{Message: "reddit authentication failed; run: rdlxd auth"}
		case "private_or_forbidden":
			return fmt.Errorf("reddit source is private, forbidden, or requires authentication")
		case "not_found", "gone":
			return fmt.Errorf("reddit source was not found or is gone")
		case "rate_limited":
			return fmt.Errorf("reddit rate limit reached; retry later")
		default:
			if responseErr.Retryable {
				return fmt.Errorf("reddit request failed temporarily; retry later")
			}
		}
	}
	return err
}

func getenv(name string) string {
	return strings.TrimSpace(strings.Trim(os.Getenv(name), `"`))
}
