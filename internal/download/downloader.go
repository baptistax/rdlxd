package download

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/baptistax/rdlxd/internal/media"
	"github.com/baptistax/rdlxd/internal/storage"
)

type Downloader struct {
	HTTPClient *http.Client
}

type Result struct {
	Path        string
	PartialPath string
	BlobPath    string
	BlobID      string
	SHA256      string
	Size        int64
	ContentType string
	HTTPStatus  int
	ReusedBlob  bool
	ReusedFile  bool
}

func NewDownloader() *Downloader {
	return &Downloader{HTTPClient: &http.Client{Timeout: 60 * time.Second}}
}

func (d *Downloader) DownloadCandidate(ctx context.Context, candidate media.MediaCandidate, mediaDir string) (Result, error) {
	return d.DownloadCandidateToStore(ctx, candidate, mediaDir, mediaDir, "")
}

func (d *Downloader) DownloadCandidateToStore(ctx context.Context, candidate media.MediaCandidate, mediaDir, tempDir, blobsDir string) (Result, error) {
	var result Result
	if candidate.URL == "" {
		return result, fmt.Errorf("candidate url is required")
	}
	policy := DefaultRetryPolicy()
	var lastErr error
	for attempt := 1; attempt <= policy.MaxAttempts; attempt++ {
		result, lastErr = d.downloadCandidateOnce(ctx, candidate, mediaDir, tempDir, blobsDir)
		if lastErr == nil {
			return result, nil
		}
		if !policy.CanRetry(attempt, IsRetryableError(lastErr)) {
			break
		}
		if err := sleep(ctx, backoffDelay(attempt, RetryAfter(lastErr))); err != nil {
			return result, err
		}
	}
	return result, lastErr
}

func (d *Downloader) downloadCandidateOnce(ctx context.Context, candidate media.MediaCandidate, mediaDir, tempDir, blobsDir string) (Result, error) {
	var result Result
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, candidate.URL, nil)
	if err != nil {
		return result, err
	}
	req.Header.Set("User-Agent", "rdlxd/0.1")
	resp, err := d.httpClient().Do(req)
	if err != nil {
		return result, NetworkError{Err: err}
	}
	defer resp.Body.Close()

	result.HTTPStatus = resp.StatusCode
	result.ContentType = resp.Header.Get("Content-Type")
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return result, HTTPStatusError{
			StatusCode: resp.StatusCode,
			Retryable:  isRetryableHTTPStatus(resp.StatusCode),
			RetryAfter: parseRetryAfter(resp.Header.Get("Retry-After")),
		}
	}
	if err := ValidateCandidateContentType(candidate, result.ContentType); err != nil {
		return result, err
	}

	fileName := FinalFileName(candidate, result.ContentType)
	finalPath, err := SafeFinalPath(mediaDir, fileName)
	if err != nil {
		return result, err
	}
	partialPath, err := candidatePartialPath(tempDir, candidate)
	if err != nil {
		return result, err
	}
	result.PartialPath = partialPath

	size, err := DownloadToPartial(resp.Body, partialPath)
	if err != nil {
		return result, NetworkError{Err: err}
	}
	result.Size = size
	if resp.ContentLength >= 0 && resp.ContentLength != size {
		return result, ContentLengthError{Expected: resp.ContentLength, Actual: size}
	}

	sha, err := SHA256File(partialPath)
	if err != nil {
		return result, err
	}
	result.SHA256 = sha
	blobID, err := BlobIDFromSHA256(sha)
	if err != nil {
		return result, err
	}
	result.BlobID = blobID
	if blobsDir != "" {
		blobPath, err := storage.SafeJoin(blobsDir, blobID)
		if err != nil {
			return result, err
		}
		result.BlobPath = blobPath
		if _, err := os.Stat(blobPath); err == nil {
			result.ReusedBlob = true
			_ = os.Remove(partialPath)
		} else if errors.Is(err, os.ErrNotExist) {
			if err := PromoteVerifiedFile(partialPath, blobPath); err != nil {
				return result, err
			}
		} else {
			return result, err
		}
		if err := materializeBlob(blobPath, finalPath); err != nil {
			return result, err
		}
	} else {
		if err := PromoteVerifiedFile(partialPath, finalPath); err != nil {
			return result, err
		}
	}
	result.Path = finalPath
	return result, nil
}

func DownloadToPartial(reader io.Reader, partialPath string) (int64, error) {
	if reader == nil {
		return 0, fmt.Errorf("reader is required")
	}
	if err := os.MkdirAll(filepath.Dir(partialPath), 0755); err != nil {
		return 0, err
	}
	file, err := os.Create(partialPath)
	if err != nil {
		return 0, err
	}
	written, copyErr := io.Copy(file, reader)
	closeErr := file.Close()
	if copyErr != nil {
		return written, copyErr
	}
	if closeErr != nil {
		return written, closeErr
	}
	return written, nil
}

func PromoteVerifiedFile(partialPath, finalPath string) error {
	if partialPath == "" || finalPath == "" {
		return fmt.Errorf("partial and final paths are required")
	}
	if err := os.MkdirAll(filepath.Dir(finalPath), 0755); err != nil {
		return err
	}
	if err := os.Remove(finalPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return os.Rename(partialPath, finalPath)
}

func partialPathFor(finalPath string) (string, error) {
	if finalPath == "" {
		return "", fmt.Errorf("final path is required")
	}
	return finalPath + ".partial", nil
}

func candidatePartialPath(tempDir string, candidate media.MediaCandidate) (string, error) {
	if tempDir == "" {
		return partialPathFor(candidate.CandidateID)
	}
	name := candidate.CandidateID
	if name == "" {
		name = fmt.Sprintf("candidate_%03d", candidate.Index)
	}
	return storage.SafeJoin(tempDir, SanitizeFileName(name)+".partial")
}

func materializeBlob(blobPath, finalPath string) error {
	if existingSHA, err := SHA256File(finalPath); err == nil {
		blobSHA, err := SHA256File(blobPath)
		if err != nil {
			return err
		}
		if existingSHA == blobSHA {
			return nil
		}
	}
	if err := os.MkdirAll(filepath.Dir(finalPath), 0755); err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(finalPath), "."+filepath.Base(finalPath)+".*.tmp")
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
	src, err := os.Open(blobPath)
	if err != nil {
		temp.Close()
		return err
	}
	_, copyErr := io.Copy(temp, src)
	closeSrcErr := src.Close()
	closeTempErr := temp.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeSrcErr != nil {
		return closeSrcErr
	}
	if closeTempErr != nil {
		return closeTempErr
	}
	if err := os.Remove(finalPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := os.Rename(tempName, finalPath); err != nil {
		return err
	}
	removeTemp = false
	return nil
}

type HTTPStatusError struct {
	StatusCode int
	Retryable  bool
	RetryAfter time.Duration
}

func (e HTTPStatusError) Error() string {
	return fmt.Sprintf("download failed with status %d", e.StatusCode)
}

type ContentLengthError struct {
	Expected int64
	Actual   int64
}

func (e ContentLengthError) Error() string {
	return fmt.Sprintf("content length mismatch: expected %d bytes, got %d", e.Expected, e.Actual)
}

type NetworkError struct {
	Err error
}

func (e NetworkError) Error() string {
	if e.Err == nil {
		return "network error"
	}
	return e.Err.Error()
}

func (e NetworkError) Unwrap() error {
	return e.Err
}

func IsRetryableError(err error) bool {
	var statusErr HTTPStatusError
	if errors.As(err, &statusErr) {
		return statusErr.Retryable
	}
	var lengthErr ContentLengthError
	if errors.As(err, &lengthErr) {
		return true
	}
	var networkErr NetworkError
	if errors.As(err, &networkErr) {
		if networkErr.Err == nil {
			return true
		}
		lower := strings.ToLower(networkErr.Err.Error())
		return strings.Contains(lower, "timeout") ||
			strings.Contains(lower, "temporary") ||
			strings.Contains(lower, "connection reset") ||
			strings.Contains(lower, "eof")
	}
	return false
}

func RetryAfter(err error) time.Duration {
	var statusErr HTTPStatusError
	if errors.As(err, &statusErr) {
		return statusErr.RetryAfter
	}
	return 0
}

func isRetryableHTTPStatus(statusCode int) bool {
	switch statusCode {
	case http.StatusRequestTimeout, http.StatusTooManyRequests, http.StatusInternalServerError,
		http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

func parseRetryAfter(value string) time.Duration {
	if value == "" {
		return 0
	}
	var seconds int
	if _, err := fmt.Sscanf(strings.TrimSpace(value), "%d", &seconds); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	if when, err := http.ParseTime(value); err == nil {
		if delay := time.Until(when); delay > 0 {
			return delay
		}
	}
	return 0
}

func backoffDelay(attempt int, retryAfter time.Duration) time.Duration {
	if retryAfter > 0 {
		if retryAfter > 30*time.Second {
			return 30 * time.Second
		}
		return retryAfter
	}
	delay := time.Duration(150*(1<<(attempt-1))) * time.Millisecond
	if delay > 2*time.Second {
		return 2 * time.Second
	}
	return delay
}

func sleep(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (d *Downloader) httpClient() *http.Client {
	if d != nil && d.HTTPClient != nil {
		return d.HTTPClient
	}
	return http.DefaultClient
}
