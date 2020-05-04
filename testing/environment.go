package testing

import (
	"fmt"
	"path/filepath"
)

type Environment struct {
	Name            string
	CloudConfigName string
	Focus           bool
}

func (e Environment) manifest() string {
	return e.filePathForDir("deployments")
}

func (e Environment) cloudConfig() string {
	if e.CloudConfigName == "" {
		return ""
	} else {
		return filepath.Join(KitDir, "spec", "cloud_configs",
			fmt.Sprintf("%s.yml", e.CloudConfigName))
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
