package storage

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

func strToText(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *s, Valid: true}
}

func timeToTimestamptz(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}

func int64ToInt4(v int64) pgtype.Int4 {
	return pgtype.Int4{Int32: int32(v), Valid: true}
}

func pgtextStr(t pgtype.Text) string { return t.String }

func pgtextPtr(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	s := t.String
	return &s
}

func pgtstzTime(t pgtype.Timestamptz) time.Time { return t.Time }

func pgtstzPtr(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	tt := t.Time
	return &tt
}

func pgfloat4Ptr(f pgtype.Float4) *float64 {
	if !f.Valid {
		return nil
	}
	v := float64(f.Float32)
	return &v
}

func pgint4Ptr(i pgtype.Int4) *int64 {
	if !i.Valid {
		return nil
	}
	v := int64(i.Int32)
	return &v
}
