package spec_test

import (
	"path/filepath"
	"runtime"

	. "github.com/genesis-community/testkit/testing"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Interal Kit", func() {
	BeforeSuite(func() {
		_, filename, _, _ := runtime.Caller(0)
		KitDir, _ = filepath.Abs(filepath.Join(filepath.Dir(filename), "../"))
	})

	Test(Environment{
		Name:          "baseline",
		CloudConfig:   "aws",
		RuntimeConfig: "dns",
	})
	Test(Environment{
		Name:        "baseline",
		CloudConfig: "aws",
	})
	Test(Environment{
		Name:        "openvpn",
		CloudConfig: "aws",
	})
	Test(Environment{
		Name:        "provided-user",
		CloudConfig: "aws",
	})
	Test(Environment{
		Name:        "detect-cpi",
		CloudConfig: "aws",
		CPI:         "aws",
	})
	Test(Environment{
		Name:        "bosh-variables",
		CloudConfig: "aws",
	})
	Test(Environment{
		Name:        "credhub",
		CloudConfig: "aws",
	})
	Test(Environment{
		Name:        "test-exodus",
		CloudConfig: "aws",
		Exodus:      "old-version",
	})
	Test(Environment{
		Name:        "blueprint-error",
		CloudConfig: "aws",
		OutputMatchers: OutputMatchers{
			GenesisAddSecrets: ContainSubstring("this-does-not-exist"),
			GenesisCheck:      ContainSubstring("this-does-not-exist"),
			GenesisManifest:   ContainSubstring("this-does-not-exist"),
		},
	})
})
