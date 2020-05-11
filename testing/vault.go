package testing

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"gopkg.in/yaml.v2"

	"github.com/gofrs/flock"

	. "github.com/onsi/gomega"
)

var vaultStartLock *flock.Flock

func init() {
	vaultStartLock = flock.NewFlock(filepath.Join(KitDir, ".testing_vault_start_lock"))
}

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
	vaultStartLock.Lock()
	defer vaultStartLock.Unlock()
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

func (v *vault) Import(i io.Reader) {
	stub, err := ioutil.ReadAll(i)
	Expect(err).ToNot(HaveOccurred())

	tmp := map[string]map[string]string{}
	err = yaml.Unmarshal(stub, &tmp)
	Expect(err).ToNot(HaveOccurred())

	stubJson, err := json.Marshal(tmp)
	Expect(err).ToNot(HaveOccurred())

	cmd := v.safe("import")
	cmd.Stdin = bytes.NewBuffer(stubJson)
	cmd.Run()
	if cmd.ProcessState.ExitCode() != 0 {
		Expect(fmt.Sprintf("failed to import: %s into vault", stub)).To(BeNil())
	}
}

func (v *vault) Export(path string) []byte {
	v.logger.Println("Exporting vault stub")
	cmd := v.safe("export", "/")
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Run()
	if cmd.ProcessState.ExitCode() != 0 {
		Expect("failed to export vault").To(BeNil())
	}

	return stubValues(buf.Bytes(), path)
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

func stubValues(in []byte, path string) []byte {
	input := make(map[string]interface{})
	err := json.Unmarshal(in, &input)
	Expect(err).ToNot(HaveOccurred())

	// Replace values of export with string containing the full vault path
	return jq{
		query: `
		  to_entries
		    | map(
		      .key as $p |
		      .value = (
			.value | to_entries
			  | map(.value = "<!\($p | sub($base; "{meta.vault}")):\(.key)!>")
			| from_entries
		      )
		    )
		  | from_entries`,
		variables: []string{"$base"},
		values:    []interface{}{path},
	}.Run(input)
}
