package timestamp

import "time"

// MakeMillisecondTimestamp is using to convert time.Time to a number of milliseconds
func MakeMillisecondTimestamp(t time.Time) uint64 {
	return uint64(t.UnixNano() / int64(time.Millisecond))
}
