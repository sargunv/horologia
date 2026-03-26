package types

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// Timestamptz constructs a valid pgtype.Timestamptz from a time.Time.
func Timestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}
