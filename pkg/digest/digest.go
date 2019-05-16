package digest

import (
	"crypto/sha256"
	"encoding/hex"
	"github.com/sirupsen/logrus"
	"sort"
)

func SHA256Map(data map[string]string) string {
	var (
		keys   []string
		digest = sha256.New()
	)

	if len(data) == 0 {
		return ""
	}

	for k := range data {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	for _, key := range keys {
		if _, err := digest.Write([]byte(key)); err != nil {
			logrus.Error("Failed to write to digest")
		}

		if _, err := digest.Write([]byte(data[key])); err != nil {
			logrus.Error("Failed to write to digest")
		}
	}

	return hex.EncodeToString(digest.Sum(nil))
}
