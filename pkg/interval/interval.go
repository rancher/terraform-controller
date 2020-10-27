package interval

import "time"

const (
	DefaultInterval = 30 * time.Minute
)

func NeedsUpdate(lastCheck time.Time, interval time.Duration) bool {
	if lastCheck.IsZero() {
		return true
	}

	if interval == 0 {
		interval = DefaultInterval
	}

	return time.Now().After(lastCheck.Add(interval))
}
