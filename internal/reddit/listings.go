package reddit

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/baptistax/rdlxd/internal/post"
)

type ListingFetcher interface {
	FetchPosts(ctx context.Context, source Source) ([]post.Post, error)
}

type ClientListingFetcher struct {
	Client *Client
}

func (f ClientListingFetcher) FetchPosts(ctx context.Context, source Source) ([]post.Post, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if f.Client == nil {
		return nil, fmt.Errorf("reddit client is required")
	}
	if source.Type == SourcePost {
		one, err := (ClientPostFetcher{Client: f.Client}).FetchPost(ctx, source)
		if err != nil || one == nil {
			return nil, err
		}
		return []post.Post{*one}, nil
	}
	if source.Type != SourceSubreddit && source.Type != SourceUser {
		return nil, fmt.Errorf("unsupported reddit source type: %s", source.Type)
	}
	if source.Limit <= 0 {
		return []post.Post{}, nil
	}

	var posts []post.Post
	seen := map[string]struct{}{}
	after := ""
	count := 0
	for len(posts) < source.Limit {
		pageLimit := source.Limit - len(posts)
		if pageLimit > 100 {
			pageLimit = 100
		}
		endpoint := listingEndpoint(source.Endpoint, pageLimit, after, count)
		data, _, err := f.Client.GetRaw(ctx, endpoint)
		if err != nil {
			return posts, err
		}
		page, err := ParseListing(data)
		if err != nil {
			return posts, err
		}
		for _, p := range page.Posts {
			key := p.Name
			if key == "" {
				key = "t3_" + p.ID
			}
			if key == "" {
				continue
			}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			posts = append(posts, p)
			if len(posts) >= source.Limit {
				break
			}
		}
		if page.After == "" {
			break
		}
		after = page.After
		count += len(page.Posts)
	}
	return posts, nil
}

type ListingPage struct {
	Posts  []post.Post
	After  string
	Before string
}

type redditThing struct {
	Kind string          `json:"kind"`
	Data json.RawMessage `json:"data"`
}

type listingData struct {
	Children []redditThing `json:"children"`
	After    string        `json:"after"`
	Before   string        `json:"before"`
}

func ParseListing(data []byte) (ListingPage, error) {
	var page ListingPage
	if len(data) == 0 {
		return page, fmt.Errorf("empty reddit response")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return page, err
	}
	var first byte
	for _, b := range data {
		if b > ' ' {
			first = b
			break
		}
	}
	if first == '[' {
		var listings []json.RawMessage
		if err := json.Unmarshal(data, &listings); err != nil {
			return page, err
		}
		for _, listing := range listings {
			next, err := ParseListing(listing)
			if err != nil {
				return page, err
			}
			page.Posts = append(page.Posts, next.Posts...)
			if page.After == "" {
				page.After = next.After
			}
			if page.Before == "" {
				page.Before = next.Before
			}
		}
		return page, nil
	}

	var thing redditThing
	if err := json.Unmarshal(data, &thing); err != nil {
		return page, err
	}
	if thing.Kind != "Listing" && thing.Kind != "" {
		if thing.Kind == "t3" {
			p, err := parsePostThing(thing)
			if err != nil {
				return page, err
			}
			page.Posts = append(page.Posts, p)
		}
		return page, nil
	}
	var listing listingData
	if err := json.Unmarshal(thing.Data, &listing); err != nil {
		return page, err
	}
	page.After = listing.After
	page.Before = listing.Before
	for _, child := range listing.Children {
		if child.Kind != "t3" {
			continue
		}
		p, err := parsePostThing(child)
		if err != nil {
			return page, err
		}
		page.Posts = append(page.Posts, p)
	}
	return page, nil
}

func parsePostThing(thing redditThing) (post.Post, error) {
	var p post.Post
	if len(thing.Data) == 0 {
		return p, fmt.Errorf("post thing has no data")
	}
	if err := json.Unmarshal(thing.Data, &p); err != nil {
		return p, err
	}
	p.RawJSON = append(json.RawMessage(nil), thing.Data...)
	return post.Normalize(p), nil
}

func listingEndpoint(base string, limit int, after string, count int) string {
	values := url.Values{}
	if limit > 0 {
		values.Set("limit", fmt.Sprintf("%d", limit))
	}
	if after != "" {
		values.Set("after", after)
	}
	if count > 0 {
		values.Set("count", fmt.Sprintf("%d", count))
	}
	if strings.Contains(base, "?") {
		return base + "&" + values.Encode()
	}
	return base + "?" + values.Encode()
}
