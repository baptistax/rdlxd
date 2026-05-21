package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type SQLiteState struct {
	db *sql.DB
}

func OpenSQLiteState(path string) (*SQLiteState, error) {
	if path == "" {
		return nil, fmt.Errorf("state path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	state := &SQLiteState{db: db}
	if err := state.initialize(); err != nil {
		db.Close()
		return nil, err
	}
	return state, nil
}

func (s *SQLiteState) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *SQLiteState) initialize() error {
	statements := []string{
		`PRAGMA foreign_keys = ON`,
		`PRAGMA journal_mode = WAL`,
		`CREATE TABLE IF NOT EXISTS posts (
			post_id TEXT PRIMARY KEY,
			name TEXT NOT NULL DEFAULT '',
			permalink TEXT NOT NULL DEFAULT '',
			title TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT '',
			substatus TEXT NOT NULL DEFAULT '',
			retryable INTEGER NOT NULL DEFAULT 0,
			reason TEXT NOT NULL DEFAULT '',
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS assets (
			asset_id TEXT PRIMARY KEY,
			post_id TEXT NOT NULL,
			candidate_id TEXT NOT NULL DEFAULT '',
			url TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT '',
			substatus TEXT NOT NULL DEFAULT '',
			blob_id TEXT NOT NULL DEFAULT '',
			path TEXT NOT NULL DEFAULT '',
			retryable INTEGER NOT NULL DEFAULT 0,
			reason TEXT NOT NULL DEFAULT '',
			updated_at TEXT NOT NULL,
			FOREIGN KEY(post_id) REFERENCES posts(post_id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS blobs (
			blob_id TEXT PRIMARY KEY,
			sha256 TEXT NOT NULL UNIQUE,
			size INTEGER NOT NULL,
			path TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS attempts (
			attempt_id TEXT PRIMARY KEY,
			post_id TEXT NOT NULL DEFAULT '',
			asset_id TEXT NOT NULL DEFAULT '',
			run_id TEXT NOT NULL DEFAULT '',
			started_at TEXT NOT NULL,
			finished_at TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT '',
			error_code TEXT NOT NULL DEFAULT '',
			retryable INTEGER NOT NULL DEFAULT 0,
			message TEXT NOT NULL DEFAULT '',
			etag TEXT NOT NULL DEFAULT '',
			last_modified TEXT NOT NULL DEFAULT '',
			content_length INTEGER NOT NULL DEFAULT 0,
			bytes_downloaded INTEGER NOT NULL DEFAULT 0,
			partial_path TEXT NOT NULL DEFAULT '',
			resume_supported INTEGER NOT NULL DEFAULT 0,
			last_http_status INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS runs (
			run_id TEXT PRIMARY KEY,
			source TEXT NOT NULL DEFAULT '',
			started_at TEXT NOT NULL,
			finished_at TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT ''
		)`,
	}
	for _, statement := range statements {
		if _, err := s.db.Exec(statement); err != nil {
			return err
		}
	}
	return nil
}

func (s *SQLiteState) CreateRun(runID, source string) error {
	if runID == "" {
		return fmt.Errorf("run id is required")
	}
	_, err := s.db.Exec(
		`INSERT INTO runs (run_id, source, started_at, status)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(run_id) DO UPDATE SET source = excluded.source, status = excluded.status`,
		runID, source, nowText(), "running",
	)
	return err
}

func (s *SQLiteState) FinishRun(runID, status string) error {
	if runID == "" {
		return fmt.Errorf("run id is required")
	}
	_, err := s.db.Exec(
		`UPDATE runs SET finished_at = ?, status = ? WHERE run_id = ?`,
		nowText(), status, runID,
	)
	return err
}

func (s *SQLiteState) UpsertPost(record PostRecord) error {
	if record.PostID == "" {
		return fmt.Errorf("post id is required")
	}
	_, err := s.db.Exec(
		`INSERT INTO posts (post_id, name, permalink, title, status, substatus, retryable, reason, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(post_id) DO UPDATE SET
			name = excluded.name,
			permalink = excluded.permalink,
			title = excluded.title,
			status = excluded.status,
			substatus = excluded.substatus,
			retryable = excluded.retryable,
			reason = excluded.reason,
			updated_at = excluded.updated_at`,
		record.PostID, record.Name, record.Permalink, record.Title, record.Status, record.Substatus,
		boolInt(record.Retryable), record.Reason, nowText(),
	)
	return err
}

func (s *SQLiteState) UpsertAsset(record AssetRecord) error {
	if record.AssetID == "" {
		return fmt.Errorf("asset id is required")
	}
	if record.PostID == "" {
		return fmt.Errorf("post id is required")
	}
	_, err := s.db.Exec(
		`INSERT INTO assets (asset_id, post_id, candidate_id, url, status, substatus, blob_id, path, retryable, reason, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(asset_id) DO UPDATE SET
			post_id = excluded.post_id,
			candidate_id = excluded.candidate_id,
			url = excluded.url,
			status = excluded.status,
			substatus = excluded.substatus,
			blob_id = excluded.blob_id,
			path = excluded.path,
			retryable = excluded.retryable,
			reason = excluded.reason,
			updated_at = excluded.updated_at`,
		record.AssetID, record.PostID, record.CandidateID, record.URL, record.Status, record.Substatus,
		record.BlobID, record.Path, boolInt(record.Retryable), record.Reason, nowText(),
	)
	return err
}

func (s *SQLiteState) UpsertBlob(record BlobRecord) error {
	if record.BlobID == "" {
		return fmt.Errorf("blob id is required")
	}
	if record.SHA256 == "" {
		return fmt.Errorf("sha256 is required")
	}
	_, err := s.db.Exec(
		`INSERT INTO blobs (blob_id, sha256, size, path, created_at)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(blob_id) DO UPDATE SET sha256 = excluded.sha256, size = excluded.size, path = excluded.path`,
		record.BlobID, record.SHA256, record.Size, record.Path, nowText(),
	)
	return err
}

func (s *SQLiteState) RecordAttempt(record AttemptRecord) error {
	if record.AttemptID == "" {
		return fmt.Errorf("attempt id is required")
	}
	_, err := s.db.Exec(
		`INSERT INTO attempts (
			attempt_id, post_id, asset_id, run_id, started_at, finished_at, status, error_code, retryable,
			message, etag, last_modified, content_length, bytes_downloaded, partial_path, resume_supported, last_http_status
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		record.AttemptID, record.PostID, record.AssetID, record.RunID, nowText(), nowText(), record.Status,
		record.ErrorCode, boolInt(record.Retryable), record.Message, record.ETag, record.LastModified,
		record.ContentLength, record.BytesDownloaded, record.PartialPath, boolInt(record.ResumeSupported), record.LastHTTPStatus,
	)
	return err
}

func (s *SQLiteState) ListIncompletePosts() ([]FailedPost, error) {
	rows, err := s.db.Query(
		`SELECT post_id, permalink, status, substatus,
			CASE WHEN reason != '' THEN reason ELSE substatus END AS reason,
			retryable
		 FROM posts
		 WHERE status != 'downloaded'
		 ORDER BY updated_at ASC, post_id ASC`,
	)
	if err != nil {
		return nil, err
	}
	var failed []FailedPost
	for rows.Next() {
		var row FailedPost
		var retryable int
		if err := rows.Scan(&row.PostID, &row.Permalink, &row.Status, &row.Substatus, &row.Reason, &retryable); err != nil {
			return nil, err
		}
		row.Status = incompleteListStatus(row.Status)
		if row.Substatus == "" && row.Status == "unsupported" {
			row.Substatus = "no_media_candidate"
		}
		if row.Reason == "" {
			row.Reason = row.Substatus
		}
		row.Retryable = retryable != 0
		if row.Permalink == "" {
			row.Permalink = row.PostID
		}
		row.LastError = row.Reason
		failed = append(failed, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	for i := range failed {
		candidates, err := s.listFailedCandidates(failed[i].PostID)
		if err != nil {
			return nil, err
		}
		failed[i].CandidatesAttempted = candidates
	}
	return failed, nil
}

func (s *SQLiteState) listFailedCandidates(postID string) ([]FailedCandidate, error) {
	rows, err := s.db.Query(
		`SELECT asset_id, candidate_id, url, status, substatus, reason, retryable
		 FROM assets
		 WHERE post_id = ?
		 ORDER BY asset_id ASC`,
		postID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var candidates []FailedCandidate
	for rows.Next() {
		var candidate FailedCandidate
		var retryable int
		if err := rows.Scan(
			&candidate.AssetID,
			&candidate.CandidateID,
			&candidate.URL,
			&candidate.Status,
			&candidate.Substatus,
			&candidate.Reason,
			&retryable,
		); err != nil {
			return nil, err
		}
		candidate.Retryable = retryable != 0
		candidates = append(candidates, candidate)
	}
	return candidates, rows.Err()
}

func (s *SQLiteState) GetSummaryCounts() (SummaryCounts, error) {
	var summary SummaryCounts
	rows, err := s.db.Query(`SELECT status, COUNT(*) FROM posts GROUP BY status`)
	if err != nil {
		return summary, err
	}
	defer rows.Close()

	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return summary, err
		}
		summary.PostsFound += count
		switch summaryStatus(status) {
		case "downloaded":
			summary.Downloaded += count
		case "partial":
			summary.Partial += count
			summary.NotFullyDownloaded += count
		case "failed":
			summary.Failed += count
			summary.NotFullyDownloaded += count
		case "unsupported":
			summary.Unsupported += count
			summary.NotFullyDownloaded += count
		}
	}
	return summary, rows.Err()
}

func summaryStatus(status string) string {
	switch status {
	case "downloaded", "partial", "failed", "unsupported":
		return status
	case "skipped":
		return "unsupported"
	default:
		return "unsupported"
	}
}

func incompleteListStatus(status string) string {
	switch status {
	case "partial", "failed", "unsupported":
		return status
	case "downloaded":
		return status
	default:
		return "unsupported"
	}
}

func (s *SQLiteState) MarkPostStatus(postID, status, substatus, reason string, retryable bool) error {
	if postID == "" {
		return fmt.Errorf("post id is required")
	}
	_, err := s.db.Exec(
		`UPDATE posts SET status = ?, substatus = ?, reason = ?, retryable = ?, updated_at = ? WHERE post_id = ?`,
		status, substatus, reason, boolInt(retryable), nowText(), postID,
	)
	return err
}

func (s *SQLiteState) MarkAssetStatus(assetID, status, substatus, reason string, retryable bool) error {
	if assetID == "" {
		return fmt.Errorf("asset id is required")
	}
	_, err := s.db.Exec(
		`UPDATE assets SET status = ?, substatus = ?, reason = ?, retryable = ?, updated_at = ? WHERE asset_id = ?`,
		status, substatus, reason, boolInt(retryable), nowText(), assetID,
	)
	return err
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func nowText() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}
