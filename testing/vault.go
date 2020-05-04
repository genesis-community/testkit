package testing

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"gopkg.in/yaml.v2"

	. "github.com/onsi/gomega"
)

type vault struct {
	stub    string
	homeDir string
	logger  *log.Logger
	server  *exec.Cmd
}

func newVault(stub string, homeDir string, logger *log.Logger) *vault {
	v := vault{
		stub:    stub,
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

	v.logger.Println("Importing vault stub")
	stub, err := os.Open(v.stub)
	Expect(err).ToNot(HaveOccurred())

	i := v.safe("import")
	i.Stdin = stub
	err = i.Run()
	Expect(err).ToNot(HaveOccurred())
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
