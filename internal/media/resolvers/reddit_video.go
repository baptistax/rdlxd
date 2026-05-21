package resolvers

import (
	"context"
	"html"

	"github.com/baptistax/rdlxd/internal/media"
	"github.com/baptistax/rdlxd/internal/post"
)

type RedditVideoResolver struct{}

func (RedditVideoResolver) Name() string {
	return "reddit_video"
}

func (r RedditVideoResolver) Resolve(ctx context.Context, pc media.PostContext) ([]media.MediaCandidate, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	p := pc.Post
	video := redditVideoFromPost(p)
	if video == nil || video.FallbackURL == "" {
		return nil, nil
	}
	notes := []string{string(media.SubstatusVideoMayBeSilent)}
	rawURL := html.UnescapeString(video.FallbackURL)
	return []media.MediaCandidate{{
		PostID:              postID(p),
		URL:                 rawURL,
		ResolverName:        r.Name(),
		MediaKind:           media.MediaKindVideo,
		ContentRole:         "reddit_video",
		Quality:             "fallback",
		Required:            true,
		Index:               1,
		ExpectedContentType: "video/mp4",
		ExpectedExtension:   ".mp4",
		SourceField:         "secure_media.reddit_video.fallback_url",
		RequiresProbe:       true,
		Notes:               notes,
	}}, nil
}

func redditVideoFromPost(p post.Post) *post.RedditVideo {
	if p.SecureMedia != nil && p.SecureMedia.RedditVideo != nil {
		return p.SecureMedia.RedditVideo
	}
	if p.Media != nil && p.Media.RedditVideo != nil {
		return p.Media.RedditVideo
	}
	return nil
}
