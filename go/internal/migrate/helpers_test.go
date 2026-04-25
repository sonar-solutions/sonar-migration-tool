package migrate

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/sonar-solutions/sonar-migration-tool/internal/common"
)

func TestLoadCSVToJSONL(t *testing.T) {
	dir := t.TempDir()

	// Create a test CSV.
	csvContent := "name,value\nfoo,bar\nbaz,qux\n"
	if err := os.WriteFile(filepath.Join(dir, "test.csv"), []byte(csvContent), 0o644); err != nil {
		t.Fatal(err)
	}

	runDir := filepath.Join(dir, "run-01")
	os.MkdirAll(runDir, 0o755)
	store := common.NewDataStore(runDir)

	e := &Executor{
		Store:     store,
		ExportDir: dir,
	}

	if err := loadCSVToJSONL(e, "testTask", "test.csv"); err != nil {
		t.Fatalf("loadCSVToJSONL: %v", err)
	}

	items, err := store.ReadAll("testTask")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	// Verify first item.
	var row map[string]any
	json.Unmarshal(items[0], &row)
	if row["name"] != "foo" {
		t.Errorf("expected name=foo, got %v", row["name"])
	}
}

func TestLoadCSVToJSONLMissingFile(t *testing.T) {
	dir := t.TempDir()
	runDir := filepath.Join(dir, "run-01")
	os.MkdirAll(runDir, 0o755)
	store := common.NewDataStore(runDir)

	e := &Executor{
		Store:     store,
		ExportDir: dir,
	}

	// Missing CSV should result in empty output, not error (LoadCSV returns nil for missing).
	err := loadCSVToJSONL(e, "testTask", "nonexistent.csv")
	if err != nil {
		t.Fatalf("expected no error for missing CSV, got %v", err)
	}
}

func TestForEachMigrateItem(t *testing.T) {
	dir := t.TempDir()
	store := common.NewDataStore(dir)

	// Write test dependency data.
	w, _ := store.Writer("dep")
	w.WriteChunk([]json.RawMessage{
		json.RawMessage(`{"key":"a"}`),
		json.RawMessage(`{"key":"b"}`),
	})

	e := &Executor{
		Store: store,
		Sem:   make(chan struct{}, 5),
	}

	var count int
	err := forEachMigrateItem(context.Background(), e, "test", "dep",
		func(_ context.Context, item json.RawMessage, w *common.ChunkWriter) error {
			count++
			return nil
		})
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Errorf("expected 2 iterations, got %d", count)
	}
}

func TestForEachMigrateItemFiltered(t *testing.T) {
	dir := t.TempDir()
	store := common.NewDataStore(dir)

	w, _ := store.Writer("dep")
	w.WriteChunk([]json.RawMessage{
		json.RawMessage(`{"key":"a","skip":true}`),
		json.RawMessage(`{"key":"b","skip":false}`),
	})

	e := &Executor{
		Store: store,
		Sem:   make(chan struct{}, 5),
	}

	var keys []string
	err := forEachMigrateItemFiltered(context.Background(), e, "test", "dep",
		func(item json.RawMessage) bool {
			return !extractBool(item, "skip")
		},
		func(_ context.Context, item json.RawMessage, w *common.ChunkWriter) error {
			keys = append(keys, extractField(item, "key"))
			return nil
		})
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 1 || keys[0] != "b" {
		t.Errorf("expected [b], got %v", keys)
	}
}

func TestUnsupportedLanguages(t *testing.T) {
	if !unsupportedLanguages["c++"] {
		t.Error("expected c++ to be unsupported")
	}
	if unsupportedLanguages["java"] {
		t.Error("expected java to be supported")
	}
}

func TestValidPermissions(t *testing.T) {
	if !validPermissions["scan"] {
		t.Error("expected scan to be valid")
	}
	if validPermissions["delete"] {
		t.Error("expected delete to be invalid")
	}
}
