package common

import (
	"path/filepath"
	"strings"
	"fmt"
)


// SafePathJoin joins base with userInput 
// and checks for dict-traversal attach
func SafePathJoin(base, userInput string) (string, error) {
	joined := filepath.Join(base, userInput)
	if !strings.HasPrefix(joined, filepath.Clean(base) + string(filepath.Separator)) {
		return "", fmt.Errorf("Path traversal detected")
	}
	return joined, nil
}