package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/kamusis/axon-cli/internal/config"
	"github.com/kamusis/axon-cli/internal/vendor"
	"github.com/spf13/cobra"
)

var vendorListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured vendor entries and their sync health",
	Long: `vendor list prints every entry in the 'vendors' block of ~/.axon/axon.yaml
alongside its local sync status, without touching the network.

STATUS is one of:
  synced (<sha>)  the last-mirrored commit, and the Hub destination exists
  pending         never synced (no recorded commit for this entry)
  missing dest    a commit was recorded, but the Hub destination is gone

Use 'axon vendor sync <name>' to (re-)sync a single entry shown here.`,
	Args: cobra.NoArgs,
	RunE: runVendorList,
}

func init() {
	vendorCmd.AddCommand(vendorListCmd)
}

func runVendorList(_ *cobra.Command, _ []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("cannot load config: %w\nRun 'axon init' first.", err)
	}

	if len(cfg.Vendors) == 0 {
		printWarn("", "no vendors configured — add a 'vendors' block to ~/.axon/axon.yaml")
	} else {
		if err := validateVendors(cfg.Vendors); err != nil {
			return err
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tDEST\tREF\tSTATUS\tREPO")
		for _, v := range cfg.Vendors {
			ref := v.Ref
			if ref == "" {
				ref = "main"
			}
			status, err := vendorSyncStatus(cfg.RepoPath, v)
			if err != nil {
				status = fmt.Sprintf("error: %v", err)
			}
			repoCol := v.Repo
			if v.Subdir != "." {
				repoCol = fmt.Sprintf("%s (%s)", v.Repo, v.Subdir)
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", v.Name, v.Dest, ref, status, repoCol)
		}
		if err := w.Flush(); err != nil {
			return err
		}
	}

	return warnUnconfiguredVendors(cfg)
}

// warnUnconfiguredVendors compares the Hub-tracked vendor registry (written by
// `axon vendor sync` — possibly on a different machine sharing this Hub)
// against the vendors configured locally, and warns about entries present in
// the Hub but missing here. This is the case where another machine mirrored a
// vendor and pushed it, this machine pulled the files via `axon sync`, but its
// own axon.yaml was never updated with that vendor's entry — so it would
// otherwise silently miss out on future re-syncs.
func warnUnconfiguredVendors(cfg *config.Config) error {
	registered, err := vendor.ReadRegistry(cfg.RepoPath)
	if err != nil {
		return err
	}
	missing := unregisteredLocally(cfg.Vendors, registered)
	if len(missing) == 0 {
		return nil
	}

	printBullet("In Hub but not in your axon.yaml (synced by another machine):")
	for _, e := range missing {
		ref := e.Ref
		if ref == "" {
			ref = "main"
		}
		printWarn(e.Name, fmt.Sprintf(
			"dest=%s repo=%s subdir=%s ref=%s — run 'axon vendor import %s' to add it",
			e.Dest, e.Repo, e.Subdir, ref, e.Name,
		))
	}
	return nil
}

// unregisteredLocally returns the registry entries whose name isn't present
// among the locally configured vendors, preserving registry order.
func unregisteredLocally(local []config.Vendor, registered []vendor.RegistryEntry) []vendor.RegistryEntry {
	names := make(map[string]struct{}, len(local))
	for _, v := range local {
		names[v.Name] = struct{}{}
	}
	var missing []vendor.RegistryEntry
	for _, e := range registered {
		if _, ok := names[e.Name]; !ok {
			missing = append(missing, e)
		}
	}
	return missing
}

// vendorSyncStatus reports the local sync health of a single vendor entry
// without any network access: "synced (<sha>)", "pending", or "missing dest".
func vendorSyncStatus(hubRoot string, v config.Vendor) (string, error) {
	storedSHA, err := vendor.ReadVendorSHA(v.Name)
	if err != nil {
		return "", fmt.Errorf("could not read stored SHA: %w", err)
	}
	if storedSHA == "" {
		return "pending", nil
	}

	cleanDest, err := vendor.ValidateDest(v.Dest)
	if err != nil {
		return "", err
	}
	info, statErr := os.Stat(filepath.Join(hubRoot, cleanDest))
	if statErr != nil {
		if os.IsNotExist(statErr) {
			return "missing dest", nil
		}
		return "", fmt.Errorf("cannot stat destination: %w", statErr)
	}
	if !info.IsDir() {
		return "missing dest", nil
	}

	return fmt.Sprintf("synced (%.8s)", storedSHA), nil
}
