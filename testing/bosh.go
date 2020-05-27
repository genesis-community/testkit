package testing

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"

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

func (b *bosh) Interpolate(manifest []byte, boshVars []byte, credhubStub string) []byte {
	m := writeTmpFile(manifest)
	defer os.Remove(m)
	v := writeTmpFile(boshVars)
	defer os.Remove(v)

	args := []string{
		"int", m, "--vars-file", v, "--var-errs", "--var-errs-unused",
	}

	if _, err := os.Stat(credhubStub); err == nil {
		args = append(args, "--vars-file", credhubStub)
	}

	cmd := b.bosh(args...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Run()

	Expect(cmd.ProcessState.ExitCode()).To(Equal(0))
	return buf.Bytes()
}

func (b *bosh) GenerateCredhubStub(manifest []byte, boshVars []byte) []byte {
	b.logger.Println(string(manifest))
	m := writeTmpFile(manifest)
	defer os.Remove(m)
	v := writeTmpFile(boshVars)
	defer os.Remove(v)
	cs := writeTmpFile([]byte("{}"))
	defer os.Remove(cs)

	cmd := b.bosh("int", m, "--vars-file", v, "--vars-store", cs, "--var-errs")
	cmd.Run()
	Expect(cmd.ProcessState.ExitCode()).To(Equal(0))

	creds, err := ioutil.ReadFile(cs)
	Expect(err).ToNot(HaveOccurred())
	return stubCredhubValues(creds)
}

func (b *bosh) bosh(arg ...string) *exec.Cmd {
	cmd := exec.Command("bosh", arg...)
	cmd.Stdout = b.logger.Writer()
	cmd.Stderr = b.logger.Writer()
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("HOME=%s", b.workDir),
	)
	return cmd
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
