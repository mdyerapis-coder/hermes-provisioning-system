package androiddist

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func canonicalDirectory(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", errors.New("path is required")
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	absolute = filepath.Clean(absolute)
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", err
	}
	resolved = filepath.Clean(resolved)
	if resolved != absolute {
		return "", errors.New("directory path must not contain symbolic links")
	}
	info, err := os.Lstat(absolute)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", errors.New("path is not a directory")
	}
	return absolute, nil
}

func canonicalRegularFile(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", errors.New("path is required")
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	absolute = filepath.Clean(absolute)
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", err
	}
	resolved = filepath.Clean(resolved)
	if resolved != absolute {
		return "", errors.New("file path must not contain symbolic links")
	}
	info, err := os.Lstat(absolute)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", errors.New("path is not a regular file")
	}
	return absolute, nil
}

func ensureSafeDirectoryTree(root, relative string) (string, error) {
	if err := validateRelativePath(relative); err != nil {
		return "", err
	}
	current := root
	for _, part := range strings.Split(filepath.ToSlash(relative), "/") {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if errors.Is(err, os.ErrNotExist) {
			if err := os.Mkdir(current, 0o755); err != nil {
				return "", fmt.Errorf("create repository directory %s: %w", current, err)
			}
			if err := os.Chmod(current, 0o755); err != nil {
				_ = os.Remove(current)
				return "", fmt.Errorf("set repository directory permissions %s: %w", current, err)
			}
			continue
		}
		if err != nil {
			return "", fmt.Errorf("inspect repository directory %s: %w", current, err)
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return "", fmt.Errorf("repository path component is not a real directory: %s", current)
		}
	}
	return current, nil
}

func decodeStrictFile(path string, maxBytes int64, target any) error {
	_, err := decodeStrictFileBytes(path, maxBytes, target)
	return err
}

func decodeStrictFileBytes(path string, maxBytes int64, target any) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, errors.New("JSON input must be a regular non-symbolic-link file")
	}
	if info.Size() > maxBytes {
		return nil, fmt.Errorf("JSON input exceeds %d bytes", maxBytes)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return nil, err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, errors.New("JSON input contains multiple values")
		}
		return nil, err
	}
	return data, nil
}

func marshalJSON(value any) ([]byte, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode JSON: %w", err)
	}
	return append(data, '\n'), nil
}

func writeExclusiveFile(path string, data []byte, mode os.FileMode) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	ok := false
	defer func() {
		_ = file.Close()
		if !ok {
			_ = os.Remove(path)
		}
	}()
	if err := file.Chmod(mode.Perm()); err != nil {
		return fmt.Errorf("set permissions on %s: %w", path, err)
	}
	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("sync %s: %w", path, err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close %s: %w", path, err)
	}
	ok = true
	return nil
}

func copyRegularFile(source, destination string, mode os.FileMode) (int64, string, error) {
	info, err := os.Lstat(source)
	if err != nil {
		return 0, "", fmt.Errorf("inspect source APK: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return 0, "", errors.New("source APK must be a regular non-symbolic-link file")
	}
	input, err := os.Open(source)
	if err != nil {
		return 0, "", fmt.Errorf("open source APK: %w", err)
	}
	defer input.Close()
	output, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return 0, "", fmt.Errorf("create staged APK: %w", err)
	}
	ok := false
	defer func() {
		_ = output.Close()
		if !ok {
			_ = os.Remove(destination)
		}
	}()
	if err := output.Chmod(mode.Perm()); err != nil {
		return 0, "", fmt.Errorf("set staged APK permissions: %w", err)
	}
	hash := sha256.New()
	written, err := io.Copy(io.MultiWriter(output, hash), bufio.NewReader(input))
	if err != nil {
		return 0, "", fmt.Errorf("copy APK: %w", err)
	}
	if err := output.Sync(); err != nil {
		return 0, "", fmt.Errorf("sync staged APK: %w", err)
	}
	if err := output.Close(); err != nil {
		return 0, "", fmt.Errorf("close staged APK: %w", err)
	}
	ok = true
	return written, hex.EncodeToString(hash.Sum(nil)), nil
}

func digestRegularFile(path string) (string, int64, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return "", 0, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return "", 0, errors.New("file must be a regular non-symbolic-link file")
	}
	file, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(hash.Sum(nil)), info.Size(), nil
}

func digestBytes(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

func normalizeDigest(value string) string {
	replacer := strings.NewReplacer(":", "", " ", "", "\t", "", "\r", "", "\n", "")
	return strings.ToLower(replacer.Replace(strings.TrimSpace(value)))
}

func validDigest(value string) bool {
	decoded, err := hex.DecodeString(normalizeDigest(value))
	return err == nil && len(decoded) == sha256.Size
}

func syncDirectory(path string) error {
	if err := os.Chmod(path, 0o755); err != nil {
		return fmt.Errorf("set directory permissions: %w", err)
	}
	directory, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open directory for sync: %w", err)
	}
	defer directory.Close()
	if err := directory.Sync(); err != nil {
		return fmt.Errorf("sync directory: %w", err)
	}
	return nil
}
