package reddit

import (
	"context"
	"fmt"

	"github.com/baptistax/rdlxd/internal/post"
)

type PostFetcher interface {
	FetchPost(ctx context.Context, source Source) (*post.Post, error)
}

type ClientPostFetcher struct {
	Client *Client
}

func (f ClientPostFetcher) FetchPost(ctx context.Context, source Source) (*post.Post, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if f.Client == nil {
		return nil, fmt.Errorf("reddit client is required")
	}
	if source.Type != SourcePost {
		return nil, fmt.Errorf("source is not a post")
	}
	endpoints := []string{"/api/info?id=t3_" + source.PostID}
	if source.Endpoint != "" {
		endpoints = append(endpoints, source.Endpoint)
	}
	var lastErr error
	for _, endpoint := range endpoints {
		data, _, err := f.Client.GetRaw(ctx, endpoint)
		if err != nil {
			lastErr = err
			continue
		}
		page, err := ParseListing(data)
		if err != nil {
			lastErr = err
			continue
		}
		if len(page.Posts) > 0 {
			return &page.Posts[0], nil
		}
		lastErr = fmt.Errorf("post not found")
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("post not found")
}
