package storage

func WriteFailedJSON(path string, rows []FailedPost) error {
	if rows == nil {
		rows = []FailedPost{}
	}
	return AtomicWriteJSON(path, rows)
}
