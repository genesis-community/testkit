package testing

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v3"
)

type bosh struct {
	environment Environment
	workDir     string
	logger      *log.Logger
}

func newBosh(environment Environment, workDir string, logger *log.Logger) *bosh {
	b := bosh{
		environment: environment,
		workDir:     workDir,
		logger:      logger,
	}

	return &b
}

func (b *bosh) Interpolate(manifest []byte, boshVars []byte, credhubVars string, credhubStub string) []byte {
	m := writeTmpFile(manifest)
	defer os.Remove(m)
	v := writeTmpFile(boshVars)
	defer os.Remove(v)

	args := []string{
		"int", m, "--vars-file", v, "--var-errs", "--var-errs-unused",
	}

	if _, err := os.Stat(credhubVars); err == nil {
		args = append(args, "--vars-file", credhubVars)
	}

	if _, err := os.Stat(credhubStub); err == nil {
		args = append(args, "--vars-file", credhubStub)
	}

	return b.bosh(args...).Run(nil)
}

func (b *bosh) GenerateCredhubStub(manifest []byte, boshVars []byte) []byte {
	b.logger.Println(string(manifest))
	m := writeTmpFile(manifest)
	defer os.Remove(m)
	v := writeTmpFile(boshVars)
	defer os.Remove(v)
	cs := writeTmpFile([]byte("{}"))
	defer os.Remove(cs)

	b.bosh("int", m, "--vars-file", v, "--vars-store", cs, "--var-errs").Run(nil)

	creds, err := ioutil.ReadFile(cs)
	Expect(err).ToNot(HaveOccurred())
	return stubCredhubValues(creds)
}

func (b *bosh) bosh(arg ...string) *Cmd {
	return NewCmd("bosh", arg, []string{
		fmt.Sprintf("HOME=%s", b.workDir),
	}, b.logger)
}

func writeTmpFile(data []byte) string {
	tmpfile, err := ioutil.TempFile("", "testkit-bosh-tmp-file")
	Expect(err).ToNot(HaveOccurred())
	err = ioutil.WriteFile(tmpfile.Name(), data, 0644)
	return tmpfile.Name()
}

func stubCredhubValues(in []byte) []byte {
	input := make(map[string]interface{})
	err := yaml.Unmarshal(in, &input)
	Expect(err).ToNot(HaveOccurred())

	return jq{
		query: `
		   with_entries(.key as $p |
		     if (.value | type) == "string" then
			.value = "<!{credhub}:\($p)!>"
		     else
			.value = (.value | with_entries(.value = "<!{credhub}:\($p).\(.key)!>"))
		     end)`,
	}.Run(input)
}
