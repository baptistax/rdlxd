package reddit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientPostFetcherUsesAPIInfoListing(t *testing.T) {
	var requested string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requested = r.URL.Path + "?" + r.URL.RawQuery
		if r.URL.Path != "/api/info.json" {
			t.Fatalf("path = %s, want /api/info.json", r.URL.Path)
		}
		if r.URL.Query().Get("id") != "t3_abc123" {
			t.Fatalf("id = %q, want t3_abc123", r.URL.Query().Get("id"))
		}
		_, _ = w.Write([]byte(`{"kind":"Listing","data":{"children":[{"kind":"t3","data":{"id":"abc123","name":"t3_abc123","title":"single"}}]}}`))
	}))
	defer server.Close()

	client := NewClient("rdlxd-test")
	client.PublicMode = true
	client.PublicBaseURL = server.URL
	source, err := ParseSource("https://redd.it/abc123", 1)
	if err != nil {
		t.Fatalf("ParseSource returned error: %v", err)
	}
	post, err := (ClientPostFetcher{Client: client}).FetchPost(context.Background(), *source)
	if err != nil {
		t.Fatalf("FetchPost returned error: %v", err)
	}
	if post.Name != "t3_abc123" {
		t.Fatalf("post = %+v", post)
	}
	if requested == "" {
		t.Fatal("server was not requested")
	}
}
