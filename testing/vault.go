package testing

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

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

	Eventually(func() string {
		if _, err := os.Stat(filepath.Join(v.homeDir, ".saferc")); os.IsNotExist(err) {
			return ".saferc has not been created yet; vault is not yet running"
		} else {
			return ".saferc created"
		}
	}).Should(Equal(".saferc created"))

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
