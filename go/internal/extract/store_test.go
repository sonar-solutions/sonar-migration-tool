package extract

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDataStoreWriteAndReadAll(t *testing.T) {
	dir := t.TempDir()
	ds := NewDataStore(dir)

	w, err := ds.Writer("testTask")
	if err != nil {
		t.Fatal(err)
	}
	items := []json.RawMessage{
		json.RawMessage(`{"a":1}`),
		json.RawMessage(`{"a":2}`),
	}
	if err := w.WriteChunk(items); err != nil {
		t.Fatal(err)
	}

	got, err := ds.ReadAll("testTask")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 items, got %d", len(got))
	}
}

func TestDataStoreReadAllMultipleChunks(t *testing.T) {
	dir := t.TempDir()
	ds := NewDataStore(dir)

	w, err := ds.Writer("multiChunk")
	if err != nil {
		t.Fatal(err)
	}
	_ = w.WriteOne(json.RawMessage(`{"chunk":1}`))
	_ = w.WriteOne(json.RawMessage(`{"chunk":2}`))
	_ = w.WriteOne(json.RawMessage(`{"chunk":3}`))

	got, err := ds.ReadAll("multiChunk")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 items across chunks, got %d", len(got))
	}
}

func TestDataStoreReadAllMissing(t *testing.T) {
	ds := NewDataStore(t.TempDir())
	got, err := ds.ReadAll("nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestDataStoreCompletion(t *testing.T) {
	ds := NewDataStore(t.TempDir())
	if ds.IsComplete("task1") {
		t.Error("expected incomplete")
	}
	ds.MarkComplete("task1")
	if !ds.IsComplete("task1") {
		t.Error("expected complete")
	}
}

func TestDataStoreTaskDirExists(t *testing.T) {
	dir := t.TempDir()
	ds := NewDataStore(dir)

	if ds.TaskDirExists("nodir") {
		t.Error("expected false for non-existent dir")
	}

	_ = os.MkdirAll(filepath.Join(dir, "existingTask"), 0o755)
	if !ds.TaskDirExists("existingTask") {
		t.Error("expected true for existing dir")
	}
}
