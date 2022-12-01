package prometheus

import (
	"time"

	"kubegems.io/kubegems/pkg/log"
)

// parse and set default value
func ParseRangeTime(startStr, endStr string, loc *time.Location) (time.Time, time.Time) {
	start, err1 := time.ParseInLocation(time.RFC3339, startStr, loc)
	end, err2 := time.ParseInLocation(time.RFC3339, endStr, loc)
	if err1 != nil || err2 != nil {
		log.Warnf("parse time failed, start: %v, end: %v", err1, err2)
		now := time.Now().In(loc)
		start = now.Add(-30 * time.Minute)
		end = now
	}
	return start, end
}
