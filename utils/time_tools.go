package utils

import "time"

func CurrentTimeInMilli() int64 {
	// return time.Now().UnixNano() / int64(time.Millisecond)
	return time.Now().UnixNano()
}
