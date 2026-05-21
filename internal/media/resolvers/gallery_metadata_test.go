package resolvers

import (
	"context"
	"testing"

	"github.com/baptistax/rdlxd/internal/media"
	"github.com/baptistax/rdlxd/internal/post"
)

func TestGalleryMetadataResolverPreservesGalleryOrder(t *testing.T) {
	candidates, err := GalleryMetadataResolver{}.Resolve(context.Background(), media.PostContext{Post: post.Post{
		Name:      "t3_gallery",
		IsGallery: true,
		GalleryData: &post.GalleryData{Items: []post.GalleryItem{
			{MediaID: "second"},
			{MediaID: "first"},
		}},
		MediaMetadata: map[string]post.MediaMetadata{
			"first":  {Mime: "image/jpeg", Source: post.MediaSource{URL: "https://i.redd.it/first.jpg"}},
			"second": {Mime: "image/png", Source: post.MediaSource{URL: "https://i.redd.it/second.png"}},
		},
	}})
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if len(candidates) != 2 {
		t.Fatalf("candidates = %d, want 2", len(candidates))
	}
	if candidates[0].URL != "https://i.redd.it/second.png" || candidates[0].Index != 1 {
		t.Fatalf("first candidate = %+v", candidates[0])
	}
	if candidates[1].URL != "https://i.redd.it/first.jpg" || candidates[1].Index != 2 {
		t.Fatalf("second candidate = %+v", candidates[1])
	}
}

func TestGalleryMetadataResolverKeepsMissingItemAsRequiredCandidate(t *testing.T) {
	candidates, err := GalleryMetadataResolver{}.Resolve(context.Background(), media.PostContext{Post: post.Post{
		Name:      "t3_gallery",
		IsGallery: true,
		GalleryData: &post.GalleryData{Items: []post.GalleryItem{
			{MediaID: "present"},
			{MediaID: "missing"},
		}},
		MediaMetadata: map[string]post.MediaMetadata{
			"present": {Mime: "image/jpeg", Source: post.MediaSource{URL: "https://i.redd.it/present.jpg"}},
		},
	}})
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if len(candidates) != 2 {
		t.Fatalf("candidates = %d, want 2", len(candidates))
	}
	if candidates[1].URL != "" || !candidates[1].Required || candidates[1].Index != 2 {
		t.Fatalf("missing candidate = %+v", candidates[1])
	}
}

func TestGalleryMetadataResolverPrefersSourceOverSmallPreview(t *testing.T) {
	candidates, err := GalleryMetadataResolver{}.Resolve(context.Background(), media.PostContext{Post: post.Post{
		Name:      "t3_gallery",
		IsGallery: true,
		GalleryData: &post.GalleryData{Items: []post.GalleryItem{
			{MediaID: "image"},
		}},
		MediaMetadata: map[string]post.MediaMetadata{
			"image": {
				Mime:   "image/jpeg",
				Source: post.MediaSource{URL: "https://preview.redd.it/full.jpg?width=4000&amp;auto=webp", Width: 4000, Height: 3000},
				Images: []post.MediaSource{
					{URL: "https://preview.redd.it/thumb.jpg?width=320&amp;auto=webp", Width: 320, Height: 240},
				},
			},
		},
	}})
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("candidates = %d, want 1", len(candidates))
	}
	if candidates[0].URL != "https://preview.redd.it/full.jpg?width=4000&auto=webp" {
		t.Fatalf("url = %s", candidates[0].URL)
	}
	if candidates[0].Width != 4000 || candidates[0].Height != 3000 {
		t.Fatalf("dimensions = %dx%d", candidates[0].Width, candidates[0].Height)
	}
	if len(candidates[0].Alternatives) != 2 {
		t.Fatalf("alternatives = %+v", candidates[0].Alternatives)
	}
}

func TestGalleryMetadataResolverUsesLargestOriginalCandidate(t *testing.T) {
	candidates, err := GalleryMetadataResolver{}.Resolve(context.Background(), media.PostContext{Post: post.Post{
		Name:      "t3_gallery",
		IsGallery: true,
		GalleryData: &post.GalleryData{Items: []post.GalleryItem{
			{MediaID: "image"},
		}},
		MediaMetadata: map[string]post.MediaMetadata{
			"image": {
				Mime:   "image/jpeg",
				Source: post.MediaSource{URL: "https://preview.redd.it/source.jpg", Width: 1200, Height: 800},
				Originals: []post.MediaSource{
					{URL: "https://preview.redd.it/original.jpg", Width: 4000, Height: 3000},
				},
				Images: []post.MediaSource{
					{URL: "https://preview.redd.it/thumb.jpg", Width: 320, Height: 240},
				},
			},
		},
	}})
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("candidates = %d, want 1", len(candidates))
	}
	if candidates[0].URL != "https://preview.redd.it/original.jpg" {
		t.Fatalf("url = %s", candidates[0].URL)
	}
	if candidates[0].SourceField != "media_metadata.image.o.0.u" {
		t.Fatalf("source field = %s", candidates[0].SourceField)
	}
}
