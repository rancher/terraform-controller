package digest

import (
	"crypto/sha256"
	"encoding/hex"
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
		digest.Write([]byte(key))
		digest.Write([]byte(data[key]))
	}

	return hex.EncodeToString(digest.Sum(nil))
}
