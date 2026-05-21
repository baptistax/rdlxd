package resolvers

import (
	"context"
	"html"

	"github.com/baptistax/rdlxd/internal/media"
	"github.com/baptistax/rdlxd/internal/post"
)

type UnsupportedPostResolver struct{}

func (UnsupportedPostResolver) Name() string {
	return "unsupported_post"
}

func (r UnsupportedPostResolver) Resolve(ctx context.Context, pc media.PostContext) ([]media.MediaCandidate, error) {
	return r.ResolveWithCurrent(ctx, pc, nil)
}

func (r UnsupportedPostResolver) ResolveWithCurrent(ctx context.Context, pc media.PostContext, current []media.MediaCandidate) ([]media.MediaCandidate, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(current) > 0 {
		return nil, nil
	}
	p := pc.Post
	rawURL := html.UnescapeString(mediaURL(p))
	sourceField := mediaURLSourceField(p)
	reason := "post has no supported media"
	substatus := string(media.SubstatusNoMediaCandidate)
	if rawURL != "" {
		reason = "external link is not supported"
		substatus = string(media.SubstatusExternalPageUnsupported)
	}
	if p.IsSelf {
		sourceField = "selftext"
	} else if !post.HasExpectedMedia(p) {
		sourceField = "post"
	}
	return []media.MediaCandidate{{
		PostID:               postID(p),
		URL:                  rawURL,
		ResolverName:         r.Name(),
		MediaKind:            media.MediaKindUnknown,
		ContentRole:          "unsupported",
		Required:             true,
		Index:                1,
		SourceField:          sourceField,
		Unsupported:          true,
		UnsupportedSubstatus: substatus,
		UnsupportedReason:    reason,
	}}, nil
}
