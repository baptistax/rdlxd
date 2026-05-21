package output

import (
	"fmt"

	"github.com/baptistax/rdlxd/internal/storage"
)

func FormatDownloadSummary(source string, summary storage.SummaryCounts, outputPath string) string {
	return fmt.Sprintf(`Source: %s
Posts found: %d
Downloaded: %d
Partial: %d
Failed: %d
Unsupported: %d

Not fully downloaded: %d posts
Incomplete list: rdlxd failed %s
Output: %s
`, source, summary.PostsFound, summary.Downloaded, summary.Partial, summary.Failed, summary.Unsupported, summary.NotFullyDownloaded, outputPath, outputPath)
}

func FormatStatusSummary(summary storage.SummaryCounts) string {
	return fmt.Sprintf(`Posts found: %d
Downloaded: %d
Partial: %d
Failed: %d
Unsupported: %d
Not fully downloaded: %d posts
`, summary.PostsFound, summary.Downloaded, summary.Partial, summary.Failed, summary.Unsupported, summary.NotFullyDownloaded)
}
