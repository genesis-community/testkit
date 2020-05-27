package testing

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	dyff "github.com/genesis-community/testkit/dyff_yaml_matcher"
)

func Test(e Environment) {
	context(e.Focus, fmt.Sprintf("given a environment manifest: %s", e.manifest()), func() {
		var (
			v       *vault
			g       *genesis
			b       *bosh
			workDir string
			logger  *log.Logger
		)

		BeforeEach(func() {
			var err error
			workDir, err = ioutil.TempDir(os.TempDir(), "*-testkit-home")
			Expect(err).ToNot(HaveOccurred())

			logger = log.New(GinkgoWriter, fmt.Sprintf("deployment(%s) ", e.Name), 0)

			v = newVault(workDir, logger)
			v.Start()

			g = newGenesis(e, workDir, logger)

			b = newBosh(e, workDir, logger)

			createVaultCacheIffMissing(e.vaultCache(), v, g, logger)

			cache, err := os.Open(e.vaultCache())
			Expect(err).ToNot(HaveOccurred())
			defer cache.Close()
			v.Import(cache)
		})

		It(fmt.Sprintf("renders a manifest which matches: %s", e.result()), func() {
			g.Check()

			manifestResult := g.Manifest()
			createCredhubStubIffMissing(e.credhubStub(), b,
				manifestResult, logger)
			manifest := b.Interpolate(manifestResult.manifest,
				manifestResult.boshVariables, e.credhubStub())

			createResultIfMissingForManifest(e.result(), manifest, logger)
			result, err := ioutil.ReadFile(e.result())
			Expect(err).ToNot(HaveOccurred())

			Expect(manifest).To(dyff.MatchYAML(result))
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

func createVaultCacheIffMissing(vaultCache string, v *vault, g *genesis, logger *log.Logger) {
	if _, err := os.Stat(vaultCache); os.IsNotExist(err) {
		if vaultProvided := g.ProvidedSecretsStub(); string(vaultProvided) != "{}" {
			logger.Printf("detected provided secrets importing:\n%s", vaultProvided)
			v.Import(bytes.NewBuffer(vaultProvided))
		}

		logger.Printf("adding secrets to stub: %s", vaultCache)
		g.AddSecrets()
		createParentDirsAndWriteFile(vaultCache, v.Export(g.base()))
	}
}

func createCredhubStubIffMissing(credhubStub string, b *bosh, m manifestResult, logger *log.Logger) {
	if _, err := os.Stat(credhubStub); os.IsNotExist(err) {
		logger.Printf("creating Credhub stub: %s", credhubStub)
		if m.credhub {
			createParentDirsAndWriteFile(credhubStub, b.GenerateCredhubStub(m.manifest, m.boshVariables))
		}
	}
}

func createParentDirsAndWriteFile(file string, content []byte) {
	err := os.MkdirAll(filepath.Dir(file), 0755)
	Expect(err).ToNot(HaveOccurred())
	err = ioutil.WriteFile(file, content, 0644)
	Expect(err).ToNot(HaveOccurred())
}
