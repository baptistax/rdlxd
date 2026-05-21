package media

type Status string

const (
	StatusDownloaded  Status = "downloaded"
	StatusPartial     Status = "partial"
	StatusFailed      Status = "failed"
	StatusUnsupported Status = "unsupported"
	StatusSkipped     Status = "skipped"
)

type Substatus string

const (
	SubstatusTextPostSaved           Substatus = "text_post_saved"
	SubstatusNoMediaExpected         Substatus = "no_media_expected"
	SubstatusNoMediaCandidate        Substatus = "no_media_candidate"
	SubstatusAllCandidatesFailed     Substatus = "all_candidates_failed"
	SubstatusSomeCandidatesFailed    Substatus = "some_candidates_failed"
	SubstatusContentTypeNotMedia     Substatus = "content_type_not_media"
	SubstatusExternalPageUnsupported Substatus = "external_page_not_supported"
	SubstatusDeletedOrRemoved        Substatus = "deleted_or_removed"
	SubstatusAuthRequired            Substatus = "auth_required"
	SubstatusPrivateOrQuarantined    Substatus = "private_or_quarantined"
	SubstatusRateLimited             Substatus = "rate_limited"
	SubstatusRetryExhausted          Substatus = "retry_exhausted"
	SubstatusNSFWExcluded            Substatus = "nsfw_excluded"
	SubstatusBlobReused              Substatus = "blob_reused"
	SubstatusFileReused              Substatus = "file_reused"
	SubstatusVideoMayBeSilent        Substatus = "video_may_be_silent"
	SubstatusPreviewOnly             Substatus = "preview_only"
)

func IsNotFullyDownloaded(status string) bool {
	return status == string(StatusPartial) || status == string(StatusFailed) || status == string(StatusUnsupported)
}
