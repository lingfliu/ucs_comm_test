package utils

import "time"

func CurrentTimeInNano() int64 {
	return time.Now().UnixNano()
}
func CurrentTimeInMilli() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

func CurrentTimeInMicro() int64 {
	return time.Now().UnixNano() / 1000
}
