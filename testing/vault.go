package testing

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/itchyny/gojq"

	"gopkg.in/yaml.v2"

	. "github.com/onsi/gomega"
)

type vault struct {
	homeDir string
	logger  *log.Logger
	server  *exec.Cmd
}

func newVault(homeDir string, logger *log.Logger) *vault {
	v := vault{
		homeDir: homeDir,
		logger:  logger,
	}
	v.server = v.safe("local", "--memory")
	return &v
}

func (v *vault) Start() {
	v.logger.Println("Starting vault")
	err := v.server.Start()
	Expect(err).ToNot(HaveOccurred())

	Eventually(func() int {
		v.logger.Println("Waiting for vault to be ready..")
		s := v.safe("get", "secret/handshake")
		s.Stdout = ioutil.Discard
		s.Stderr = ioutil.Discard
		s.Run()
		return s.ProcessState.ExitCode()
	}, "2s", "100ms").Should(Equal(0))

}

func (v *vault) Import(s string) {
	v.logger.Println("Importing vault stub")
	stub, err := os.Open(s)
	defer stub.Close()
	Expect(err).ToNot(HaveOccurred())

	cmd := v.safe("import")
	cmd.Stdin = stub
	cmd.Run()
	if cmd.ProcessState.ExitCode() != 0 {
		Expect(fmt.Sprintf("failed to import: %s into vault", stub)).To(BeNil())
	}
}

func (v *vault) Export() []byte {
	v.logger.Println("Exporting vault stub")
	cmd := v.safe("export", "/")
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Run()
	if cmd.ProcessState.ExitCode() != 0 {
		Expect("failed to export vault").To(BeNil())
	}

	return stubValues(buf.Bytes())
}

func (v *vault) Stop() {
	err := v.server.Process.Kill()
	Expect(err).ToNot(HaveOccurred())
}

func (v *vault) safe(arg ...string) *exec.Cmd {
	cmd := exec.Command("safe", arg...)
	cmd.Stdout = v.logger.Writer()
	cmd.Stderr = v.logger.Writer()
	cmd.Env = append(os.Environ(), fmt.Sprintf("HOME=%s", v.homeDir))
	return cmd
}

func GetCurrentVaultTarget(homeDir string) string {
	config := struct {
		Current string `yaml:"current"`
	}{}
	raw, err := ioutil.ReadFile(filepath.Join(homeDir, ".saferc"))
	Expect(err).ToNot(HaveOccurred())
	err = yaml.Unmarshal(raw, &config)
	Expect(err).ToNot(HaveOccurred())
	return config.Current
}

func stubValues(in []byte) []byte {
	input := make(map[string]interface{})
	err := json.Unmarshal(in, &input)
	Expect(err).ToNot(HaveOccurred())

	// Replace values of export with string containing the full vault path
	query, err := gojq.Parse(`
          to_entries
            | map(
              .key as $p |
              .value = (
                .value | to_entries
                  | map(.value = "STUB \($p):\(.key)")
                | from_entries
              )
            )
          | from_entries`,
	)
	Expect(err).ToNot(HaveOccurred())

	var buf bytes.Buffer

	iter := query.Run(input)
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := v.(error); ok {
			Expect(err).ToNot(HaveOccurred())
		}
		out, err := json.MarshalIndent(v, "", "  ")
		Expect(err).ToNot(HaveOccurred())
		_, err = buf.Write(out)
		Expect(err).ToNot(HaveOccurred())
	}
	return buf.Bytes()
}
