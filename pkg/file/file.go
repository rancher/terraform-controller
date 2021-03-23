package file

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func Exists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func Touch(f string) (*os.File, error) {
	filename, err := HomeDir(f)
	if err != nil {
		return nil, err
	}

	if Exists(filename) {
		return os.Open(filename)
	}

	p, err := filepath.Abs(filepath.Dir(filename))
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(p); os.IsNotExist(err) {
		if err := os.Mkdir(p, fs.FileMode(0600)); err != nil {
			return nil, err
		}
	}

	return os.Create(filename)
}

func HomeDir(filename string) (string, error) {
	if strings.Contains(filename, "~/") {
		homedir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		filename = strings.Replace(filename, "~/", "", 1)
		filename = path.Join(homedir, filename)
	}
	return filename, nil
}
