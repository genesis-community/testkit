package spec_test

import (
	"path/filepath"
	"runtime"

	. "github.com/genesis-community/salvation/testing"
	. "github.com/onsi/ginkgo"
)

var _ = Describe("Interal Kit", func() {
	BeforeSuite(func() {
		_, filename, _, _ := runtime.Caller(0)
		KitDir, _ = filepath.Abs(filepath.Join(filepath.Dir(filename), "../"))
	})

	Test(Environment{
		Name:        "baseline",
		CloudConfig: "aws",
	})
	Test(Environment{
		Name:        "openvpn",
		CloudConfig: "aws",
	})
})
