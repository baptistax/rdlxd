package resolvers

import (
	"context"
	"os"
	"testing"

	"github.com/baptistax/rdlxd/internal/media"
	"github.com/baptistax/rdlxd/internal/post"
)

func TestDirectMediaURLResolverPreservesQueryStringAndHasNoSideEffects(t *testing.T) {
	tempDir := t.TempDir()
	before, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("ReadDir returned error: %v", err)
	}
	rawURL := "https://example.com/media/image.jpg?width=960&token=abc123"
	candidates, err := DirectMediaURLResolver{}.Resolve(context.Background(), media.PostContext{
		Post: post.Post{Name: "t3_abc", URL: rawURL},
	})
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	after, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("ReadDir returned error: %v", err)
	}
	if len(after) != len(before) {
		t.Fatal("resolver wrote to disk")
	}
	if len(candidates) != 1 {
		t.Fatalf("candidates = %d, want 1", len(candidates))
	}
	if candidates[0].URL != rawURL {
		t.Fatalf("url = %s, want %s", candidates[0].URL, rawURL)
	}
}
