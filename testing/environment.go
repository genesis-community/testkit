package testing

import (
	"fmt"
	"path/filepath"
)

type Environment struct {
	Name        string
	CloudConfig string
	Exodus      string
	CPI         string
	Focus       bool
}

func (e Environment) manifest() string {
	return e.filePathForDir("deployments", e.Name)
}

func (e Environment) cloudConfigManifest() string {
	return e.filePathForDir("cloud_configs", e.CloudConfig)
}

func (e Environment) exodusStub() string {
	return e.filePathForDir("exodus", e.Exodus)
}

func (e Environment) vaultCache() string {
	return e.filePathForDir("vault", e.Name)
}

func (e Environment) credhubStub() string {
	return e.filePathForDir("credhub", e.Name)
}

func (e Environment) result() string {
	return e.filePathForDir("results", e.Name)
}

func (e Environment) filePathForDir(dir string, name string) string {
	if name == "" {
		return ""
	}
	return filepath.Join(KitDir, "spec", dir,
		fmt.Sprintf("%s.yml", name))
}
