package download

import (
	"fmt"
	"mime"
	"strings"

	"github.com/baptistax/rdlxd/internal/media"
)

type ContentTypeError struct {
	ContentType string
	Substatus   string
}

func (e ContentTypeError) Error() string {
	if e.ContentType == "" {
		return "content type is not media"
	}
	return fmt.Sprintf("content type is not media: %s", e.ContentType)
}

func NormalizeContentType(contentType string) string {
	if contentType == "" {
		return ""
	}
	parsed, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return strings.ToLower(strings.TrimSpace(contentType))
	}
	return strings.ToLower(parsed)
}

func IsMediaContentType(contentType string) bool {
	normalized := NormalizeContentType(contentType)
	return strings.HasPrefix(normalized, "image/") ||
		strings.HasPrefix(normalized, "video/") ||
		strings.HasPrefix(normalized, "audio/")
}

func ValidateCandidateContentType(candidate media.MediaCandidate, contentType string) error {
	normalized := NormalizeContentType(contentType)
	if candidate.MediaKind == media.MediaKindText {
		if normalized == "" || strings.HasPrefix(normalized, "text/") {
			return nil
		}
	}
	if candidate.ExpectedContentType != "" && normalized == NormalizeContentType(candidate.ExpectedContentType) {
		return nil
	}
	switch candidate.MediaKind {
	case media.MediaKindImage, media.MediaKindGIF:
		if strings.HasPrefix(normalized, "image/") {
			return nil
		}
	case media.MediaKindVideo:
		if strings.HasPrefix(normalized, "video/") {
			return nil
		}
	case media.MediaKindAudio:
		if strings.HasPrefix(normalized, "audio/") {
			return nil
		}
	case media.MediaKindUnknown:
		if IsMediaContentType(normalized) {
			return nil
		}
	}
	if normalized == "text/html" || normalized == "application/xhtml+xml" {
		return ContentTypeError{ContentType: normalized, Substatus: string(media.SubstatusExternalPageUnsupported)}
	}
	return ContentTypeError{ContentType: normalized, Substatus: string(media.SubstatusContentTypeNotMedia)}
}
