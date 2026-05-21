package storage

import (
	"path/filepath"
	"testing"
)

func TestSQLiteStateInitializesAndAggregatesStatuses(t *testing.T) {
	state, err := OpenSQLiteState(filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatalf("OpenSQLiteState returned error: %v", err)
	}
	defer state.Close()

	records := []PostRecord{
		{PostID: "t3_downloaded", Status: "downloaded", Substatus: "text_post_saved"},
		{PostID: "t3_partial", Status: "partial", Reason: "some candidates failed", Retryable: true},
		{PostID: "t3_failed", Status: "failed", Reason: "all candidates failed", Retryable: true},
		{PostID: "t3_unsupported", Status: "unsupported", Reason: "external page"},
		{PostID: "t3_skipped", Status: "skipped", Reason: "legacy skipped post"},
	}
	for _, record := range records {
		if err := state.UpsertPost(record); err != nil {
			t.Fatalf("UpsertPost returned error: %v", err)
		}
	}

	summary, err := state.GetSummaryCounts()
	if err != nil {
		t.Fatalf("GetSummaryCounts returned error: %v", err)
	}
	if summary.Downloaded != 1 || summary.Partial != 1 || summary.Failed != 1 || summary.Unsupported != 2 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	if summary.Downloaded+summary.Partial+summary.Failed+summary.Unsupported != summary.PostsFound {
		t.Fatalf("summary counters do not cover posts found: %+v", summary)
	}
	if summary.NotFullyDownloaded != 4 {
		t.Fatalf("not fully downloaded = %d, want 4", summary.NotFullyDownloaded)
	}
}

func TestFailedListEmptyWorks(t *testing.T) {
	state, err := OpenSQLiteState(filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatalf("OpenSQLiteState returned error: %v", err)
	}
	defer state.Close()

	rows, err := state.ListIncompletePosts()
	if err != nil {
		t.Fatalf("ListIncompletePosts returned error: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("rows = %d, want 0", len(rows))
	}
}
