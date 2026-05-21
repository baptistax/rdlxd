package output

import (
	"fmt"
	"strings"

	"github.com/baptistax/rdlxd/internal/storage"
)

func FormatFailedRows(rows []storage.FailedPost) string {
	if len(rows) == 0 {
		return "No incomplete posts.\n"
	}
	var builder strings.Builder
	builder.WriteString("Permalink\tStatus\tReason\tRetryable\n")
	for _, row := range rows {
		retryable := "false"
		if row.Retryable {
			retryable = "true"
		}
		fmt.Fprintf(&builder, "%s\t%s\t%s\t%s\n", row.Permalink, row.Status, row.Reason, retryable)
	}
	return builder.String()
}
