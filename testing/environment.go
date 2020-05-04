package testing

import (
	"fmt"
	"path/filepath"
)

type Environment struct {
	Name        string
	CloudConfig string
	Focus       bool
}

func (e Environment) manifest() string {
	return e.filePathForDir("deployments")
}

func (e Environment) cloudConfigManifest() string {
	if e.CloudConfig == "" {
		return ""
	} else {
		return filepath.Join(KitDir, "spec", "cloud_configs",
			fmt.Sprintf("%s.yml", e.CloudConfig))
	}
}

func (e Environment) vaultStub() string {
	return e.filePathForDir("vault_stubs")
}

func (e Environment) result() string {
	return e.filePathForDir("results")
}

func (e Environment) filePathForDir(dir string) string {
	return filepath.Join(KitDir, "spec", dir,
		fmt.Sprintf("%s.yml", e.Name))
}
