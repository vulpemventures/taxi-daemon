package testutils

import (
	"log"
	"time"
)

// TimeTrack is using to log the time of a function
func TimeTrack(start time.Time, name string) {
	elapsed := time.Since(start)
	log.Printf("%s took %s", name, elapsed)
}
