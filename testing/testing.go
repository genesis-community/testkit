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

			v = newVault(workDir, logger)
			v.Start()

			g = newGenesis(e, workDir, logger)

			createVaultCacheIffMissing(e.vaultCache(), e.vaultProvided(), v, g, logger)

			v.Import(e.vaultCache())
		})

		It(fmt.Sprintf("renders a manifest which matches: %s", e.result()), func() {
			manifest := g.Manifest()

			createResultIfMissingForManifest(e.result(), manifest, logger)
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

func createResultIfMissingForManifest(result string, manifest []byte, logger *log.Logger) {
	if _, err := os.Stat(result); os.IsNotExist(err) {
		logger.Printf("creating new result file: %s", result)
		createParentDirsAndWriteFile(result, manifest)
	}
}

func createVaultCacheIffMissing(vaultCache string, vaultProvided string, v *vault, g *genesis, logger *log.Logger) {
	if _, err := os.Stat(vaultProvided); err == nil {
		logger.Printf("detected: %s loading before adding secrets", vaultProvided)
		v.Import(vaultProvided)
	}
	if _, err := os.Stat(vaultCache); os.IsNotExist(err) {
		logger.Printf("adding secrets to stub: %s", vaultCache)
		g.AddSecrets()
		createParentDirsAndWriteFile(vaultCache, v.Export())
	}
}

func createParentDirsAndWriteFile(file string, content []byte) {
	err := os.MkdirAll(filepath.Dir(file), 0755)
	Expect(err).ToNot(HaveOccurred())
	err = ioutil.WriteFile(file, content, 0644)
	Expect(err).ToNot(HaveOccurred())
}
