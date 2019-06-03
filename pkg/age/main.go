package age

import (
	"fmt"
	"math"
	"time"
)

func Age(timestamp time.Time) string {
	now := time.Now()
	if timestamp.After(now) {
		return "to late"
	}

	diff := now.Sub(timestamp)
	hours := diff.Hours()

	age := ""
	days := float64(0)

	if hours >= 24 {
		days = math.Floor(hours / 24)
		age = age + fmt.Sprintf("%.0f", days) + "d"
	}

	hs, mf := math.Modf(hours)

	if hs > 0 || days > 0 {
		age = age + fmt.Sprintf("%.0f", hs) + "h"
	}
	ms := mf * 60

	if days < 1 && (ms > 0 || hs > 0) {
		age = age + fmt.Sprintf("%.0f", ms) + ""
	}

	return age
}
