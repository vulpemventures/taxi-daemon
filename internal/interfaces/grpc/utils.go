package grpc

import "time"

func makeMillisecondTimestamp(t time.Time) uint64 {
	return uint64(t.UnixNano() / int64(time.Millisecond))
}
