package vendor

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestReadRegistry_MissingFile(t *testing.T) {
	entries, err := ReadRegistry(t.TempDir())
	if err != nil {
		t.Fatalf("ReadRegistry: %v", err)
	}
	if entries != nil {
		t.Errorf("entries = %v, want nil for missing registry file", entries)
	}
}

func TestWriteRegistry_RoundTrip(t *testing.T) {
	hubRoot := t.TempDir()
	want := []RegistryEntry{
		{Name: "book-to-skill", Repo: "https://example.com/a/b", Subdir: ".", Dest: "skills/book"},
		{Name: "cangjie-skill", Repo: "https://example.com/a/c", Subdir: "skills", Dest: "skills/cangjie", Ref: "v1.2"},
	}
	if err := WriteRegistry(hubRoot, want); err != nil {
		t.Fatalf("WriteRegistry: %v", err)
	}

	got, err := ReadRegistry(hubRoot)
	if err != nil {
		t.Fatalf("ReadRegistry: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ReadRegistry = %+v, want %+v", got, want)
	}
}

func TestReadRegistry_InvalidYAML(t *testing.T) {
	hubRoot := t.TempDir()
	if err := WriteRegistry(hubRoot, nil); err != nil {
		t.Fatal(err)
	}
	// Corrupt the file with invalid YAML.
	if err := os.WriteFile(filepath.Join(hubRoot, RegistryFileName), []byte("vendors: [this is not valid: yaml"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadRegistry(hubRoot); err == nil {
		t.Error("expected error for invalid YAML")
	}
}
