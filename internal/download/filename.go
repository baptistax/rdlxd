package download

import (
	"fmt"
	"mime"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/baptistax/rdlxd/internal/media"
	"github.com/baptistax/rdlxd/internal/storage"
)

func SanitizeFileName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	name = strings.ReplaceAll(name, ":", "_")
	name = strings.ReplaceAll(name, "\x00", "")
	var builder strings.Builder
	for _, r := range name {
		if r < 32 {
			builder.WriteRune('_')
			continue
		}
		switch r {
		case '<', '>', '"', '|', '?', '*':
			builder.WriteRune('_')
		default:
			builder.WriteRune(r)
		}
	}
	cleaned := strings.Trim(builder.String(), " .")
	for strings.Contains(cleaned, "..") {
		cleaned = strings.ReplaceAll(cleaned, "..", "__")
	}
	if cleaned == "" || cleaned == "." || cleaned == ".." {
		return "file"
	}
	if isWindowsReservedName(cleaned) {
		cleaned = "_" + cleaned
	}
	if len(cleaned) > 120 {
		cleaned = cleaned[:120]
	}
	return cleaned
}

func FinalFileName(candidate media.MediaCandidate, contentType string) string {
	if candidate.MediaKind == media.MediaKindText {
		if strings.TrimSpace(candidate.PostID) != "" {
			return SanitizeFileName(candidate.PostID + "_text.txt")
		}
		return "text.txt"
	}
	ext := ExtensionForContentTypeOrURL(contentType, candidate.URL, candidate.ExpectedExtension)
	index := candidate.Index
	if index <= 0 {
		index = 1
	}
	prefix := ""
	if strings.TrimSpace(candidate.PostID) != "" {
		prefix = SanitizeFileName(candidate.PostID)
	}
	if prefix == "" {
		return SanitizeFileName(fmt.Sprintf("%03d%s", index, ext))
	}
	return SanitizeFileName(fmt.Sprintf("%s_%03d%s", prefix, index, ext))
}

func ExtensionForContentTypeOrURL(contentType, rawURL, fallback string) string {
	normalized := NormalizeContentType(contentType)
	if normalized != "" {
		switch normalized {
		case "image/jpeg":
			return ".jpg"
		case "video/mp4":
			return ".mp4"
		}
		if exts, err := mime.ExtensionsByType(normalized); err == nil && len(exts) > 0 {
			return normalizeExtension(exts[0])
		}
	}
	if ext := extensionFromURL(rawURL); ext != "" {
		return ext
	}
	if fallback != "" {
		return normalizeExtension(fallback)
	}
	return ".bin"
}

func SafeFinalPath(rootDir, fileName string) (string, error) {
	name := SanitizeFileName(fileName)
	return storage.SafeJoin(rootDir, name)
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
	return normalizeExtension(ext)
}

func normalizeExtension(ext string) string {
	ext = strings.ToLower(strings.TrimSpace(ext))
	if ext == "" {
		return ""
	}
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	if len(ext) > 12 {
		return ""
	}
	return ext
}

func isWindowsReservedName(name string) bool {
	stem := strings.ToUpper(name)
	if index := strings.Index(stem, "."); index >= 0 {
		stem = stem[:index]
	}
	switch stem {
	case "CON", "PRN", "AUX", "NUL", "COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9", "LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9":
		return true
	default:
		return false
	}
}
