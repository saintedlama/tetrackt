package persistence

import (
	"io"
	"os"
	"path/filepath"
)

// WriteFileAtomically writes to outputPath atomically: the write function receives
// a temporary file; on success the temp file is renamed to outputPath.
func WriteFileAtomically(outputPath string, write func(w io.Writer) error) error {
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return err
	}
	dir := filepath.Dir(outputPath)
	tmp, err := os.CreateTemp(dir, "tetrackt-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	cleanup := func() {
		tmp.Close()
		_ = os.Remove(tmpPath)
	}
	if err := write(tmp); err != nil {
		cleanup()
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	_ = os.Remove(outputPath)
	if err := os.Rename(tmpPath, outputPath); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
}
