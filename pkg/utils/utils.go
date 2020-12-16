package utils

import "time"

func MakeMillisecondTimestamp(t time.Time) uint64 {
	return uint64(t.UnixNano() / int64(time.Millisecond))
}
