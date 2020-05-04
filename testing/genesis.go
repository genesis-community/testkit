package testing

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	. "github.com/onsi/gomega"
)

type genesis struct {
	environment Environment
	workDir     string
	logger      *log.Logger
}

func (g *genesis) deploymentsDir() string {
	return filepath.Join(g.workDir, "deployments")
}

func newGenesis(environment Environment, workDir string, logger *log.Logger) *genesis {
	g := genesis{
		environment: environment,
		workDir:     workDir,
		logger:      logger,
	}

	g.init()
	return &g
}

func (g *genesis) init() {
	g.logger.Println(fmt.Sprintf("initializing genesis workdir: %s", g.workDir))
	err := g.git("config", "--global", "user.name", "Ci Runner").Run()
	Expect(err).ToNot(HaveOccurred())
	err = g.git("config", "--global", "user.email", "ci@starkandwayne.com").Run()
	Expect(err).ToNot(HaveOccurred())

	currentVault := GetCurrentVaultTarget(g.workDir)
	err = g.genesis("init",
		"--link-dev-kit", KitDir,
		"--vault", currentVault,
		"--cwd", g.workDir,
		"--directory", "deployments",
	).Run()
	Expect(err).ToNot(HaveOccurred())

	g.logger.Println(fmt.Sprintf("copying environment file %s into workdir: %s",
		g.environment.manifest(), g.deploymentsDir()))
	env := fmt.Sprintf("%s.yml", g.environment.Name)
	envFile := filepath.Join(g.deploymentsDir(), env)
	copyFile(g.environment.manifest(), envFile)
}

func (g *genesis) Manifest() []byte {
	g.logger.Println(fmt.Sprintf("generating manifest for: %s", g.environment.Name))
	args := []string{
		"manifest",
		"--cwd", g.deploymentsDir(),
		"--no-redact",
	}
	if g.environment.cloudConfigManifest() != "" {
		args = append(args, "--cloud-config", g.environment.cloudConfigManifest())
	}
	args = append(args, g.environment.Name)
	cmd := g.genesis(args...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Run()
	if cmd.ProcessState.ExitCode() != 0 {
		Expect("failed to render manifest").To(BeNil())
	}
	return buf.Bytes()
}

func (g *genesis) AddSecrets() {
	args := []string{
		"add-secrets",
		"--cwd", g.deploymentsDir(),
		g.environment.Name,
	}
	cmd := g.genesis(args...)
	cmd.Run()
	if cmd.ProcessState.ExitCode() != 0 {
		Expect("failed to add secrets to vault").To(BeNil())
	}
}

func (g *genesis) git(arg ...string) *exec.Cmd {
	cmd := exec.Command("git", arg...)
	cmd.Stdout = g.logger.Writer()
	cmd.Stderr = g.logger.Writer()
	cmd.Env = append(os.Environ(), fmt.Sprintf("HOME=%s", g.workDir))
	return cmd
}

func (g *genesis) genesis(arg ...string) *exec.Cmd {
	cmd := exec.Command("genesis", arg...)
	cmd.Stdout = g.logger.Writer()
	cmd.Stderr = g.logger.Writer()
	cmd.Env = append(os.Environ(), fmt.Sprintf("HOME=%s", g.workDir))
	return cmd
}

func copyFile(src string, dst string) {
	data, err := ioutil.ReadFile(src)
	Expect(err).ToNot(HaveOccurred())
	err = ioutil.WriteFile(dst, data, 0644)
	Expect(err).ToNot(HaveOccurred())
}
