package media

import (
	"context"
	"testing"
)

type fakeResolver struct {
	name       string
	calls      *[]string
	candidates []MediaCandidate
}

func (r fakeResolver) Name() string {
	return r.name
}

func (r fakeResolver) Resolve(ctx context.Context, post PostContext) ([]MediaCandidate, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	*r.calls = append(*r.calls, r.name)
	return r.candidates, nil
}

func TestMediaPipelineCallsResolversInOrder(t *testing.T) {
	var calls []string
	pipeline := NewPipeline(
		fakeResolver{name: "first", calls: &calls},
		fakeResolver{name: "second", calls: &calls},
	)
	if _, err := pipeline.Resolve(context.Background(), PostContext{}); err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if len(calls) != 2 || calls[0] != "first" || calls[1] != "second" {
		t.Fatalf("calls = %v", calls)
	}
}

func TestMediaPipelineDeduplicatesConservatively(t *testing.T) {
	var calls []string
	candidate := MediaCandidate{URL: "https://example.com/a.jpg?token=1", MediaKind: MediaKindImage, ContentRole: "primary", Index: 1}
	pipeline := NewPipeline(
		fakeResolver{name: "first", calls: &calls, candidates: []MediaCandidate{candidate}},
		fakeResolver{name: "second", calls: &calls, candidates: []MediaCandidate{candidate}},
	)
	result, err := pipeline.Resolve(context.Background(), PostContext{})
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if len(result.Candidates) != 1 {
		t.Fatalf("candidates = %d, want 1", len(result.Candidates))
	}
	if result.Candidates[0].URL != candidate.URL {
		t.Fatalf("url = %s, want %s", result.Candidates[0].URL, candidate.URL)
	}
}
