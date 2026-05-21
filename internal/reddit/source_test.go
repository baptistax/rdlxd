package reddit

import "testing"

func TestParseSourceRecognizesInitialInputs(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantType SourceType
		wantUser string
		wantSub  string
		wantPost string
		wantSort string
	}{
		{name: "user shorthand", input: "u/example_user", wantType: SourceUser, wantUser: "example_user", wantSort: "submitted"},
		{name: "user path", input: "/u/example-user", wantType: SourceUser, wantUser: "example-user", wantSort: "submitted"},
		{name: "user url", input: "https://www.reddit.com/user/example/submitted/", wantType: SourceUser, wantUser: "example", wantSort: "submitted"},
		{name: "subreddit shorthand", input: "r/example", wantType: SourceSubreddit, wantSub: "example", wantSort: "hot"},
		{name: "subreddit shorthand sort", input: "r/example/new", wantType: SourceSubreddit, wantSub: "example", wantSort: "new"},
		{name: "subreddit url", input: "https://www.reddit.com/r/example/", wantType: SourceSubreddit, wantSub: "example", wantSort: "hot"},
		{name: "subreddit new", input: "https://www.reddit.com/r/example/new/", wantType: SourceSubreddit, wantSub: "example", wantSort: "new"},
		{name: "subreddit top", input: "https://www.reddit.com/r/example/top/", wantType: SourceSubreddit, wantSub: "example", wantSort: "top"},
		{name: "subreddit controversial", input: "https://www.reddit.com/r/example/controversial/", wantType: SourceSubreddit, wantSub: "example", wantSort: "controversial"},
		{name: "comments url", input: "https://www.reddit.com/r/example/comments/abc123/title/", wantType: SourcePost, wantSub: "example", wantPost: "abc123"},
		{name: "short post url", input: "https://redd.it/abc123", wantType: SourcePost, wantPost: "abc123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source, err := ParseSource(tt.input, 25)
			if err != nil {
				t.Fatalf("ParseSource returned error: %v", err)
			}
			if source.Type != tt.wantType {
				t.Fatalf("type = %s, want %s", source.Type, tt.wantType)
			}
			if source.Username != tt.wantUser {
				t.Fatalf("username = %q, want %q", source.Username, tt.wantUser)
			}
			if source.Subreddit != tt.wantSub {
				t.Fatalf("subreddit = %q, want %q", source.Subreddit, tt.wantSub)
			}
			if source.PostID != tt.wantPost {
				t.Fatalf("post id = %q, want %q", source.PostID, tt.wantPost)
			}
			if source.Sort != tt.wantSort {
				t.Fatalf("sort = %q, want %q", source.Sort, tt.wantSort)
			}
			if source.Limit != 25 {
				t.Fatalf("limit = %d, want 25", source.Limit)
			}
		})
	}
}

func TestParseSourceRejectsInvalidName(t *testing.T) {
	if _, err := ParseSource("r/../secret", 10); err == nil {
		t.Fatal("expected invalid source error")
	}
}
