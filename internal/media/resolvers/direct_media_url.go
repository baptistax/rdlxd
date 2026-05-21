package resolvers

import (
	"context"
	"html"

	"github.com/baptistax/rdlxd/internal/media"
)

type DirectMediaURLResolver struct{}

func (DirectMediaURLResolver) Name() string {
	return "direct_media_url"
}

func (r DirectMediaURLResolver) Resolve(ctx context.Context, pc media.PostContext) ([]media.MediaCandidate, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	p := pc.Post
	if p.IsSelf {
		return nil, nil
	}
	rawURL := mediaURL(p)
	sourceField := mediaURLSourceField(p)
	if rawURL == "" || !isHTTPURL(rawURL) {
		return nil, nil
	}
	rawURL = html.UnescapeString(rawURL)
	kind := kindFromURL(rawURL)
	if kind == media.MediaKindUnknown && isRedditPageURL(rawURL) {
		return nil, nil
	}
	return []media.MediaCandidate{{
		PostID:            postID(p),
		URL:               rawURL,
		ResolverName:      r.Name(),
		MediaKind:         kind,
		ContentRole:       "external_or_direct",
		Quality:           "source",
		Required:          true,
		Index:             1,
		ExpectedExtension: extensionFromURL(rawURL),
		SourceField:       sourceField,
		RequiresProbe:     true,
	}}, nil
}
