package storage

type StateStore interface {
	Close() error
	CreateRun(runID, source string) error
	FinishRun(runID, status string) error
	UpsertPost(record PostRecord) error
	UpsertAsset(record AssetRecord) error
	UpsertBlob(record BlobRecord) error
	RecordAttempt(record AttemptRecord) error
	ListIncompletePosts() ([]FailedPost, error)
	GetSummaryCounts() (SummaryCounts, error)
	MarkPostStatus(postID, status, substatus, reason string, retryable bool) error
	MarkAssetStatus(assetID, status, substatus, reason string, retryable bool) error
}

type PostRecord struct {
	PostID    string
	Name      string
	Permalink string
	Title     string
	Status    string
	Substatus string
	Retryable bool
	Reason    string
}

type AssetRecord struct {
	AssetID     string
	PostID      string
	CandidateID string
	URL         string
	Status      string
	Substatus   string
	BlobID      string
	Path        string
	Retryable   bool
	Reason      string
}

type BlobRecord struct {
	BlobID string
	SHA256 string
	Size   int64
	Path   string
}

type AttemptRecord struct {
	AttemptID       string
	PostID          string
	AssetID         string
	RunID           string
	Status          string
	ErrorCode       string
	Retryable       bool
	Message         string
	ETag            string
	LastModified    string
	ContentLength   int64
	BytesDownloaded int64
	PartialPath     string
	ResumeSupported bool
	LastHTTPStatus  int
}

type FailedPost struct {
	PostID              string            `json:"post_id"`
	Permalink           string            `json:"permalink"`
	Status              string            `json:"status"`
	Substatus           string            `json:"substatus,omitempty"`
	Reason              string            `json:"reason"`
	LastError           string            `json:"last_error,omitempty"`
	Retryable           bool              `json:"retryable"`
	CandidatesAttempted []FailedCandidate `json:"candidates_attempted,omitempty"`
}

type FailedCandidate struct {
	AssetID     string `json:"asset_id"`
	CandidateID string `json:"candidate_id"`
	URL         string `json:"url"`
	Status      string `json:"status"`
	Substatus   string `json:"substatus,omitempty"`
	Reason      string `json:"reason,omitempty"`
	Retryable   bool   `json:"retryable"`
}

type SummaryCounts struct {
	PostsFound         int `json:"posts_found"`
	Downloaded         int `json:"downloaded"`
	Partial            int `json:"partial"`
	Failed             int `json:"failed"`
	Unsupported        int `json:"unsupported"`
	Skipped            int `json:"skipped"`
	NotFullyDownloaded int `json:"not_fully_downloaded"`
}
