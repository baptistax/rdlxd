package reddit

import (
	"fmt"
	"net/url"
	"path"
	"strconv"
	"strings"
)

type SourceType string

const (
	SourceSubreddit SourceType = "subreddit"
	SourceUser      SourceType = "user"
	SourcePost      SourceType = "post"
	SourceURL       SourceType = "url"
)

type Source struct {
	Type                  SourceType
	RawInput              string
	CanonicalURL          string
	Subreddit             string
	Username              string
	PostID                string
	Sort                  string
	TimeRange             string
	Limit                 int
	AuthNeeded            bool
	Endpoint              string
	IncludeCommentsFuture bool
}

func ParseSource(input string, limit int) (*Source, error) {
	raw := strings.TrimSpace(input)
	if raw == "" {
		return nil, fmt.Errorf("source cannot be empty")
	}
	if limit < 0 {
		return nil, fmt.Errorf("limit must be zero or greater")
	}

	if parsed, err := url.Parse(raw); err == nil && parsed.Scheme != "" {
		return parseURLSource(raw, parsed, limit)
	}
	return parsePathSource(raw, limit)
}

func parsePathSource(raw string, limit int) (*Source, error) {
	clean := strings.Trim(path.Clean("/"+strings.TrimSpace(raw)), "/")
	parts := splitPath(clean)
	if len(parts) < 2 {
		return nil, fmt.Errorf("unsupported source: %s", raw)
	}

	switch strings.ToLower(parts[0]) {
	case "r":
		if len(parts) > 3 {
			return nil, fmt.Errorf("unsupported source: %s", raw)
		}
		if !isSafeRedditName(parts[1]) {
			return nil, fmt.Errorf("invalid subreddit name: %s", parts[1])
		}
		sort := ""
		if len(parts) == 3 && isSupportedSort(strings.ToLower(parts[2])) {
			sort = strings.ToLower(parts[2])
		}
		return subredditSource(raw, parts[1], sort, limit), nil
	case "u", "user":
		if len(parts) != 2 {
			return nil, fmt.Errorf("unsupported source: %s", raw)
		}
		if !isSafeRedditName(parts[1]) {
			return nil, fmt.Errorf("invalid username: %s", parts[1])
		}
		return userSource(raw, parts[1], limit), nil
	default:
		return nil, fmt.Errorf("unsupported source: %s", raw)
	}
}

func parseURLSource(raw string, parsed *url.URL, limit int) (*Source, error) {
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return &Source{
			Type:         SourceURL,
			RawInput:     raw,
			CanonicalURL: raw,
			Limit:        limit,
			AuthNeeded:   false,
			Endpoint:     raw,
		}, nil
	}
	host := strings.ToLower(parsed.Hostname())
	parts := splitPath(parsed.EscapedPath())

	if host == "redd.it" {
		if len(parts) < 1 || !isSafePostID(parts[0]) {
			return nil, fmt.Errorf("invalid redd.it post URL")
		}
		return postSource(raw, "", parts[0], limit), nil
	}

	if !isRedditHost(host) {
		return &Source{
			Type:         SourceURL,
			RawInput:     raw,
			CanonicalURL: raw,
			Limit:        limit,
			AuthNeeded:   false,
			Endpoint:     raw,
		}, nil
	}

	if len(parts) >= 2 && strings.EqualFold(parts[0], "user") {
		username, err := url.PathUnescape(parts[1])
		if err != nil {
			return nil, err
		}
		if !isSafeRedditName(username) {
			return nil, fmt.Errorf("invalid username: %s", username)
		}
		return userSource(raw, username, limit), nil
	}

	if len(parts) >= 2 && strings.EqualFold(parts[0], "r") {
		subreddit, err := url.PathUnescape(parts[1])
		if err != nil {
			return nil, err
		}
		if !isSafeRedditName(subreddit) {
			return nil, fmt.Errorf("invalid subreddit name: %s", subreddit)
		}
		if len(parts) >= 4 && strings.EqualFold(parts[2], "comments") {
			postID, err := url.PathUnescape(parts[3])
			if err != nil {
				return nil, err
			}
			if !isSafePostID(postID) {
				return nil, fmt.Errorf("invalid post id: %s", postID)
			}
			return postSource(raw, subreddit, postID, limit), nil
		}
		sort := ""
		if len(parts) >= 3 {
			sort = strings.ToLower(parts[2])
			if !isSupportedSort(sort) {
				sort = ""
			}
		}
		return subredditSource(raw, subreddit, sort, limit), nil
	}

	return &Source{
		Type:         SourceURL,
		RawInput:     raw,
		CanonicalURL: raw,
		Limit:        limit,
		AuthNeeded:   true,
		Endpoint:     parsed.RequestURI(),
	}, nil
}

func subredditSource(raw, subreddit, sort string, limit int) *Source {
	if sort == "" {
		sort = "hot"
	}
	endpoint := "/r/" + subreddit + "/" + sort
	return &Source{
		Type:         SourceSubreddit,
		RawInput:     raw,
		CanonicalURL: "https://www.reddit.com/r/" + subreddit + "/" + sort + "/",
		Subreddit:    subreddit,
		Sort:         sort,
		Limit:        limit,
		AuthNeeded:   true,
		Endpoint:     endpoint,
	}
}

func userSource(raw, username string, limit int) *Source {
	return &Source{
		Type:         SourceUser,
		RawInput:     raw,
		CanonicalURL: "https://www.reddit.com/user/" + username + "/submitted/",
		Username:     username,
		Sort:         "submitted",
		Limit:        limit,
		AuthNeeded:   true,
		Endpoint:     "/user/" + username + "/submitted",
	}
}

func postSource(raw, subreddit, postID string, limit int) *Source {
	canonical := "https://redd.it/" + postID
	endpoint := "/comments/" + postID
	if subreddit != "" {
		canonical = "https://www.reddit.com/r/" + subreddit + "/comments/" + postID + "/"
		endpoint = "/r/" + subreddit + "/comments/" + postID
	}
	return &Source{
		Type:         SourcePost,
		RawInput:     raw,
		CanonicalURL: canonical,
		Subreddit:    subreddit,
		PostID:       postID,
		Limit:        limit,
		AuthNeeded:   true,
		Endpoint:     endpoint,
	}
}

func splitPath(value string) []string {
	trimmed := strings.Trim(value, "/")
	if trimmed == "" {
		return nil
	}
	rawParts := strings.Split(trimmed, "/")
	parts := make([]string, 0, len(rawParts))
	for _, part := range rawParts {
		if part != "" {
			parts = append(parts, part)
		}
	}
	return parts
}

func isRedditHost(host string) bool {
	return host == "reddit.com" || host == "www.reddit.com" || host == "old.reddit.com" || host == "new.reddit.com"
}

func isSupportedSort(sort string) bool {
	switch sort {
	case "hot", "new", "top", "controversial", "rising":
		return true
	default:
		return false
	}
}

func isSafeRedditName(value string) bool {
	if value == "" || len(value) > 64 {
		return false
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			continue
		}
		return false
	}
	return true
}

func isSafePostID(value string) bool {
	if value == "" || len(value) > 32 {
		return false
	}
	for _, r := range strings.ToLower(value) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			continue
		}
		return false
	}
	return true
}

func SourceLimitQuery(limit int) string {
	if limit <= 0 {
		return ""
	}
	return "limit=" + strconv.Itoa(limit)
}
