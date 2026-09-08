package cmd

import (
	"fmt"
	"strings"

	"github.com/kamusis/axon-cli/internal/config"
	"github.com/kamusis/axon-cli/internal/vendor"
	"github.com/spf13/cobra"
)

var vendorImportCmd = &cobra.Command{
	Use:   "import [name]",
	Short: "Import vendor entries from the Hub registry into axon.yaml",
	Long: `vendor import reads the Hub-tracked vendor registry
(.axon-vendors.lock.yaml, written by 'axon vendor sync' — possibly run on
another machine sharing this Hub) and appends any entries missing from this
machine's axon.yaml 'vendors' block.

With no argument, every missing entry is imported. Given a name, only the
matching entry is imported.

Existing 'vendors' entries in axon.yaml are never modified — import only
appends entries that don't already exist locally by name. Run
'axon vendor sync' afterwards to actually mirror the newly added entries.`,
	Args:              cobra.MaximumNArgs(1),
	RunE:              runVendorImport,
	ValidArgsFunction: completeMissingVendorNames,
}

// completeMissingVendorNames provides shell <TAB> completion for
// `axon vendor import [name]`, suggesting registry entries not yet present
// in this machine's axon.yaml.
func completeMissingVendorNames(_ *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	cfg, err := config.Load()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	registered, err := vendor.ReadRegistry(cfg.RepoPath)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	var matches []string
	for _, e := range unregisteredLocally(cfg.Vendors, registered) {
		if strings.HasPrefix(e.Name, toComplete) {
			matches = append(matches, e.Name)
		}
	}
	return matches, cobra.ShellCompDirectiveNoFileComp
}

func init() {
	vendorCmd.AddCommand(vendorImportCmd)
}

func runVendorImport(_ *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("cannot load config: %w\nRun 'axon init' first.", err)
	}

	registered, err := vendor.ReadRegistry(cfg.RepoPath)
	if err != nil {
		return err
	}

	missing := unregisteredLocally(cfg.Vendors, registered)
	if len(args) == 1 {
		missing, err = selectMissingByName(missing, args[0])
		if err != nil {
			return err
		}
	}

	if len(missing) == 0 {
		printOK("", "nothing to import — axon.yaml already has every vendor in the Hub registry")
		return nil
	}

	cfg.Vendors = appendVendorsFromRegistry(cfg.Vendors, missing)
	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("cannot write config: %w", err)
	}

	printSection("Vendor Import")
	for _, e := range missing {
		printOK(e.Name, fmt.Sprintf("added to axon.yaml (dest=%s repo=%s)", e.Dest, e.Repo))
	}
	printBullet("Run 'axon vendor sync' to mirror the newly added entries.")

	return nil
}

// selectMissingByName returns the single missing entry matching name, or an
// error. Since missing already excludes entries present in the local config,
// a lookup failure means the name is either absent from the Hub registry or
// already configured locally — both are covered by one message.
func selectMissingByName(missing []vendor.RegistryEntry, name string) ([]vendor.RegistryEntry, error) {
	for _, e := range missing {
		if e.Name == name {
			return []vendor.RegistryEntry{e}, nil
		}
	}
	return nil, fmt.Errorf("no vendor named %q to import (either not in the Hub registry, or already configured locally)", name)
}

// appendVendorsFromRegistry converts registry entries into config.Vendor and
// appends them after local's existing entries, leaving local's order intact.
func appendVendorsFromRegistry(local []config.Vendor, entries []vendor.RegistryEntry) []config.Vendor {
	out := make([]config.Vendor, len(local), len(local)+len(entries))
	copy(out, local)
	for _, e := range entries {
		out = append(out, config.Vendor{Name: e.Name, Repo: e.Repo, Subdir: e.Subdir, Dest: e.Dest, Ref: e.Ref})
	}
	return out
}
