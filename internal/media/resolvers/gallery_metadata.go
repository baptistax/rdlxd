package resolvers

import (
	"context"
	"fmt"
	"html"

	"github.com/baptistax/rdlxd/internal/media"
	"github.com/baptistax/rdlxd/internal/post"
)

type RedditGalleryResolver struct{}

type GalleryMetadataResolver = RedditGalleryResolver

func (RedditGalleryResolver) Name() string {
	return "reddit_gallery"
}

func (r RedditGalleryResolver) Resolve(ctx context.Context, pc media.PostContext) ([]media.MediaCandidate, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	p := pc.Post
	if !p.IsGallery || p.GalleryData == nil || len(p.GalleryData.Items) == 0 {
		return nil, nil
	}
	candidates := make([]media.MediaCandidate, 0, len(p.GalleryData.Items))
	for index, item := range p.GalleryData.Items {
		metadata, ok := p.MediaMetadata[item.MediaID]
		if !ok {
			candidates = append(candidates, missingGalleryCandidate(p, r.Name(), item.MediaID, index+1))
			continue
		}
		best, alternatives := galleryMetadataURL(metadata, item.MediaID)
		if best.URL == "" {
			candidates = append(candidates, missingGalleryCandidate(p, r.Name(), item.MediaID, index+1))
			continue
		}
		rawURL := html.UnescapeString(best.URL)
		kind := best.MediaKind
		if kind == media.MediaKindUnknown {
			kind = kindFromMime(metadata.Mime)
		}
		if kind == media.MediaKindUnknown {
			kind = kindFromURL(rawURL)
		}
		expectedContentType := best.ExpectedContentType
		if expectedContentType == "" {
			expectedContentType = metadata.Mime
		}
		expectedExtension := extensionFromMime(expectedContentType)
		if expectedExtension == "" {
			expectedExtension = extensionFromURL(rawURL)
		}
		notes := append([]string{}, best.Notes...)
		unescapedAlternatives := make([]media.MediaAlternative, 0, len(alternatives))
		for _, alternative := range alternatives {
			alternative.URL = html.UnescapeString(alternative.URL)
			unescapedAlternatives = append(unescapedAlternatives, alternative)
		}
		candidates = append(candidates, media.MediaCandidate{
			PostID:              postID(p),
			URL:                 rawURL,
			ResolverName:        r.Name(),
			MediaKind:           kind,
			ContentRole:         "gallery_item",
			Quality:             best.Quality,
			Required:            true,
			Index:               index + 1,
			Width:               best.Width,
			Height:              best.Height,
			ExpectedContentType: expectedContentType,
			ExpectedExtension:   expectedExtension,
			SourceField:         best.SourceField,
			RequiresProbe:       kind == media.MediaKindUnknown,
			Notes:               notes,
			Alternatives:        unescapedAlternatives,
		})
	}
	return candidates, nil
}

type galleryURLCandidate struct {
	URL                 string
	SourceField         string
	Quality             string
	Width               int
	Height              int
	Rank                int
	MediaKind           media.MediaKind
	ExpectedContentType string
	Notes               []string
}

func galleryMetadataURL(metadata post.MediaMetadata, mediaID string) (galleryURLCandidate, []media.MediaAlternative) {
	var candidates []galleryURLCandidate
	candidates = append(candidates, gallerySourceCandidates(metadata.Source, "media_metadata."+mediaID+".s", "source", 3)...)
	for i, original := range metadata.Originals {
		candidates = append(candidates, gallerySourceCandidates(original, fmt.Sprintf("media_metadata.%s.o.%d", mediaID, i), "source", 3)...)
	}
	for i, image := range metadata.Images {
		next := gallerySourceCandidates(image, fmt.Sprintf("media_metadata.%s.p.%d", mediaID, i), "preview", 1)
		for j := range next {
			next[j].Notes = append(next[j].Notes, string(media.SubstatusPreviewOnly))
		}
		candidates = append(candidates, next...)
	}
	alternatives := make([]media.MediaAlternative, 0, len(candidates))
	var best galleryURLCandidate
	for _, candidate := range candidates {
		alternatives = append(alternatives, media.MediaAlternative{
			URL:         candidate.URL,
			SourceField: candidate.SourceField,
			Quality:     candidate.Quality,
			Width:       candidate.Width,
			Height:      candidate.Height,
		})
		if betterGalleryCandidate(candidate, best) {
			best = candidate
		}
	}
	return best, alternatives
}

func gallerySourceCandidates(source post.MediaSource, baseField, quality string, rank int) []galleryURLCandidate {
	var candidates []galleryURLCandidate
	if source.URL != "" {
		candidates = append(candidates, galleryURLCandidate{
			URL:         source.URL,
			SourceField: baseField + ".u",
			Quality:     quality,
			Width:       source.Width,
			Height:      source.Height,
			Rank:        rank,
		})
	}
	if source.GIF != "" {
		candidates = append(candidates, galleryURLCandidate{
			URL:                 source.GIF,
			SourceField:         baseField + ".gif",
			Quality:             quality,
			Width:               source.Width,
			Height:              source.Height,
			Rank:                rank,
			MediaKind:           media.MediaKindGIF,
			ExpectedContentType: "image/gif",
		})
	}
	if source.MP4 != "" {
		candidates = append(candidates, galleryURLCandidate{
			URL:                 source.MP4,
			SourceField:         baseField + ".mp4",
			Quality:             quality,
			Width:               source.Width,
			Height:              source.Height,
			Rank:                rank,
			MediaKind:           media.MediaKindGIF,
			ExpectedContentType: "video/mp4",
		})
	}
	return candidates
}

func betterGalleryCandidate(candidate, best galleryURLCandidate) bool {
	if candidate.URL == "" {
		return false
	}
	if best.URL == "" {
		return true
	}
	if candidate.Rank != best.Rank {
		return candidate.Rank > best.Rank
	}
	candidateArea := candidate.Width * candidate.Height
	bestArea := best.Width * best.Height
	if candidateArea != bestArea {
		return candidateArea > bestArea
	}
	if candidate.MediaKind != media.MediaKindUnknown && best.MediaKind == media.MediaKindUnknown {
		return true
	}
	return false
}

func missingGalleryCandidate(p post.Post, resolverName, mediaID string, index int) media.MediaCandidate {
	return media.MediaCandidate{
		PostID:        postID(p),
		ResolverName:  resolverName,
		MediaKind:     media.MediaKindUnknown,
		ContentRole:   "gallery_item",
		Required:      true,
		Index:         index,
		SourceField:   "media_metadata." + mediaID,
		RequiresProbe: false,
		Notes:         []string{"gallery_metadata_missing"},
	}
}
