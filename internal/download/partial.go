package download

type PartialDownloadState struct {
	ETag            string
	LastModified    string
	ContentLength   int64
	BytesDownloaded int64
	PartialPath     string
	ResumeSupported bool
	LastHTTPStatus  int
}
