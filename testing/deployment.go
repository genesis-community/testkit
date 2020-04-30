package testing

import (
	"fmt"
	"path/filepath"
)

type Deployment struct {
	Name  string
	Focus bool
}

func (d Deployment) manifest() string {
	return d.filePathForDir("deployments")
}

func (d Deployment) vaultStub() string {
	return d.filePathForDir("vault_stubs")
}

func (d Deployment) result() string {
	return d.filePathForDir("results")
}

func (d Deployment) filePathForDir(dir string) string {
	return filepath.Join(SpecRoot, dir,
		fmt.Sprintf("%s.yml", d.Name))
}
