package post

import "strings"

func Normalize(p Post) Post {
	if p.Name == "" && p.ID != "" {
		p.Name = "t3_" + p.ID
	}
	if p.ID == "" && strings.HasPrefix(p.Name, "t3_") {
		p.ID = strings.TrimPrefix(p.Name, "t3_")
	}
	if p.SubredditNamePrefixed == "" && p.Subreddit != "" {
		p.SubredditNamePrefixed = "r/" + p.Subreddit
	}
	if p.Permalink != "" && strings.HasPrefix(p.Permalink, "/") {
		p.Permalink = "https://www.reddit.com" + p.Permalink
	}
	return p
}

func PostFolderName(p Post) string {
	normalized := Normalize(p)
	if normalized.Name != "" {
		return normalized.Name
	}
	if normalized.ID != "" {
		return "t3_" + normalized.ID
	}
	return "t3_unknown"
}
