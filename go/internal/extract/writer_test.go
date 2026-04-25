package extract

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestChunkWriterWriteChunk(t *testing.T) {
	dir := t.TempDir()
	taskDir := filepath.Join(dir, "testTask")
	w, err := NewChunkWriter(taskDir)
	if err != nil {
		t.Fatal(err)
	}

	items := []json.RawMessage{
		json.RawMessage(`{"key":"p1"}`),
		json.RawMessage(`{"key":"p2"}`),
	}
	if err := w.WriteChunk(items); err != nil {
		t.Fatal(err)
	}

	// Verify file exists and content is correct.
	content, err := os.ReadFile(filepath.Join(taskDir, "results.1.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	expected := "{\"key\":\"p1\"}\n{\"key\":\"p2\"}\n"
	if string(content) != expected {
		t.Errorf("expected %q, got %q", expected, string(content))
	}
}

func TestChunkWriterWriteOne(t *testing.T) {
	dir := t.TempDir()
	taskDir := filepath.Join(dir, "testTask")
	w, err := NewChunkWriter(taskDir)
	if err != nil {
		t.Fatal(err)
	}

	if err := w.WriteOne(json.RawMessage(`{"single":true}`)); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(filepath.Join(taskDir, "results.1.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	expected := "{\"single\":true}\n"
	if string(content) != expected {
		t.Errorf("expected %q, got %q", expected, string(content))
	}
}

func TestChunkWriterEmptySlice(t *testing.T) {
	dir := t.TempDir()
	taskDir := filepath.Join(dir, "testTask")
	w, err := NewChunkWriter(taskDir)
	if err != nil {
		t.Fatal(err)
	}

	if err := w.WriteChunk(nil); err != nil {
		t.Fatal(err)
	}

	// No file should be created.
	entries, _ := os.ReadDir(taskDir)
	if len(entries) != 0 {
		t.Errorf("expected no files, got %d", len(entries))
	}
}

func TestChunkWriterConcurrentWrites(t *testing.T) {
	dir := t.TempDir()
	taskDir := filepath.Join(dir, "testTask")
	w, err := NewChunkWriter(taskDir)
	if err != nil {
		t.Fatal(err)
	}

	const goroutines = 10
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			_ = w.WriteOne(json.RawMessage(`{"concurrent":true}`))
		}()
	}
	wg.Wait()

	entries, err := os.ReadDir(taskDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != goroutines {
		t.Errorf("expected %d files, got %d", goroutines, len(entries))
	}
}
