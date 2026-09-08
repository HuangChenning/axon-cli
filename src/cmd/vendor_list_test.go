package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kamusis/axon-cli/internal/config"
	"github.com/kamusis/axon-cli/internal/vendor"
	"github.com/spf13/cobra"
)

func TestVendorSyncStatus_Pending(t *testing.T) {
	resetVendorCache(t)

	v := config.Vendor{Name: "never-synced", Dest: "skills/foo"}
	status, err := vendorSyncStatus(t.TempDir(), v)
	if err != nil {
		t.Fatalf("vendorSyncStatus: %v", err)
	}
	if status != "pending" {
		t.Errorf("status = %q, want %q", status, "pending")
	}
}

func TestVendorSyncStatus_Synced(t *testing.T) {
	resetVendorCache(t)

	hubRoot := t.TempDir()
	destAbs := filepath.Join(hubRoot, "skills", "foo")
	if err := os.MkdirAll(destAbs, 0o755); err != nil {
		t.Fatal(err)
	}

	v := config.Vendor{Name: "foo-skill", Dest: "skills/foo"}
	if err := vendor.WriteVendorSHA(v.Name, "7d2a8f10abcdef"); err != nil {
		t.Fatal(err)
	}

	status, err := vendorSyncStatus(hubRoot, v)
	if err != nil {
		t.Fatalf("vendorSyncStatus: %v", err)
	}
	if status != "synced (7d2a8f10)" {
		t.Errorf("status = %q, want %q", status, "synced (7d2a8f10)")
	}
}

func TestVendorSyncStatus_MissingDest(t *testing.T) {
	resetVendorCache(t)

	hubRoot := t.TempDir()
	// Do NOT create skills/foo — destination was deleted after a prior sync.

	v := config.Vendor{Name: "foo-skill", Dest: "skills/foo"}
	if err := vendor.WriteVendorSHA(v.Name, "7d2a8f10abcdef"); err != nil {
		t.Fatal(err)
	}

	status, err := vendorSyncStatus(hubRoot, v)
	if err != nil {
		t.Fatalf("vendorSyncStatus: %v", err)
	}
	if status != "missing dest" {
		t.Errorf("status = %q, want %q", status, "missing dest")
	}
}

func TestVendorNames(t *testing.T) {
	vendors := []config.Vendor{{Name: "a"}, {Name: "b"}, {Name: "c"}}
	got := vendorNames(vendors)
	want := []string{"a", "b", "c"}
	if len(got) != len(want) {
		t.Fatalf("vendorNames = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("vendorNames[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

// ── unregisteredLocally ───────────────────────────────────────────────────────

func TestUnregisteredLocally_FlagsEntryMissingFromLocalConfig(t *testing.T) {
	local := []config.Vendor{{Name: "v1"}}
	registered := []vendor.RegistryEntry{
		{Name: "v1", Repo: "https://github.com/x/y.git", Dest: "skills/v1"},
		{Name: "v2", Repo: "https://github.com/x/z.git", Dest: "skills/v2"},
	}

	got := unregisteredLocally(local, registered)
	if len(got) != 1 || got[0].Name != "v2" {
		t.Errorf("unregisteredLocally = %+v, want only v2", got)
	}
}

func TestUnregisteredLocally_NoneMissing(t *testing.T) {
	local := []config.Vendor{{Name: "v1"}, {Name: "v2"}}
	registered := []vendor.RegistryEntry{{Name: "v1"}, {Name: "v2"}}

	got := unregisteredLocally(local, registered)
	if len(got) != 0 {
		t.Errorf("unregisteredLocally = %+v, want none missing", got)
	}
}

// TestUnregisteredLocally_EmptyLocalConfig covers the exact scenario reported:
// a machine with zero vendors configured locally (e.g. axon.yaml was never
// updated after a teammate added a vendor on another machine) should still see
// every registry entry flagged as missing.
func TestUnregisteredLocally_EmptyLocalConfig(t *testing.T) {
	registered := []vendor.RegistryEntry{{Name: "v1"}, {Name: "v2"}}

	got := unregisteredLocally(nil, registered)
	if len(got) != 2 {
		t.Errorf("unregisteredLocally = %+v, want both entries flagged", got)
	}
}

func TestCompleteVendorNames_FiltersByPrefix(t *testing.T) {
	// os.UserHomeDir() reads USERPROFILE on Windows and HOME on Unix.
	home := t.TempDir()
	for _, key := range []string{"HOME", "USERPROFILE"} {
		t.Setenv(key, home)
	}

	axonDir := filepath.Join(home, ".axon")
	if err := os.MkdirAll(axonDir, 0o755); err != nil {
		t.Fatal(err)
	}
	yaml := "repo_path: " + filepath.Join(home, "hub") + "\nvendors:\n" +
		"  - name: book-to-skill\n    repo: https://example.com/a/b\n    subdir: .\n    dest: skills/book\n" +
		"  - name: cangjie-skill\n    repo: https://example.com/a/c\n    subdir: .\n    dest: skills/cangjie\n"
	if err := os.WriteFile(filepath.Join(axonDir, "axon.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}

	matches, directive := completeVendorNames(nil, nil, "book")
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("directive = %v, want ShellCompDirectiveNoFileComp", directive)
	}
	if len(matches) != 1 || matches[0] != "book-to-skill" {
		t.Errorf("matches = %v, want [book-to-skill]", matches)
	}
}
