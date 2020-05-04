package testing

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func Test(e Environment) {
	context(e.Focus, fmt.Sprintf("given a environment manifest: %s", e.manifest()), func() {
		var (
			v       *vault
			g       *genesis
			workDir string
			logger  *log.Logger
		)

		BeforeEach(func() {
			var err error
			workDir, err = ioutil.TempDir(os.TempDir(), "*-salvation-home")
			Expect(err).ToNot(HaveOccurred())

			logger = log.New(GinkgoWriter, fmt.Sprintf("deployment(%s) ", e.Name), 0)

			v = newVault(e.vaultStub(), workDir, logger)
			v.Start()

			g = newGenesis(e, workDir, logger)
		})

		It(fmt.Sprintf("renders a manifest which matches: %s", e.result()), func() {
			manifest := g.Manifest()
			if _, err := os.Stat(e.result()); os.IsNotExist(err) {
				logger.Printf("creating new result file: %s", e.result())
				err = os.MkdirAll(filepath.Dir(e.result()), 0755)
				Expect(err).ToNot(HaveOccurred())
				err = ioutil.WriteFile(e.result(), manifest, 0644)
				Expect(err).ToNot(HaveOccurred())
			}
			result, err := ioutil.ReadFile(e.result())
			Expect(err).ToNot(HaveOccurred())

			Expect(manifest).To(MatchYAML(result))
		})

		AfterEach(func() {
			v.Stop()
			err := os.RemoveAll(workDir)
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
