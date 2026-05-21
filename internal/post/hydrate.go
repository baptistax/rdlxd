package post

import "context"

func Hydrate(ctx context.Context, posts []Post) ([]Post, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	normalized := make([]Post, 0, len(posts))
	for _, p := range posts {
		normalized = append(normalized, Normalize(p))
	}
	return normalized, nil
}
