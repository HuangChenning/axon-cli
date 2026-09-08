package vendor

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// RegistryFileName is the Hub-tracked file recording which vendor entries
// have been mirrored into this Hub. Because it lives inside the Hub, it
// travels to every machine via `axon sync`, so a machine that pulls Hub
// content mirrored by another machine can still discover which vendor it
// came from — even if that vendor was never added to this machine's own
// axon.yaml.
const RegistryFileName = ".axon-vendors.lock.yaml"

// RegistryEntry records one vendor's provenance for cross-machine discovery.
type RegistryEntry struct {
	Name   string `yaml:"name"`
	Repo   string `yaml:"repo"`
	Subdir string `yaml:"subdir"`
	Dest   string `yaml:"dest"`
	Ref    string `yaml:"ref,omitempty"`
}

type registryFile struct {
	Vendors []RegistryEntry `yaml:"vendors"`
}

// RegistryPath returns the absolute path to the vendor registry inside the Hub.
func RegistryPath(hubRoot string) string {
	return filepath.Join(hubRoot, RegistryFileName)
}

// ReadRegistry reads the Hub-tracked vendor registry. Returns a nil slice
// (not an error) when the file doesn't exist yet — e.g. a Hub that has never
// had `axon vendor sync` run against it.
func ReadRegistry(hubRoot string) ([]RegistryEntry, error) {
	data, err := os.ReadFile(RegistryPath(hubRoot))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading vendor registry: %w", err)
	}
	var f registryFile
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("invalid YAML in %s: %w", RegistryPath(hubRoot), err)
	}
	return f.Vendors, nil
}

// WriteRegistry overwrites the Hub-tracked vendor registry with entries.
func WriteRegistry(hubRoot string, entries []RegistryEntry) error {
	data, err := yaml.Marshal(registryFile{Vendors: entries})
	if err != nil {
		return fmt.Errorf("marshal vendor registry: %w", err)
	}
	if err := os.WriteFile(RegistryPath(hubRoot), data, 0o644); err != nil {
		return fmt.Errorf("writing vendor registry %s: %w", RegistryPath(hubRoot), err)
	}
	return nil
}
