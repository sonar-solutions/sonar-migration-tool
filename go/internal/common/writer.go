package common

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
)

// ChunkWriter writes JSONL output for a single task.
// Thread-safe: concurrent goroutines can call WriteChunk / WriteOne
// and each gets a unique results.N.jsonl file via atomic indexing.
type ChunkWriter struct {
	dir      string
	chunkIdx int32 // atomic, 0-based; file names are 1-indexed
}

// NewChunkWriter creates a writer for the given task directory,
// creating it if necessary.
func NewChunkWriter(taskDir string) (*ChunkWriter, error) {
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating task dir %s: %w", taskDir, err)
	}
	return &ChunkWriter{dir: taskDir}, nil
}

// WriteChunk writes a slice of raw JSON objects as JSONL to results.N.jsonl.
// No-op for empty slices.
func (w *ChunkWriter) WriteChunk(objects []json.RawMessage) error {
	if len(objects) == 0 {
		return nil
	}
	idx := atomic.AddInt32(&w.chunkIdx, 1) // 1-indexed
	path := filepath.Join(w.dir, fmt.Sprintf("results.%d.jsonl", idx))
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating %s: %w", path, err)
	}
	defer f.Close()
	for _, obj := range objects {
		if _, err := f.Write(obj); err != nil {
			return err
		}
		if _, err := f.WriteString("\n"); err != nil {
			return err
		}
	}
	return nil
}

// WriteOne writes a single JSON object as a one-line chunk file.
func (w *ChunkWriter) WriteOne(obj json.RawMessage) error {
	return w.WriteChunk([]json.RawMessage{obj})
}
