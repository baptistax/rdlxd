package post

func HasExpectedMedia(p Post) bool {
	if p.IsSelf {
		return p.SelfText != "" || p.Title != ""
	}
	if p.IsGallery || p.IsVideo {
		return true
	}
	if p.URLOverriddenByDest != "" || p.URL != "" {
		return true
	}
	if p.Preview != nil && len(p.Preview.Images) > 0 {
		return true
	}
	return false
}

func IsDeletedOrRemoved(p Post) bool {
	return p.RemovedByCategory != "" || p.Author == "[deleted]" || p.SelfText == "[removed]" || p.SelfText == "[deleted]"
}
