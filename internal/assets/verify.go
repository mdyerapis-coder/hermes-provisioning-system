package assets

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
)

// Verification is the result of hashing one local asset.
type Verification struct {
	Path     string
	Expected string
	Actual   string
	Match    bool
	Size     int64
}

// VerifyFile calculates the SHA-256 digest of path and compares it with expected.
func VerifyFile(path, expected string) (Verification, error) {
	expected = strings.ToLower(strings.TrimSpace(expected))
	decoded, err := hex.DecodeString(expected)
	if err != nil || len(decoded) != sha256.Size {
		return Verification{}, fmt.Errorf("expected SHA-256 must contain exactly 64 hexadecimal characters")
	}

	file, err := os.Open(path)
	if err != nil {
		return Verification{}, fmt.Errorf("open asset: %w", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return Verification{}, fmt.Errorf("stat asset: %w", err)
	}
	if !info.Mode().IsRegular() {
		return Verification{}, fmt.Errorf("asset is not a regular file: %s", path)
	}

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return Verification{}, fmt.Errorf("hash asset: %w", err)
	}
	actual := hex.EncodeToString(hash.Sum(nil))

	return Verification{
		Path:     path,
		Expected: expected,
		Actual:   actual,
		Match:    actual == expected,
		Size:     info.Size(),
	}, nil
}
