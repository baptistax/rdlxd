package media

type MediaKind string

const (
	MediaKindImage   MediaKind = "image"
	MediaKindVideo   MediaKind = "video"
	MediaKindAudio   MediaKind = "audio"
	MediaKindGIF     MediaKind = "gif"
	MediaKindText    MediaKind = "text"
	MediaKindUnknown MediaKind = "unknown"
)

type MediaCandidate struct {
	CandidateID          string
	PostID               string
	ParentPostID         string
	URL                  string
	ResolverName         string
	MediaKind            MediaKind
	ContentRole          string
	Quality              string
	Required             bool
	Index                int
	Width                int
	Height               int
	ExpectedContentType  string
	ExpectedExtension    string
	SourceField          string
	RequiresProbe        bool
	Notes                []string
	Alternatives         []MediaAlternative
	Unsupported          bool
	UnsupportedSubstatus string
	UnsupportedReason    string
}

type MediaAlternative struct {
	URL         string
	SourceField string
	Quality     string
	Width       int
	Height      int
}
