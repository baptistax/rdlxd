package media

import (
	"context"

	"github.com/baptistax/rdlxd/internal/post"
)

type PostContext struct {
	Post  post.Post
	Depth int
}

type MediaResolver interface {
	Name() string
	Resolve(ctx context.Context, post PostContext) ([]MediaCandidate, error)
}

type CurrentAwareResolver interface {
	MediaResolver
	ResolveWithCurrent(ctx context.Context, post PostContext, current []MediaCandidate) ([]MediaCandidate, error)
}
