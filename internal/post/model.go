package post

import "encoding/json"

type Post struct {
	ID                    string                   `json:"id"`
	Name                  string                   `json:"name"`
	Subreddit             string                   `json:"subreddit"`
	SubredditNamePrefixed string                   `json:"subreddit_name_prefixed"`
	Author                string                   `json:"author"`
	AuthorFullname        string                   `json:"author_fullname"`
	Title                 string                   `json:"title"`
	SelfText              string                   `json:"selftext"`
	SelfTextHTML          string                   `json:"selftext_html"`
	URL                   string                   `json:"url"`
	URLOverriddenByDest   string                   `json:"url_overridden_by_dest"`
	Permalink             string                   `json:"permalink"`
	Domain                string                   `json:"domain"`
	IsSelf                bool                     `json:"is_self"`
	IsVideo               bool                     `json:"is_video"`
	IsGallery             bool                     `json:"is_gallery"`
	Over18                bool                     `json:"over_18"`
	Spoiler               bool                     `json:"spoiler"`
	RemovedByCategory     string                   `json:"removed_by_category"`
	CreatedUTC            float64                  `json:"created_utc"`
	NumComments           int                      `json:"num_comments"`
	Score                 int                      `json:"score"`
	PostHint              string                   `json:"post_hint"`
	GalleryData           *GalleryData             `json:"gallery_data"`
	MediaMetadata         map[string]MediaMetadata `json:"media_metadata"`
	Media                 *Media                   `json:"media"`
	SecureMedia           *Media                   `json:"secure_media"`
	Preview               *Preview                 `json:"preview"`
	CrosspostParent       string                   `json:"crosspost_parent"`
	CrosspostParentList   []Post                   `json:"crosspost_parent_list"`
	Thumbnail             string                   `json:"thumbnail"`
	RawJSON               json.RawMessage          `json:"raw_json"`
}

type GalleryData struct {
	Items []GalleryItem `json:"items"`
}

type GalleryItem struct {
	MediaID     string `json:"media_id"`
	ID          int    `json:"id"`
	Caption     string `json:"caption"`
	OutboundURL string `json:"outbound_url"`
}

type MediaMetadata struct {
	Status    string        `json:"status"`
	Mime      string        `json:"m"`
	Source    MediaSource   `json:"s"`
	Images    []MediaSource `json:"p"`
	Originals []MediaSource `json:"o"`
	ID        string        `json:"id"`
}

type MediaSource struct {
	URL    string `json:"u"`
	MP4    string `json:"mp4"`
	GIF    string `json:"gif"`
	Width  int    `json:"x"`
	Height int    `json:"y"`
}

type Media struct {
	RedditVideo *RedditVideo `json:"reddit_video"`
}

type RedditVideo struct {
	FallbackURL      string  `json:"fallback_url"`
	DashURL          string  `json:"dash_url"`
	HLSURL           string  `json:"hls_url"`
	Duration         int     `json:"duration"`
	Height           int     `json:"height"`
	Width            int     `json:"width"`
	ScrubberMediaURL string  `json:"scrubber_media_url"`
	BitrateKbps      int     `json:"bitrate_kbps"`
	Framerate        float64 `json:"framerate"`
	IsGIF            bool    `json:"is_gif"`
	HasAudio         bool    `json:"has_audio"`
}

type Preview struct {
	Images []PreviewImage `json:"images"`
}

type PreviewImage struct {
	ID          string                  `json:"id"`
	Source      PreviewSource           `json:"source"`
	Resolutions []PreviewSource         `json:"resolutions"`
	Variants    map[string]PreviewImage `json:"variants"`
}

type PreviewSource struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}
