package resolvers

import (
	"net/url"
	"path/filepath"
	"strings"

	"github.com/baptistax/rdlxd/internal/media"
	"github.com/baptistax/rdlxd/internal/post"
)

func postID(p post.Post) string {
	if p.Name != "" {
		return p.Name
	}
	if p.ID != "" {
		return "t3_" + p.ID
	}
	return ""
}

func mediaURL(p post.Post) string {
	if p.URLOverriddenByDest != "" {
		return p.URLOverriddenByDest
	}
	return p.URL
}

func mediaURLSourceField(p post.Post) string {
	if p.URLOverriddenByDest != "" {
		return "url_overridden_by_dest"
	}
	return "url"
}

func kindFromMime(mime string) media.MediaKind {
	lower := strings.ToLower(mime)
	switch {
	case strings.HasPrefix(lower, "image/gif"):
		return media.MediaKindGIF
	case strings.HasPrefix(lower, "image/"):
		return media.MediaKindImage
	case strings.HasPrefix(lower, "video/"):
		return media.MediaKindVideo
	case strings.HasPrefix(lower, "audio/"):
		return media.MediaKindAudio
	default:
		return media.MediaKindUnknown
	}
}

func kindFromURL(rawURL string) media.MediaKind {
	parsed, err := url.Parse(rawURL)
	pathValue := rawURL
	if err == nil {
		pathValue = parsed.Path
	}
	switch strings.ToLower(filepath.Ext(pathValue)) {
	case ".jpg", ".jpeg", ".png", ".webp", ".bmp", ".avif":
		return media.MediaKindImage
	case ".gif":
		return media.MediaKindGIF
	case ".mp4", ".m4v", ".mov", ".webm":
		return media.MediaKindVideo
	case ".mp3", ".m4a", ".wav", ".aac", ".ogg":
		return media.MediaKindAudio
	default:
		return media.MediaKindUnknown
	}
}

func extensionFromMime(mime string) string {
	switch strings.ToLower(mime) {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	case "video/mp4":
		return ".mp4"
	case "video/webm":
		return ".webm"
	case "audio/mpeg":
		return ".mp3"
	case "audio/mp4":
		return ".m4a"
	default:
		return ""
	}
}

func extensionFromURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	pathValue := rawURL
	if err == nil {
		pathValue = parsed.Path
	}
	ext := strings.ToLower(filepath.Ext(pathValue))
	if len(ext) > 12 {
		return ""
	}
	return ext
}

func isHTTPURL(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}

func isRedditMediaHost(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	return host == "i.redd.it" || host == "v.redd.it" || host == "preview.redd.it" || host == "external-preview.redd.it"
}

func isExternalPreviewHost(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	return strings.ToLower(parsed.Hostname()) == "external-preview.redd.it"
}

func isRedditPageURL(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	return host == "reddit.com" || host == "www.reddit.com" || host == "old.reddit.com" || host == "new.reddit.com" || host == "redd.it"
}
