package storage

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

func NewRunID() string {
	data := make([]byte, 8)
	if _, err := rand.Read(data); err != nil {
		return "run_" + time.Now().UTC().Format("20060102150405")
	}
	return "run_" + hex.EncodeToString(data)
}
