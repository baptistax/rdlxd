package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/baptistax/rdlxd/internal/download"
	"github.com/baptistax/rdlxd/internal/media"
	"github.com/baptistax/rdlxd/internal/post"
	"github.com/baptistax/rdlxd/internal/storage"
)

type postProcessor struct {
	layout     storage.Layout
	state      storage.StateStore
	logger     *storage.Log
	runID      string
	pipeline   *media.Pipeline
	downloader *download.Downloader
}

type postProcessOptions struct {
	IncludeNSFW bool
}

type postSidecar struct {
	Post       post.Post              `json:"post"`
	Status     string                 `json:"status"`
	Substatus  string                 `json:"substatus,omitempty"`
	Reason     string                 `json:"reason,omitempty"`
	Retryable  bool                   `json:"retryable"`
	Candidates []media.MediaCandidate `json:"candidates,omitempty"`
	Assets     []sidecarAsset         `json:"assets,omitempty"`
	Warnings   []string               `json:"warnings,omitempty"`
}

type sidecarAsset struct {
	AssetID     string `json:"asset_id"`
	CandidateID string `json:"candidate_id"`
	URL         string `json:"url,omitempty"`
	Status      string `json:"status"`
	Substatus   string `json:"substatus,omitempty"`
	Reason      string `json:"reason,omitempty"`
	Path        string `json:"path,omitempty"`
	BlobID      string `json:"blob_id,omitempty"`
	Retryable   bool   `json:"retryable"`
}

type assetOutcome struct {
	Asset     sidecarAsset
	Required  bool
	Status    string
	Substatus string
	Reason    string
	Retryable bool
	Notes     []string
}

type postStatus struct {
	Status    string
	Substatus string
	Reason    string
	Retryable bool
	Warnings  []string
}

func (p *postProcessor) processPost(ctx context.Context, rawPost post.Post, opts postProcessOptions) error {
	normalized := post.Normalize(rawPost)
	postName := post.PostFolderName(normalized)
	postDir, err := storage.PostDir(p.layout, postName)
	if err != nil {
		return err
	}
	mediaDir := p.layout.MediaDir
	if err := os.MkdirAll(mediaDir, 0755); err != nil {
		return err
	}
	if len(normalized.RawJSON) > 0 {
		if err := storage.AtomicWriteJSON(filepath.Join(postDir, "raw.json"), json.RawMessage(normalized.RawJSON)); err != nil {
			return err
		}
	}

	record := storage.PostRecord{
		PostID:    postName,
		Name:      normalized.Name,
		Permalink: normalized.Permalink,
		Title:     normalized.Title,
		Status:    string(media.StatusPartial),
		Substatus: "processing",
		Retryable: true,
	}
	if err := p.state.UpsertPost(record); err != nil {
		return err
	}
	_ = p.logger.Info(storage.LogEvent{RunID: p.runID, PostID: postName, Event: "post_persisted", Message: "post persisted"})

	if normalized.Over18 && !opts.IncludeNSFW {
		final := postStatus{Status: string(media.StatusUnsupported), Substatus: string(media.SubstatusNSFWExcluded), Reason: "NSFW post excluded; use --include-nsfw to process it", Retryable: false}
		return p.finishPost(postDir, normalized, final, nil, nil)
	}
	if post.IsDeletedOrRemoved(normalized) {
		final := postStatus{Status: string(media.StatusFailed), Substatus: string(media.SubstatusDeletedOrRemoved), Reason: "post is deleted or removed", Retryable: false}
		return p.finishPost(postDir, normalized, final, nil, nil)
	}

	result, err := p.pipeline.Resolve(ctx, media.PostContext{Post: normalized})
	if err != nil {
		return err
	}
	_ = p.logger.Info(storage.LogEvent{RunID: p.runID, PostID: postName, Event: "candidates_resolved", Message: fmt.Sprintf("%d candidates resolved", len(result.Candidates))})

	if len(result.Candidates) == 0 {
		final := statusForNoCandidates(normalized)
		return p.finishPost(postDir, normalized, final, nil, nil)
	}

	outcomes := make([]assetOutcome, 0, len(result.Candidates))
	for i := range result.Candidates {
		candidate := result.Candidates[i]
		outcome := p.processCandidate(ctx, normalized, candidate, mediaDir)
		result.Candidates[i].CandidateID = outcome.Asset.CandidateID
		outcomes = append(outcomes, outcome)
	}
	final := aggregatePostStatus(outcomes)
	return p.finishPost(postDir, normalized, final, result.Candidates, outcomes)
}

func (p *postProcessor) processCandidate(ctx context.Context, current post.Post, candidate media.MediaCandidate, mediaDir string) assetOutcome {
	postID := post.PostFolderName(current)
	if candidate.PostID == "" {
		candidate.PostID = postID
	}
	candidate.CandidateID = stableCandidateID(candidate)
	assetID := stableAssetID(postID, candidate)
	outcome := assetOutcome{
		Required: candidate.Required,
		Notes:    append([]string{}, candidate.Notes...),
		Asset: sidecarAsset{
			AssetID:     assetID,
			CandidateID: candidate.CandidateID,
			URL:         candidate.URL,
			Status:      string(media.StatusPartial),
			Retryable:   true,
		},
	}

	if candidate.Unsupported {
		outcome.Status = string(media.StatusUnsupported)
		outcome.Substatus = firstNonEmpty(candidate.UnsupportedSubstatus, string(media.SubstatusNoMediaCandidate))
		outcome.Reason = firstNonEmpty(candidate.UnsupportedReason, "unsupported post")
		outcome.Retryable = false
		outcome.Asset.Status = outcome.Status
		outcome.Asset.Substatus = outcome.Substatus
		outcome.Asset.Reason = outcome.Reason
		outcome.Asset.Retryable = false
		p.persistAssetAttempt(candidate, outcome, download.Result{}, nil)
		return outcome
	}
	if candidate.MediaKind == media.MediaKindText {
		return p.saveTextCandidate(current, candidate, mediaDir, outcome)
	}
	if candidate.URL == "" {
		outcome.Status = string(media.StatusFailed)
		outcome.Substatus = string(media.SubstatusNoMediaCandidate)
		outcome.Reason = "candidate has no downloadable URL"
		outcome.Retryable = false
		p.persistAssetAttempt(candidate, outcome, download.Result{}, nil)
		return outcome
	}
	if reused, ok := p.reuseExistingAsset(candidate, mediaDir, outcome); ok {
		return reused
	}

	_ = p.logger.Info(storage.LogEvent{RunID: p.runID, PostID: postID, AssetID: assetID, Event: "asset_download_started", Message: "asset download started"})
	result, err := p.downloader.DownloadCandidateToStore(ctx, candidate, mediaDir, p.layout.TempDir, p.layout.BlobsDir)
	if err != nil {
		outcome.Status, outcome.Substatus, outcome.Reason, outcome.Retryable = classifyDownloadFailure(err)
		outcome.Asset.Status = outcome.Status
		outcome.Asset.Substatus = outcome.Substatus
		outcome.Asset.Reason = outcome.Reason
		outcome.Asset.Retryable = outcome.Retryable
		p.persistAssetAttempt(candidate, outcome, result, err)
		_ = p.logger.Error(storage.LogEvent{RunID: p.runID, PostID: postID, AssetID: assetID, Event: "asset_download_failed", ErrorCode: outcome.Substatus, Retryable: outcome.Retryable})
		return outcome
	}

	outcome.Status = string(media.StatusDownloaded)
	outcome.Substatus = ""
	if result.ReusedBlob {
		outcome.Substatus = string(media.SubstatusBlobReused)
	}
	outcome.Reason = ""
	outcome.Retryable = false
	outcome.Asset.Status = outcome.Status
	outcome.Asset.Substatus = outcome.Substatus
	outcome.Asset.Path = result.Path
	outcome.Asset.BlobID = result.BlobID
	outcome.Asset.Retryable = false
	p.persistAssetAttempt(candidate, outcome, result, nil)
	_ = p.logger.Info(storage.LogEvent{RunID: p.runID, PostID: postID, AssetID: assetID, Event: "asset_download_completed", Message: "asset download completed"})
	return outcome
}

func (p *postProcessor) saveTextCandidate(current post.Post, candidate media.MediaCandidate, mediaDir string, outcome assetOutcome) assetOutcome {
	text := current.SelfText
	if text == "" {
		text = current.Title
	}
	finalPath, err := download.SafeFinalPath(mediaDir, download.FinalFileName(candidate, "text/plain"))
	if err != nil {
		outcome.Status = string(media.StatusFailed)
		outcome.Substatus = "text_path_error"
		outcome.Reason = err.Error()
		outcome.Retryable = false
		p.persistAssetAttempt(candidate, outcome, download.Result{}, err)
		return outcome
	}
	partialPath := finalPath + ".partial"
	if err := os.WriteFile(partialPath, []byte(text), 0644); err != nil {
		outcome.Status = string(media.StatusFailed)
		outcome.Substatus = "text_write_error"
		outcome.Reason = err.Error()
		outcome.Retryable = true
		p.persistAssetAttempt(candidate, outcome, download.Result{Path: finalPath, PartialPath: partialPath}, err)
		return outcome
	}
	if err := download.PromoteVerifiedFile(partialPath, finalPath); err != nil {
		outcome.Status = string(media.StatusFailed)
		outcome.Substatus = "text_promote_error"
		outcome.Reason = err.Error()
		outcome.Retryable = true
		p.persistAssetAttempt(candidate, outcome, download.Result{Path: finalPath, PartialPath: partialPath}, err)
		return outcome
	}
	sha, _ := download.SHA256File(finalPath)
	blobID := ""
	if sha != "" {
		blobID, _ = download.BlobIDFromSHA256(sha)
	}
	outcome.Status = string(media.StatusDownloaded)
	outcome.Substatus = string(media.SubstatusTextPostSaved)
	outcome.Reason = ""
	outcome.Retryable = false
	outcome.Asset.Status = outcome.Status
	outcome.Asset.Substatus = outcome.Substatus
	outcome.Asset.Path = finalPath
	outcome.Asset.BlobID = blobID
	outcome.Asset.Retryable = false
	p.persistAssetAttempt(candidate, outcome, download.Result{Path: finalPath, BlobID: blobID, SHA256: sha}, nil)
	return outcome
}

func (p *postProcessor) reuseExistingAsset(candidate media.MediaCandidate, mediaDir string, outcome assetOutcome) (assetOutcome, bool) {
	fileName := download.FinalFileName(candidate, candidate.ExpectedContentType)
	finalPath, err := download.SafeFinalPath(mediaDir, fileName)
	if err != nil {
		return outcome, false
	}
	info, err := os.Stat(finalPath)
	if err != nil || info.IsDir() || info.Size() == 0 {
		return outcome, false
	}
	sha, err := download.SHA256File(finalPath)
	if err != nil {
		return outcome, false
	}
	blobID, err := download.BlobIDFromSHA256(sha)
	if err != nil {
		return outcome, false
	}
	blobPath, err := storage.SafeJoin(p.layout.BlobsDir, blobID)
	if err == nil {
		if _, statErr := os.Stat(blobPath); errors.Is(statErr, os.ErrNotExist) {
			_ = copyFile(finalPath, blobPath)
		}
		_ = p.state.UpsertBlob(storage.BlobRecord{BlobID: blobID, SHA256: sha, Size: info.Size(), Path: blobPath})
	}
	outcome.Status = string(media.StatusDownloaded)
	outcome.Substatus = string(media.SubstatusFileReused)
	outcome.Retryable = false
	outcome.Asset.Status = outcome.Status
	outcome.Asset.Substatus = outcome.Substatus
	outcome.Asset.Path = finalPath
	outcome.Asset.BlobID = blobID
	outcome.Asset.Retryable = false
	p.persistAssetAttempt(candidate, outcome, download.Result{Path: finalPath, BlobPath: blobPath, BlobID: blobID, SHA256: sha, Size: info.Size()}, nil)
	return outcome, true
}

func (p *postProcessor) persistAssetAttempt(candidate media.MediaCandidate, outcome assetOutcome, result download.Result, err error) {
	postID := candidate.PostID
	if postID == "" {
		postID = outcome.Asset.AssetID
	}
	_ = p.state.UpsertAsset(storage.AssetRecord{
		AssetID:     outcome.Asset.AssetID,
		PostID:      postID,
		CandidateID: candidate.CandidateID,
		URL:         candidate.URL,
		Status:      outcome.Status,
		Substatus:   outcome.Substatus,
		BlobID:      outcome.Asset.BlobID,
		Path:        outcome.Asset.Path,
		Retryable:   outcome.Retryable,
		Reason:      outcome.Reason,
	})
	if result.BlobID != "" && result.SHA256 != "" && result.BlobPath != "" {
		_ = p.state.UpsertBlob(storage.BlobRecord{BlobID: result.BlobID, SHA256: result.SHA256, Size: result.Size, Path: result.BlobPath})
	}
	message := ""
	if err != nil {
		message = err.Error()
	}
	_ = p.state.RecordAttempt(storage.AttemptRecord{
		AttemptID:       stableAttemptID(p.runID, outcome.Asset.AssetID),
		PostID:          postID,
		AssetID:         outcome.Asset.AssetID,
		RunID:           p.runID,
		Status:          outcome.Status,
		ErrorCode:       outcome.Substatus,
		Retryable:       outcome.Retryable,
		Message:         message,
		ContentLength:   result.Size,
		BytesDownloaded: result.Size,
		PartialPath:     result.PartialPath,
		LastHTTPStatus:  result.HTTPStatus,
	})
}

func (p *postProcessor) finishPost(postDir string, current post.Post, final postStatus, candidates []media.MediaCandidate, outcomes []assetOutcome) error {
	postID := post.PostFolderName(current)
	if err := p.state.MarkPostStatus(postID, final.Status, final.Substatus, final.Reason, final.Retryable); err != nil {
		return err
	}
	assets := make([]sidecarAsset, 0, len(outcomes))
	for _, outcome := range outcomes {
		assets = append(assets, outcome.Asset)
	}
	sidecar := postSidecar{
		Post:       current,
		Status:     final.Status,
		Substatus:  final.Substatus,
		Reason:     final.Reason,
		Retryable:  final.Retryable,
		Candidates: candidates,
		Assets:     assets,
		Warnings:   final.Warnings,
	}
	if err := storage.AtomicWriteJSON(filepath.Join(postDir, "post.json"), sidecar); err != nil {
		return err
	}
	_ = p.logger.Info(storage.LogEvent{RunID: p.runID, PostID: postID, Event: "post_status_updated", Message: final.Status, Retryable: final.Retryable})
	return nil
}

func statusForNoCandidates(p post.Post) postStatus {
	return postStatus{
		Status:    string(media.StatusUnsupported),
		Substatus: string(media.SubstatusNoMediaCandidate),
		Reason:    "no supported media candidates",
		Retryable: false,
	}
}

func aggregatePostStatus(outcomes []assetOutcome) postStatus {
	selected := requiredOutcomes(outcomes)
	if len(selected) == 0 {
		return postStatus{Status: string(media.StatusUnsupported), Substatus: string(media.SubstatusNoMediaCandidate), Reason: "no supported media candidates", Retryable: false}
	}
	downloaded := 0
	unsupported := 0
	retryable := false
	warnings := map[string]struct{}{}
	previewOnly := false
	videoMayBeSilent := false
	firstReason := ""
	firstSubstatus := ""
	for _, outcome := range selected {
		if outcome.Status == string(media.StatusDownloaded) {
			downloaded++
		}
		if outcome.Status == string(media.StatusUnsupported) {
			unsupported++
		}
		if outcome.Retryable {
			retryable = true
		}
		for _, note := range outcome.Notes {
			if note == string(media.SubstatusVideoMayBeSilent) {
				warnings[note] = struct{}{}
				videoMayBeSilent = true
			}
			if note == string(media.SubstatusPreviewOnly) {
				warnings[note] = struct{}{}
				previewOnly = true
			}
		}
		if firstReason == "" && outcome.Reason != "" {
			firstReason = outcome.Reason
		}
		if firstSubstatus == "" && outcome.Substatus != "" {
			firstSubstatus = outcome.Substatus
		}
	}
	warningList := sortedWarningKeys(warnings)
	if downloaded == len(selected) {
		if previewOnly {
			return postStatus{
				Status:    string(media.StatusPartial),
				Substatus: string(media.SubstatusPreviewOnly),
				Reason:    "only preview-quality media was available",
				Retryable: false,
				Warnings:  warningList,
			}
		}
		if videoMayBeSilent {
			return postStatus{
				Status:    string(media.StatusPartial),
				Substatus: string(media.SubstatusVideoMayBeSilent),
				Reason:    "video audio was not verified",
				Retryable: false,
				Warnings:  warningList,
			}
		}
		return postStatus{Status: string(media.StatusDownloaded)}
	}
	if downloaded > 0 {
		return postStatus{
			Status:    string(media.StatusPartial),
			Substatus: string(media.SubstatusSomeCandidatesFailed),
			Reason:    firstNonEmpty(firstReason, "some candidates failed"),
			Retryable: retryable,
			Warnings:  warningList,
		}
	}
	if unsupported == len(selected) {
		return postStatus{
			Status:    string(media.StatusUnsupported),
			Substatus: firstNonEmpty(firstSubstatus, string(media.SubstatusExternalPageUnsupported)),
			Reason:    firstNonEmpty(firstReason, "unsupported media"),
			Retryable: false,
			Warnings:  warningList,
		}
	}
	return postStatus{
		Status:    string(media.StatusFailed),
		Substatus: string(media.SubstatusAllCandidatesFailed),
		Reason:    firstNonEmpty(firstReason, "all candidates failed"),
		Retryable: retryable,
		Warnings:  warningList,
	}
}

func requiredOutcomes(outcomes []assetOutcome) []assetOutcome {
	var selected []assetOutcome
	for _, outcome := range outcomes {
		if outcome.Required {
			selected = append(selected, outcome)
		}
	}
	if len(selected) > 0 {
		return selected
	}
	return outcomes
}

func classifyDownloadFailure(err error) (string, string, string, bool) {
	var contentTypeErr download.ContentTypeError
	if errors.As(err, &contentTypeErr) {
		return string(media.StatusUnsupported), contentTypeErr.Substatus, contentTypeErr.Error(), false
	}
	var statusErr download.HTTPStatusError
	if errors.As(err, &statusErr) {
		switch statusErr.StatusCode {
		case 403:
			return string(media.StatusFailed), string(media.SubstatusAuthRequired), "media request was forbidden", false
		case 404, 410:
			return string(media.StatusFailed), string(media.SubstatusDeletedOrRemoved), "media resource is gone", false
		case 429:
			return string(media.StatusFailed), string(media.SubstatusRateLimited), "media request was rate limited", true
		default:
			if statusErr.Retryable {
				return string(media.StatusFailed), "retryable_http_error", statusErr.Error(), true
			}
			return string(media.StatusFailed), "http_error", statusErr.Error(), false
		}
	}
	retryable := download.IsRetryableError(err)
	substatus := "download_error"
	if retryable {
		substatus = "retryable_download_error"
	}
	return string(media.StatusFailed), substatus, err.Error(), retryable
}

func stableCandidateID(candidate media.MediaCandidate) string {
	base := strings.Join([]string{
		candidate.PostID,
		candidate.URL,
		candidate.ResolverName,
		candidate.SourceField,
		fmt.Sprintf("%d", candidate.Index),
	}, "\x1f")
	return "candidate_" + shortID(base)
}

func stableAssetID(postID string, candidate media.MediaCandidate) string {
	base := strings.Join([]string{postID, candidate.CandidateID, fmt.Sprintf("%d", candidate.Index)}, "\x1f")
	return "asset_" + shortID(base)
}

func stableAttemptID(runID, assetID string) string {
	return "attempt_" + shortID(runID+"\x1f"+assetID+"\x1f"+fmt.Sprintf("%d", time.Now().UnixNano()))
}

func shortID(value string) string {
	var hash uint64 = 1469598103934665603
	for _, b := range []byte(value) {
		hash ^= uint64(b)
		hash *= 1099511628211
	}
	return fmt.Sprintf("%016x", hash)
}

func copyFile(srcPath, dstPath string) error {
	if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
		return err
	}
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()
	temp, err := os.CreateTemp(filepath.Dir(dstPath), "."+filepath.Base(dstPath)+".*.tmp")
	if err != nil {
		return err
	}
	tempName := temp.Name()
	removeTemp := true
	defer func() {
		if removeTemp {
			_ = os.Remove(tempName)
		}
	}()
	if _, err := io.Copy(temp, src); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := os.Remove(dstPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.Rename(tempName, dstPath); err != nil {
		return err
	}
	removeTemp = false
	return nil
}

func sortedWarningKeys(values map[string]struct{}) []string {
	if len(values) == 0 {
		return nil
	}
	warnings := make([]string, 0, len(values))
	for value := range values {
		warnings = append(warnings, value)
	}
	sort.Strings(warnings)
	return warnings
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
