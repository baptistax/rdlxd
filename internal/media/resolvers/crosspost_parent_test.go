package resolvers

import (
	"context"
	"testing"

	"github.com/baptistax/rdlxd/internal/media"
	"github.com/baptistax/rdlxd/internal/post"
)

func TestCrosspostParentResolverUsesParentWhenCurrentHasNoCandidates(t *testing.T) {
	resolver := NewCrosspostParentResolver([]media.MediaResolver{DirectMediaURLResolver{}})
	current := post.Post{
		Name: "t3_child",
		CrosspostParentList: []post.Post{{
			Name: "t3_parent",
			URL:  "https://i.redd.it/example.jpg?token=kept",
		}},
	}
	candidates, err := resolver.ResolveWithCurrent(context.Background(), media.PostContext{Post: current}, nil)
	if err != nil {
		t.Fatalf("ResolveWithCurrent returned error: %v", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("candidates = %d, want 1", len(candidates))
	}
	if candidates[0].PostID != "t3_child" || candidates[0].ParentPostID != "t3_parent" {
		t.Fatalf("unexpected post ids: %+v", candidates[0])
	}
}

func TestCrosspostParentResolverDoesNotRunWhenCurrentHasCandidates(t *testing.T) {
	resolver := NewCrosspostParentResolver([]media.MediaResolver{DirectMediaURLResolver{}})
	candidates, err := resolver.ResolveWithCurrent(
		context.Background(),
		media.PostContext{Post: post.Post{Name: "t3_child", CrosspostParentList: []post.Post{{Name: "t3_parent", URL: "https://i.redd.it/example.jpg"}}}},
		[]media.MediaCandidate{{URL: "https://example.com/current.jpg"}},
	)
	if err != nil {
		t.Fatalf("ResolveWithCurrent returned error: %v", err)
	}
	if len(candidates) != 0 {
		t.Fatalf("candidates = %d, want 0", len(candidates))
	}
}
