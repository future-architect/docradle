package docradle

import (
	"path/filepath"
	"strings"
)

// SearchFiles searches filepathes to match pattern
func SearchFiles(patterns, cwd string) ([]string, error) {
	dir := cwd
	patternList := strings.Split(patterns, ",")
	for {
		for _, pattern := range patternList {
			files, err := filepath.Glob(filepath.Join(dir, pattern))
			if err != nil {
				return nil, err
			}
			if len(files) > 0 {
				return files, nil
			}
		}
		parentDir := filepath.Dir(dir)
		if dir == parentDir {
			break
		}
		dir = parentDir
	}
	return nil, nil
}
