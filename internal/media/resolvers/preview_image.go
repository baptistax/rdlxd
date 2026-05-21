package resolvers

import (
	"context"
	"fmt"
	"html"

	"github.com/baptistax/rdlxd/internal/media"
	"github.com/baptistax/rdlxd/internal/post"
)

type RedditPreviewResolver struct{}

type PreviewImageResolver = RedditPreviewResolver

func (RedditPreviewResolver) Name() string {
	return "reddit_preview"
}

func (r RedditPreviewResolver) Resolve(ctx context.Context, pc media.PostContext) ([]media.MediaCandidate, error) {
	return r.ResolveWithCurrent(ctx, pc, nil)
}

func (r RedditPreviewResolver) ResolveWithCurrent(ctx context.Context, pc media.PostContext, current []media.MediaCandidate) ([]media.MediaCandidate, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if hasDownloadableCandidate(current) {
		return nil, nil
	}
	p := pc.Post
	if p.Preview == nil || len(p.Preview.Images) == 0 {
		return nil, nil
	}
	image := p.Preview.Images[0]
	best, alternatives := previewImageURL(image)
	if best.URL == "" {
		return nil, nil
	}
	rawURL := html.UnescapeString(best.URL)
	if !shouldUsePreviewFallback(p, rawURL) {
		return nil, nil
	}
	unescapedAlternatives := make([]media.MediaAlternative, 0, len(alternatives))
	for _, alternative := range alternatives {
		alternative.URL = html.UnescapeString(alternative.URL)
		unescapedAlternatives = append(unescapedAlternatives, alternative)
	}
	return []media.MediaCandidate{{
		PostID:            postID(p),
		URL:               rawURL,
		ResolverName:      r.Name(),
		MediaKind:         media.MediaKindImage,
		ContentRole:       "preview",
		Quality:           "preview",
		Required:          true,
		Index:             1,
		Width:             best.Width,
		Height:            best.Height,
		ExpectedExtension: extensionFromURL(rawURL),
		SourceField:       best.SourceField,
		RequiresProbe:     true,
		Notes:             []string{string(media.SubstatusPreviewOnly)},
		Alternatives:      unescapedAlternatives,
	}}, nil
}

type previewURLCandidate struct {
	URL         string
	SourceField string
	Width       int
	Height      int
}

func previewImageURL(image post.PreviewImage) (previewURLCandidate, []media.MediaAlternative) {
	var candidates []previewURLCandidate
	if image.Source.URL != "" {
		candidates = append(candidates, previewURLCandidate{
			URL:         image.Source.URL,
			SourceField: "preview.images.0.source.url",
			Width:       image.Source.Width,
			Height:      image.Source.Height,
		})
	}
	for i, resolution := range image.Resolutions {
		if resolution.URL == "" {
			continue
		}
		candidates = append(candidates, previewURLCandidate{
			URL:         resolution.URL,
			SourceField: fmt.Sprintf("preview.images.0.resolutions.%d.url", i),
			Width:       resolution.Width,
			Height:      resolution.Height,
		})
	}
	alternatives := make([]media.MediaAlternative, 0, len(candidates))
	var best previewURLCandidate
	for _, candidate := range candidates {
		alternatives = append(alternatives, media.MediaAlternative{
			URL:         candidate.URL,
			SourceField: candidate.SourceField,
			Quality:     "preview",
			Width:       candidate.Width,
			Height:      candidate.Height,
		})
		if betterPreviewCandidate(candidate, best) {
			best = candidate
		}
	}
	return best, alternatives
}

func betterPreviewCandidate(candidate, best previewURLCandidate) bool {
	if candidate.URL == "" {
		return false
	}
	if best.URL == "" {
		return true
	}
	candidateArea := candidate.Width * candidate.Height
	bestArea := best.Width * best.Height
	return candidateArea > bestArea
}

func hasDownloadableCandidate(candidates []media.MediaCandidate) bool {
	for _, candidate := range candidates {
		if candidate.Unsupported {
			continue
		}
		if candidate.URL != "" || candidate.MediaKind == media.MediaKindText {
			return true
		}
	}
	return false
}

func shouldUsePreviewFallback(p post.Post, rawURL string) bool {
	if rawURL == "" || !isHTTPURL(rawURL) || isExternalPreviewHost(rawURL) {
		return false
	}
	destination := html.UnescapeString(mediaURL(p))
	if destination == "" {
		return true
	}
	if isRedditPageURL(destination) {
		return false
	}
	if kindFromURL(destination) != media.MediaKindUnknown {
		return true
	}
	if isRedditMediaHost(destination) {
		return true
	}
	return p.PostHint == "image"
}
