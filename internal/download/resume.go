package download

import "net/http"

type ResumeDecision string

const (
	ResumeWithRange ResumeDecision = "range"
	ResumeRestart   ResumeDecision = "restart"
	ResumeReconcile ResumeDecision = "reconcile"
)

func PrepareResumeHeaders(req *http.Request, state PartialDownloadState) ResumeDecision {
	if req == nil || !state.ResumeSupported || state.BytesDownloaded <= 0 {
		return ResumeRestart
	}
	req.Header.Set("Range", "bytes="+formatInt(state.BytesDownloaded)+"-")
	if state.ETag != "" {
		req.Header.Set("If-Range", state.ETag)
	} else if state.LastModified != "" {
		req.Header.Set("If-Range", state.LastModified)
	}
	return ResumeWithRange
}

func InterpretResumeStatus(statusCode int) ResumeDecision {
	switch statusCode {
	case http.StatusPartialContent:
		return ResumeWithRange
	case http.StatusRequestedRangeNotSatisfiable:
		return ResumeReconcile
	default:
		return ResumeRestart
	}
}

func formatInt(value int64) string {
	if value == 0 {
		return "0"
	}
	var digits [20]byte
	i := len(digits)
	for value > 0 {
		i--
		digits[i] = byte('0' + value%10)
		value /= 10
	}
	return string(digits[i:])
}
