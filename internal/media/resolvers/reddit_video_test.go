package resolvers

import (
	"context"
	"testing"

	"github.com/baptistax/rdlxd/internal/media"
	"github.com/baptistax/rdlxd/internal/post"
)

func TestRedditVideoResolverUsesSecureMediaFallbackURL(t *testing.T) {
	rawURL := "https://v.redd.it/abc/DASH_720.mp4?source=fallback"
	candidates, err := RedditVideoResolver{}.Resolve(context.Background(), media.PostContext{Post: post.Post{
		Name:        "t3_video",
		SecureMedia: &post.Media{RedditVideo: &post.RedditVideo{FallbackURL: rawURL}},
		Media:       &post.Media{RedditVideo: &post.RedditVideo{FallbackURL: "https://v.redd.it/other.mp4"}},
	}})
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("candidates = %d, want 1", len(candidates))
	}
	if candidates[0].URL != rawURL {
		t.Fatalf("url = %s, want %s", candidates[0].URL, rawURL)
	}
	if len(candidates[0].Notes) == 0 || candidates[0].Notes[0] != string(media.SubstatusVideoMayBeSilent) {
		t.Fatalf("notes = %v", candidates[0].Notes)
	}
}
