package resolvers

import (
	"context"
	"testing"

	"github.com/baptistax/rdlxd/internal/media"
	"github.com/baptistax/rdlxd/internal/post"
)

func TestRedditPreviewResolverSkipsExternalPreviewForExternalPost(t *testing.T) {
	candidates, err := RedditPreviewResolver{}.Resolve(context.Background(), media.PostContext{Post: post.Post{
		Name:                "t3_external",
		URL:                 "https://redgifs.com/watch/example",
		URLOverriddenByDest: "https://redgifs.com/watch/example",
		Domain:              "redgifs.com",
		PostHint:            "rich:video",
		Preview: &post.Preview{Images: []post.PreviewImage{{
			Source: post.PreviewSource{
				URL:    "https://external-preview.redd.it/example.jpeg?auto=webp&amp;s=abc",
				Width:  480,
				Height: 864,
			},
		}}},
	}})
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if len(candidates) != 0 {
		t.Fatalf("candidates = %+v, want none", candidates)
	}
}

func TestRedditPreviewResolverUsesLargestPreviewAsPreviewOnlyFallback(t *testing.T) {
	candidates, err := RedditPreviewResolver{}.Resolve(context.Background(), media.PostContext{Post: post.Post{
		Name: "t3_preview",
		Preview: &post.Preview{Images: []post.PreviewImage{{
			Source: post.PreviewSource{
				URL:    "https://preview.redd.it/source.jpg?auto=webp&amp;s=abc",
				Width:  1280,
				Height: 720,
			},
			Resolutions: []post.PreviewSource{
				{URL: "https://preview.redd.it/small.jpg?width=320&amp;s=abc", Width: 320, Height: 180},
			},
		}}},
	}})
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("candidates = %d, want 1", len(candidates))
	}
	if candidates[0].URL != "https://preview.redd.it/source.jpg?auto=webp&s=abc" {
		t.Fatalf("url = %s", candidates[0].URL)
	}
	if !candidates[0].Required || candidates[0].Quality != "preview" {
		t.Fatalf("candidate = %+v", candidates[0])
	}
	if len(candidates[0].Notes) != 1 || candidates[0].Notes[0] != string(media.SubstatusPreviewOnly) {
		t.Fatalf("notes = %v", candidates[0].Notes)
	}
	if candidates[0].Width != 1280 || candidates[0].Height != 720 {
		t.Fatalf("dimensions = %dx%d", candidates[0].Width, candidates[0].Height)
	}
}

func TestRedditPreviewResolverSkipsWhenBetterCandidateExists(t *testing.T) {
	candidates, err := RedditPreviewResolver{}.ResolveWithCurrent(context.Background(), media.PostContext{Post: post.Post{
		Name: "t3_primary",
		Preview: &post.Preview{Images: []post.PreviewImage{{
			Source: post.PreviewSource{URL: "https://preview.redd.it/source.jpg", Width: 1280, Height: 720},
		}}},
	}}, []media.MediaCandidate{{
		URL:       "https://i.redd.it/source.jpg",
		MediaKind: media.MediaKindImage,
		Required:  true,
	}})
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if len(candidates) != 0 {
		t.Fatalf("candidates = %+v, want none", candidates)
	}
}
