package publisher

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zeebo/blake3"
)

// HashFile computes a file hash matching wrangler's hashFile function:
// hex(blake3(base64(content) + extension))[0:32]
func HashFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading file %s: %w", path, err)
	}
	b64 := base64.StdEncoding.EncodeToString(content)
	ext := strings.TrimPrefix(filepath.Ext(path), ".")
	hasher := blake3.New()
	_, _ = hasher.WriteString(b64 + ext)
	sum := hasher.Sum(nil)
	return hex.EncodeToString(sum)[:32], nil
}
