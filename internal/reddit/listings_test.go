package reddit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseListingExtractsT3Posts(t *testing.T) {
	page, err := ParseListing([]byte(`{
		"kind":"Listing",
		"data":{
			"after":"t3_next",
			"children":[
				{"kind":"t3","data":{"id":"abc","name":"t3_abc","subreddit":"example","title":"image","url":"https://i.redd.it/a.jpg"}},
				{"kind":"t1","data":{"id":"comment"}}
			]
		}
	}`))
	if err != nil {
		t.Fatalf("ParseListing returned error: %v", err)
	}
	if len(page.Posts) != 1 {
		t.Fatalf("posts = %d, want 1", len(page.Posts))
	}
	if page.Posts[0].Name != "t3_abc" || page.Posts[0].RawJSON == nil {
		t.Fatalf("unexpected post: %+v", page.Posts[0])
	}
	if page.After != "t3_next" {
		t.Fatalf("after = %q, want t3_next", page.After)
	}
}

func TestParseThreadArrayExtractsOnlyPost(t *testing.T) {
	page, err := ParseListing([]byte(`[
		{"kind":"Listing","data":{"children":[{"kind":"t3","data":{"id":"abc","name":"t3_abc","title":"post"}}]}},
		{"kind":"Listing","data":{"children":[{"kind":"t1","data":{"id":"def"}},{"kind":"more","data":{}}]}}
	]`))
	if err != nil {
		t.Fatalf("ParseListing returned error: %v", err)
	}
	if len(page.Posts) != 1 || page.Posts[0].Name != "t3_abc" {
		t.Fatalf("posts = %+v, want only t3_abc", page.Posts)
	}
}

func TestClientFetcherPaginatesAndRespectsLimit(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.URL.Query().Get("raw_json") != "1" {
			t.Fatalf("raw_json = %q, want 1", r.URL.Query().Get("raw_json"))
		}
		if r.Header.Get("User-Agent") != "rdlxd-test" {
			t.Fatalf("User-Agent = %q, want rdlxd-test", r.Header.Get("User-Agent"))
		}
		w.Header().Set("X-Ratelimit-Used", "1")
		w.Header().Set("X-Ratelimit-Remaining", "99")
		w.Header().Set("X-Ratelimit-Reset", "10")
		switch requests {
		case 1:
			if r.URL.Query().Get("limit") != "3" {
				t.Fatalf("limit = %q, want 3", r.URL.Query().Get("limit"))
			}
			_, _ = w.Write([]byte(`{"kind":"Listing","data":{"after":"t3_after","children":[{"kind":"t3","data":{"id":"a","name":"t3_a"}},{"kind":"t3","data":{"id":"b","name":"t3_b"}}]}}`))
		case 2:
			if r.URL.Query().Get("after") != "t3_after" {
				t.Fatalf("after = %q, want t3_after", r.URL.Query().Get("after"))
			}
			if r.URL.Query().Get("limit") != "1" {
				t.Fatalf("limit = %q, want 1", r.URL.Query().Get("limit"))
			}
			_, _ = w.Write([]byte(`{"kind":"Listing","data":{"after":"","children":[{"kind":"t3","data":{"id":"c","name":"t3_c"}}]}}`))
		default:
			t.Fatalf("unexpected request %d", requests)
		}
	}))
	defer server.Close()

	client := NewClient("rdlxd-test")
	client.PublicMode = true
	client.PublicBaseURL = server.URL
	source, err := ParseSource("r/example", 3)
	if err != nil {
		t.Fatalf("ParseSource returned error: %v", err)
	}
	posts, err := (ClientListingFetcher{Client: client}).FetchPosts(context.Background(), *source)
	if err != nil {
		t.Fatalf("FetchPosts returned error: %v", err)
	}
	if len(posts) != 3 {
		t.Fatalf("posts = %d, want 3", len(posts))
	}
	if requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
}

func TestClientAuthorizationHeaderOnlyAuthenticated(t *testing.T) {
	var authenticatedHeader string
	var publicHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "public") {
			publicHeader = r.Header.Get("Authorization")
		} else {
			authenticatedHeader = r.Header.Get("Authorization")
		}
		_, _ = w.Write([]byte(`{"kind":"Listing","data":{"children":[]}}`))
	}))
	defer server.Close()

	authClient := NewClient("rdlxd-test")
	authClient.OAuthBaseURL = server.URL
	authClient.AccessToken = "access-token"
	if _, _, err := authClient.GetRaw(context.Background(), "/auth"); err != nil {
		t.Fatalf("authenticated GetRaw returned error: %v", err)
	}
	publicClient := NewClient("rdlxd-test")
	publicClient.PublicMode = true
	publicClient.PublicBaseURL = server.URL
	if _, _, err := publicClient.GetRaw(context.Background(), "/public"); err != nil {
		t.Fatalf("public GetRaw returned error: %v", err)
	}
	if authenticatedHeader != "Bearer access-token" {
		t.Fatalf("authenticated Authorization = %q", authenticatedHeader)
	}
	if publicHeader != "" {
		t.Fatalf("public Authorization = %q, want empty", publicHeader)
	}
}

func TestClientParsesRateLimitHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Ratelimit-Used", "2")
		w.Header().Set("X-Ratelimit-Remaining", "98")
		w.Header().Set("X-Ratelimit-Reset", "42")
		_, _ = w.Write([]byte(`{"kind":"Listing","data":{"children":[]}}`))
	}))
	defer server.Close()

	client := NewClient("rdlxd-test")
	client.PublicMode = true
	client.PublicBaseURL = server.URL
	_, limit, err := client.GetRaw(context.Background(), "/r/example/hot")
	if err != nil {
		t.Fatalf("GetRaw returned error: %v", err)
	}
	if limit == nil || limit.Used != 2 || limit.Remaining != 98 || limit.Reset != 42 {
		t.Fatalf("limit = %+v", limit)
	}
}
