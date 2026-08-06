package manifest

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
)

const maxManifestBytes = 1 << 20

// LoadFile decodes and validates one strict JSON manifest.
func LoadFile(path string) (Manifest, error) {
	file, err := os.Open(path)
	if err != nil {
		return Manifest{}, fmt.Errorf("open manifest: %w", err)
	}
	defer file.Close()

	limited := io.LimitReader(file, maxManifestBytes+1)
	payload, err := io.ReadAll(limited)
	if err != nil {
		return Manifest{}, fmt.Errorf("read manifest: %w", err)
	}
	if len(payload) > maxManifestBytes {
		return Manifest{}, fmt.Errorf("manifest exceeds %d bytes", maxManifestBytes)
	}

	var result Manifest
	decoder := json.NewDecoder(newSliceReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&result); err != nil {
		return Manifest{}, fmt.Errorf("decode manifest: %w", err)
	}
	if err := ensureEOF(decoder); err != nil {
		return Manifest{}, err
	}
	if err := result.Validate(); err != nil {
		return Manifest{}, fmt.Errorf("validate manifest: %w", err)
	}
	return result, nil
}

func ensureEOF(decoder *json.Decoder) error {
	var trailing any
	if err := decoder.Decode(&trailing); errors.Is(err, io.EOF) {
		return nil
	} else if err != nil {
		return fmt.Errorf("decode trailing content: %w", err)
	}
	return errors.New("manifest contains multiple JSON values")
}

// sliceReader avoids retaining an os.File in tests while keeping decoding strict.
type sliceReader struct {
	data []byte
	off  int
}

func newSliceReader(data []byte) *sliceReader {
	return &sliceReader{data: data}
}

func (r *sliceReader) Read(p []byte) (int, error) {
	if r.off >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.off:])
	r.off += n
	return n, nil
}
