package resolvers

import (
	"context"

	"github.com/baptistax/rdlxd/internal/media"
)

type CrosspostParentResolver struct {
	parentResolvers []media.MediaResolver
}

func NewCrosspostParentResolver(parentResolvers []media.MediaResolver) CrosspostParentResolver {
	return CrosspostParentResolver{parentResolvers: append([]media.MediaResolver{}, parentResolvers...)}
}

func (CrosspostParentResolver) Name() string {
	return "crosspost_parent"
}

func (r CrosspostParentResolver) Resolve(ctx context.Context, pc media.PostContext) ([]media.MediaCandidate, error) {
	return nil, ctx.Err()
}

func (r CrosspostParentResolver) ResolveWithCurrent(ctx context.Context, pc media.PostContext, current []media.MediaCandidate) ([]media.MediaCandidate, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(current) > 0 || pc.Depth >= 1 || len(pc.Post.CrosspostParentList) == 0 {
		return nil, nil
	}
	parent := pc.Post.CrosspostParentList[0]
	parentPipeline := media.NewPipeline(r.parentResolvers...)
	result, err := parentPipeline.Resolve(ctx, media.PostContext{Post: parent, Depth: pc.Depth + 1})
	if err != nil {
		return nil, err
	}
	parentID := postID(parent)
	for i := range result.Candidates {
		result.Candidates[i].ParentPostID = parentID
		result.Candidates[i].PostID = postID(pc.Post)
		result.Candidates[i].ResolverName = r.Name()
		result.Candidates[i].Notes = append(result.Candidates[i].Notes, "from crosspost parent")
	}
	return result.Candidates, nil
}
