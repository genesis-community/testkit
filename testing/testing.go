package testing

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func Test(d Deployment) {
	context(d.Focus, fmt.Sprintf("given a deployment manifest: %s", d.manifest()), func() {
		var (
			v       *vault
			homeDir string
			logger  *log.Logger
		)

		BeforeEach(func() {
			var err error
			homeDir, err = ioutil.TempDir(os.TempDir(), "*-salvation-home")
			Expect(err).ToNot(HaveOccurred())

			logger = log.New(GinkgoWriter, fmt.Sprintf("deployment: %s", d.Name), 0)

			v = newVault(d.vaultStub(), homeDir, logger)
			v.Start()
		})

		It(fmt.Sprintf("renders a manifest which matches: %", d.result()), func() {

		})

		AfterEach(func() {
			v.Stop()
			err := os.Remove(homeDir)
			Expect(err).ToNot(HaveOccurred())
		})
	})
}

func context(focus bool, description string, what func()) {
	if focus {
		FContext(description, what)
	} else {
		Context(description, what)
	}
}
