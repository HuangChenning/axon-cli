package cmd

import (
	"testing"

	"github.com/kamusis/axon-cli/internal/config"
	"github.com/kamusis/axon-cli/internal/vendor"
)

// ── selectMissingByName ────────────────────────────────────────────────────────

func TestSelectMissingByName_Found(t *testing.T) {
	missing := []vendor.RegistryEntry{
		{Name: "v1", Repo: "https://github.com/x/y.git", Dest: "skills/v1"},
		{Name: "v2", Repo: "https://github.com/x/z.git", Dest: "skills/v2"},
	}
	got, err := selectMissingByName(missing, "v2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].Name != "v2" {
		t.Errorf("selectMissingByName(v2) = %+v, want single entry named v2", got)
	}
}

func TestSelectMissingByName_NotFound(t *testing.T) {
	missing := []vendor.RegistryEntry{{Name: "v1"}}
	if _, err := selectMissingByName(missing, "ghost"); err == nil {
		t.Fatal("expected error for unknown vendor name")
	}
}

// ── appendVendorsFromRegistry ─────────────────────────────────────────────────

func TestAppendVendorsFromRegistry_AppendsMissing(t *testing.T) {
	local := []config.Vendor{{Name: "v1", Repo: "https://github.com/x/y.git", Dest: "skills/v1"}}
	entries := []vendor.RegistryEntry{
		{Name: "v2", Repo: "https://github.com/x/z.git", Subdir: "sub", Dest: "skills/v2", Ref: "main"},
	}

	got := appendVendorsFromRegistry(local, entries)
	if len(got) != 2 {
		t.Fatalf("appendVendorsFromRegistry = %+v, want 2 entries", got)
	}
	if got[0].Name != "v1" {
		t.Errorf("existing entry order changed: got[0] = %+v", got[0])
	}
	want := config.Vendor{Name: "v2", Repo: "https://github.com/x/z.git", Subdir: "sub", Dest: "skills/v2", Ref: "main"}
	if got[1] != want {
		t.Errorf("appended entry = %+v, want %+v", got[1], want)
	}
}

func TestAppendVendorsFromRegistry_EmptyLocal(t *testing.T) {
	entries := []vendor.RegistryEntry{{Name: "v1", Dest: "skills/v1"}}
	got := appendVendorsFromRegistry(nil, entries)
	if len(got) != 1 || got[0].Name != "v1" {
		t.Errorf("appendVendorsFromRegistry(nil, ...) = %+v, want single v1 entry", got)
	}
}

func TestAppendVendorsFromRegistry_DoesNotMutateLocalBackingArray(t *testing.T) {
	local := make([]config.Vendor, 1, 4)
	local[0] = config.Vendor{Name: "v1"}
	before := local[:1:1] // separate backing array to detect aliasing issues

	_ = appendVendorsFromRegistry(local, []vendor.RegistryEntry{{Name: "v2"}})

	if len(local) != 1 || local[0] != before[0] {
		t.Errorf("appendVendorsFromRegistry mutated caller's local slice: %+v", local)
	}
}
