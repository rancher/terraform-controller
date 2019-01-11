package writer

import (
	"os"
)

// Write files to disk at the specified path
func Write(contents []byte, path string) error {
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666)
	if err != nil {
		return err
	}

	defer file.Close()

	_, err = file.Write(contents)

	return err
}
