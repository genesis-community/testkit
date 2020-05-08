package testing

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"

	. "github.com/onsi/gomega"
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

func (b *bosh) Interpolate(manifest []byte, boshVars []byte) []byte {
	m := writeTmpFile(manifest)
	defer os.Remove(m)
	v := writeTmpFile(boshVars)
	defer os.Remove(v)

	cmd := b.bosh("int", m, "--vars-file", v, "--var-errs")

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Run()

	if cmd.ProcessState.ExitCode() != 0 {
		Expect("failed to interpolate bosh manifest").To(BeNil())
	}
	return buf.Bytes()
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
