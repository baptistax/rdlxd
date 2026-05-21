package reddit

import (
	"net/http"
	"strconv"
)

type RateLimit struct {
	Used      float64
	Remaining float64
	Reset     float64
}

func ParseRateLimitHeaders(headers http.Header) RateLimit {
	return RateLimit{
		Used:      parseRateLimitHeader(headers.Get("X-Ratelimit-Used")),
		Remaining: parseRateLimitHeader(headers.Get("X-Ratelimit-Remaining")),
		Reset:     parseRateLimitHeader(headers.Get("X-Ratelimit-Reset")),
	}
}

func parseRateLimitHeader(value string) float64 {
	if value == "" {
		return 0
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0
	}
	return parsed
}
