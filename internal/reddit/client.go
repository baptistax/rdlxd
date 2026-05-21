package reddit

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultOAuthBaseURL  = "https://oauth.reddit.com"
	DefaultPublicBaseURL = "https://www.reddit.com"
)

type Client struct {
	HTTPClient    *http.Client
	OAuthBaseURL  string
	PublicBaseURL string
	UserAgent     string
	AccessToken   string
	PublicMode    bool
	RefreshToken  func(context.Context) (string, error)
	RetryAttempts int
	RetrySleep    func(context.Context, time.Duration) error
	Observer      func(RequestEvent)
}

type RequestEvent struct {
	Event      string
	URL        string
	StatusCode int
	RateLimit  *RateLimit
	Err        error
}

func NewClient(userAgent string) *Client {
	return &Client{
		HTTPClient:    &http.Client{Timeout: 30 * time.Second},
		OAuthBaseURL:  DefaultOAuthBaseURL,
		PublicBaseURL: DefaultPublicBaseURL,
		UserAgent:     userAgent,
		RetryAttempts: 3,
	}
}

func (c *Client) GetJSON(ctx context.Context, endpoint string, target any) (*RateLimit, error) {
	if target == nil {
		return nil, fmt.Errorf("target is required")
	}
	data, limit, err := c.GetRaw(ctx, endpoint)
	if err != nil {
		return limit, err
	}
	if err := json.Unmarshal(data, target); err != nil {
		return limit, err
	}
	return limit, nil
}

func (c *Client) GetRaw(ctx context.Context, endpoint string) ([]byte, *RateLimit, error) {
	requestURL, err := c.requestURL(endpoint)
	if err != nil {
		return nil, nil, err
	}
	var lastRateLimit *RateLimit
	attempts := c.retryAttempts()
	refreshed := false
	for attempt := 1; attempt <= attempts; attempt++ {
		data, limit, statusCode, retryAfter, err := c.getRawOnce(ctx, requestURL)
		if limit != nil {
			lastRateLimit = limit
		}
		if err == nil {
			return data, limit, nil
		}
		if statusCode == http.StatusUnauthorized && !c.PublicMode && !refreshed && c.RefreshToken != nil {
			token, refreshErr := c.RefreshToken(ctx)
			if refreshErr == nil && strings.TrimSpace(token) != "" {
				c.AccessToken = token
				refreshed = true
				continue
			}
			return nil, limit, AuthError{Message: "reddit authentication expired; run rdlxd auth again"}
		}
		if !isRetryableStatus(statusCode) && !isRetryableNetworkError(err) {
			return nil, limit, classifyHTTPError(statusCode, err)
		}
		if attempt == attempts {
			return nil, limit, classifyHTTPError(statusCode, err)
		}
		if sleepErr := c.sleep(ctx, retryDelay(attempt, retryAfter, limit)); sleepErr != nil {
			return nil, lastRateLimit, sleepErr
		}
	}
	return nil, lastRateLimit, fmt.Errorf("reddit request failed")
}

func (c *Client) getRawOnce(ctx context.Context, requestURL string) ([]byte, *RateLimit, int, time.Duration, error) {
	c.observe(RequestEvent{Event: "request_started", URL: requestURL})
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		c.observe(RequestEvent{Event: "request_finished", URL: requestURL, Err: err})
		return nil, nil, 0, 0, err
	}
	req.Header.Set("User-Agent", c.userAgent())
	if !c.PublicMode && c.AccessToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.AccessToken)
	}

	resp, err := c.httpClient().Do(req)
	if err != nil {
		c.observe(RequestEvent{Event: "request_finished", URL: requestURL, Err: err})
		return nil, nil, 0, 0, err
	}
	defer resp.Body.Close()

	limit := ParseRateLimitHeaders(resp.Header)
	c.observe(RequestEvent{Event: "rate_limit_observed", URL: requestURL, StatusCode: resp.StatusCode, RateLimit: &limit})
	retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		c.observe(RequestEvent{Event: "request_finished", URL: requestURL, StatusCode: resp.StatusCode, RateLimit: &limit})
		return nil, &limit, resp.StatusCode, retryAfter, ResponseError{
			StatusCode: resp.StatusCode,
			Code:       redditErrorCode(resp.StatusCode),
			Retryable:  isRetryableStatus(resp.StatusCode),
		}
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		c.observe(RequestEvent{Event: "request_finished", URL: requestURL, StatusCode: resp.StatusCode, RateLimit: &limit, Err: err})
		return nil, &limit, resp.StatusCode, retryAfter, err
	}
	c.observe(RequestEvent{Event: "request_finished", URL: requestURL, StatusCode: resp.StatusCode, RateLimit: &limit})
	return data, &limit, resp.StatusCode, retryAfter, nil
}

func (c *Client) requestURL(endpoint string) (string, error) {
	base := c.OAuthBaseURL
	if c.PublicMode {
		base = c.PublicBaseURL
	}
	if base == "" {
		return "", fmt.Errorf("base url is required")
	}
	parsedBase, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	parsedEndpoint, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}
	full := parsedBase.ResolveReference(parsedEndpoint)
	values := full.Query()
	values.Set("raw_json", "1")
	full.RawQuery = values.Encode()
	if !strings.HasSuffix(full.Path, ".json") && c.PublicMode {
		full.Path = strings.TrimRight(full.Path, "/") + ".json"
	}
	return full.String(), nil
}

type ResponseError struct {
	StatusCode int
	Code       string
	Retryable  bool
}

func (e ResponseError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("reddit request failed: %s", e.Code)
	}
	return fmt.Sprintf("reddit request failed with status %d", e.StatusCode)
}

type AuthError struct {
	Message string
}

func (e AuthError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return "reddit authentication is required for this source"
}

func classifyHTTPError(statusCode int, err error) error {
	if responseErr, ok := err.(ResponseError); ok {
		return responseErr
	}
	if statusCode == 0 {
		return fmt.Errorf("reddit request failed: %w", err)
	}
	return ResponseError{
		StatusCode: statusCode,
		Code:       redditErrorCode(statusCode),
		Retryable:  isRetryableStatus(statusCode),
	}
}

func redditErrorCode(statusCode int) string {
	switch statusCode {
	case http.StatusUnauthorized:
		return "auth_required"
	case http.StatusForbidden:
		return "private_or_forbidden"
	case http.StatusNotFound:
		return "not_found"
	case http.StatusGone:
		return "gone"
	case http.StatusTooManyRequests:
		return "rate_limited"
	case http.StatusRequestTimeout:
		return "request_timeout"
	default:
		if statusCode >= 500 {
			return "server_error"
		}
		if statusCode >= 400 {
			return "request_rejected"
		}
		return ""
	}
}

func isRetryableStatus(statusCode int) bool {
	switch statusCode {
	case http.StatusRequestTimeout, http.StatusTooManyRequests, http.StatusInternalServerError,
		http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

func isRetryableNetworkError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "timeout") ||
		strings.Contains(strings.ToLower(err.Error()), "temporary") ||
		strings.Contains(strings.ToLower(err.Error()), "connection reset")
}

func parseRetryAfter(value string) time.Duration {
	if value == "" {
		return 0
	}
	if seconds, err := strconv.Atoi(strings.TrimSpace(value)); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	if when, err := http.ParseTime(value); err == nil {
		if delay := time.Until(when); delay > 0 {
			return delay
		}
	}
	return 0
}

func retryDelay(attempt int, retryAfter time.Duration, limit *RateLimit) time.Duration {
	if retryAfter > 0 {
		return capDelay(retryAfter)
	}
	if limit != nil && limit.Reset > 0 && limit.Remaining <= 0 {
		return capDelay(time.Duration(limit.Reset) * time.Second)
	}
	delay := time.Duration(150*(1<<(attempt-1))) * time.Millisecond
	return capDelay(delay)
}

func capDelay(delay time.Duration) time.Duration {
	if delay <= 0 {
		return 0
	}
	if delay > 30*time.Second {
		return 30 * time.Second
	}
	return delay
}

func (c *Client) retryAttempts() int {
	if c.RetryAttempts > 0 {
		return c.RetryAttempts
	}
	return 1
}

func (c *Client) sleep(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}
	if c.RetrySleep != nil {
		return c.RetrySleep(ctx, delay)
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (c *Client) observe(event RequestEvent) {
	if c != nil && c.Observer != nil {
		c.Observer(event)
	}
}

func (c *Client) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return http.DefaultClient
}

func (c *Client) userAgent() string {
	if strings.TrimSpace(c.UserAgent) != "" {
		return c.UserAgent
	}
	return "rdlxd/0.1"
}
