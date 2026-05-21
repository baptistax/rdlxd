package resolvers

import (
	"context"
	"html"

	"github.com/baptistax/rdlxd/internal/media"
)

type RedditImageURLResolver struct{}

type PrimaryURLResolver = RedditImageURLResolver

func (RedditImageURLResolver) Name() string {
	return "reddit_image_url"
}

func (r RedditImageURLResolver) Resolve(ctx context.Context, pc media.PostContext) ([]media.MediaCandidate, error) {
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
	if kind == media.MediaKindUnknown {
		return nil, nil
	}
	return []media.MediaCandidate{{
		PostID:              postID(p),
		URL:                 rawURL,
		ResolverName:        r.Name(),
		MediaKind:           kind,
		ContentRole:         "primary",
		Quality:             "source",
		Required:            true,
		Index:               1,
		ExpectedExtension:   extensionFromURL(rawURL),
		SourceField:         sourceField,
		RequiresProbe:       kind == media.MediaKindUnknown,
		ExpectedContentType: "",
	}}, nil
}
