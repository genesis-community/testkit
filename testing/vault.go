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
	"sync/atomic"
	"syscall"

	"gopkg.in/yaml.v2"

	"github.com/onsi/ginkgo/config"
	. "github.com/onsi/gomega"
)

var port *int32

func init() {
	p := int32(0)
	port = &p
}

type vault struct {
	homeDir string
	logger  *log.Logger
	server  *exec.Cmd
	port    int32
}

func newVault(homeDir string, logger *log.Logger) *vault {
	atomic.CompareAndSwapInt32(port, 0, 8200+(int32(config.GinkgoConfig.ParallelNode)*10))
	p := atomic.AddInt32(port, 1)
	v := vault{
		homeDir: homeDir,
		logger:  logger,
		port:    p,
	}
	v.server = v.safe("local", "--memory", "--port",
		fmt.Sprintf("%d", p))
	return &v
}

func (v *vault) Start() {
	v.logger.Printf("Running on node: %d", config.GinkgoConfig.ParallelNode)
	v.logger.Printf("Starting vault at port: %d", v.port)
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
	Expect(cmd.ProcessState.ExitCode()).To(Equal(0))
}

func (v *vault) Export(path string) []byte {
	v.logger.Println("Exporting vault stub")
	cmd := v.safe("export", "/")
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Run()
	Expect(cmd.ProcessState.ExitCode()).To(Equal(0))

	return stubValues(buf.Bytes(), path)
}

func (v *vault) Stop() {
	syscall.Kill(-v.server.Process.Pid, syscall.SIGKILL)
}

func (v *vault) safe(arg ...string) *exec.Cmd {
	cmd := exec.Command("safe", arg...)
	cmd.Stdout = v.logger.Writer()
	cmd.Stderr = v.logger.Writer()
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("HOME=%s", v.homeDir),
	)
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
