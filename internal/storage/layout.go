package storage

import (
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"strings"

	"github.com/baptistax/rdlxd/internal/reddit"
)

type Layout struct {
	RootDir      string
	SourceDir    string
	PostsDir     string
	MediaDir     string
	MetadataDir  string
	ReportsDir   string
	InternalDir  string
	StatePath    string
	ManifestPath string
	FailedPath   string
	LogsPath     string
	TempDir      string
	BlobsDir     string
}

func InitializeLayout(rootDir, sourceSlug string) (Layout, error) {
	if strings.TrimSpace(rootDir) == "" {
		return Layout{}, fmt.Errorf("output directory is required")
	}
	slug := SanitizeComponent(sourceSlug)
	if slug == "" {
		return Layout{}, fmt.Errorf("source slug is required")
	}
	rootAbs, err := filepath.Abs(filepath.Clean(rootDir))
	if err != nil {
		return Layout{}, err
	}
	sourceDir, err := SafeJoin(rootAbs, slug)
	if err != nil {
		return Layout{}, err
	}
	layout := LayoutFromSourceDir(sourceDir)
	layout.RootDir = rootAbs

	for _, dir := range []string{layout.SourceDir, layout.MediaDir, layout.MetadataDir, layout.ReportsDir, layout.InternalDir, layout.TempDir, layout.BlobsDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return Layout{}, err
		}
	}
	return layout, nil
}

func LayoutFromSourceDir(sourceDir string) Layout {
	clean := filepath.Clean(sourceDir)
	internal := filepath.Join(clean, ".rdlxd")
	metadata := filepath.Join(clean, "metadata")
	reports := filepath.Join(clean, "reports")
	return Layout{
		SourceDir:    clean,
		RootDir:      filepath.Dir(clean),
		PostsDir:     metadata,
		MediaDir:     filepath.Join(clean, "media"),
		MetadataDir:  metadata,
		ReportsDir:   reports,
		InternalDir:  internal,
		StatePath:    filepath.Join(internal, "state.db"),
		ManifestPath: filepath.Join(reports, "manifest.json"),
		FailedPath:   filepath.Join(reports, "failed.json"),
		LogsPath:     filepath.Join(internal, "logs.jsonl"),
		TempDir:      filepath.Join(internal, "temp"),
		BlobsDir:     filepath.Join(internal, "blobs"),
	}
}

func SafeJoin(root string, parts ...string) (string, error) {
	if strings.TrimSpace(root) == "" {
		return "", fmt.Errorf("root path is required")
	}
	rootAbs, err := filepath.Abs(filepath.Clean(root))
	if err != nil {
		return "", err
	}
	joined := rootAbs
	for _, part := range parts {
		if filepath.IsAbs(part) {
			return "", fmt.Errorf("absolute path segment is not allowed: %s", part)
		}
		joined = filepath.Join(joined, part)
	}
	finalAbs, err := filepath.Abs(filepath.Clean(joined))
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(rootAbs, finalAbs)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("path traversal blocked: %s", finalAbs)
	}
	return finalAbs, nil
}

func PostDir(layout Layout, postName string) (string, error) {
	name := SanitizeComponent(postName)
	if name == "" {
		return "", fmt.Errorf("post name is required")
	}
	return SafeJoin(layout.MetadataDir, name)
}

func SourceSlug(source *reddit.Source) string {
	if source == nil {
		return "unknown"
	}
	switch source.Type {
	case reddit.SourceSubreddit:
		return "r_" + SanitizeSlug(source.Subreddit)
	case reddit.SourceUser:
		return "u_" + SanitizeSlug(source.Username)
	case reddit.SourcePost:
		return "post_" + SanitizeSlug(source.PostID)
	default:
		return "url_" + shortSlugHash(source.RawInput)
	}
}

func SanitizeComponent(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	value = strings.ReplaceAll(value, string(filepath.Separator), "_")
	value = strings.ReplaceAll(value, "/", "_")
	value = strings.ReplaceAll(value, "\\", "_")
	value = SanitizeSlug(value)
	if value == "." || value == ".." {
		return ""
	}
	return value
}

func SanitizeSlug(value string) string {
	value = strings.TrimSpace(value)
	var builder strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			builder.WriteRune(r + 32)
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
		case r == '_' || r == '-':
			builder.WriteRune(r)
		default:
			builder.WriteRune('_')
		}
	}
	slug := strings.Trim(builder.String(), "_-.")
	if len(slug) > 80 {
		slug = slug[:80]
	}
	return slug
}

func shortSlugHash(value string) string {
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(value))
	return fmt.Sprintf("%08x", hash.Sum32())
}
