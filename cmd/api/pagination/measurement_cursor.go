// Package pagination allows to featch data by chunks
package pagination

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"
)

type MeasurementCursor struct {
	CreatedAt time.Time `json:"created_at"`
	ID        int64     `json:"id"`
}

func Encode(c MeasurementCursor) string {
	b, _ := json.Marshal(c)
	return base64.RawURLEncoding.EncodeToString(b)
}

func Decode(s string) (MeasurementCursor, error) {
	var c MeasurementCursor
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return c, fmt.Errorf("invalid cursor: %w", err)
	}
	if err := json.Unmarshal(b, &c); err != nil {
		return c, fmt.Errorf("invalid cursor payload: %w", err)
	}
	return c, nil
}
