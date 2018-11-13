package interval

import "time"

const (
	defaultInterval = 30 * time.Minute
)

func NeedsUpdate(lastCheck time.Time, interval time.Duration) bool {
	if lastCheck.IsZero() {
		return true
	}

	if interval == 0 {
		interval = defaultInterval
	}

	return time.Now().After(lastCheck.Add(interval))
}
