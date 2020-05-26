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
	"strings"

	. "github.com/onsi/gomega"

	// gojq only works with v3
	"gopkg.in/yaml.v3"
)

var (
	pruneKeys = []string{"meta", "pipeline", "params", "bosh-variables",
		"kit", "genesis", "exodus", "compilation"}
	pruneCreateEnvKeys = []string{"resource_pools", "vm_types",
		"disk_pools", "disk_types", "networks", "azs", "vm_extensions"}
)

type genesis struct {
	environment Environment
	workDir     string
	logger      *log.Logger
}

type manifestResult struct {
	manifest      []byte
	boshVariables []byte
	credhub       bool
}

type kit struct {
	Provided map[string]interface{} `yaml:"provided"`
}

type env struct {
	Kit struct {
		Features []string `yaml:"features`
	} `yaml:"kit"`
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

func (g *genesis) kit() kit {
	k := kit{}
	raw, err := ioutil.ReadFile(filepath.Join(KitDir, "kit.yml"))
	Expect(err).ToNot(HaveOccurred())
	err = yaml.Unmarshal(raw, &k)
	Expect(err).ToNot(HaveOccurred())
	return k
}

func (g *genesis) env() env {
	e := env{}
	raw, err := ioutil.ReadFile(g.environment.manifest())
	Expect(err).ToNot(HaveOccurred())
	err = yaml.Unmarshal(raw, &e)
	Expect(err).ToNot(HaveOccurred())
	return e
}

func (g *genesis) Check() {
	g.logger.Println(fmt.Sprintf("running genesis check for: %s", g.environment.Name))
	args := []string{
		"check",
		"--cwd", g.deploymentsDir(),
		"--no-manifest",
		"--no-stemcells",
	}
	if g.environment.cloudConfigManifest() != "" {
		args = append(args, "--cloud-config", g.environment.cloudConfigManifest())
	}
	args = append(args, g.environment.Name)
	cmd := g.genesis(args...)
	cmd.Run()
	Expect(cmd.ProcessState.ExitCode()).To(Equal(0))
}

func (g *genesis) Manifest() manifestResult {
	g.logger.Println(fmt.Sprintf("generating manifest for: %s", g.environment.Name))
	raw := g.rawManifest()

	boshVariables, credhub := extractBoshVariables(raw)
	return manifestResult{
		manifest:      pruneManifest(raw, g.needsBoshCreatEnv()),
		boshVariables: boshVariables,
		credhub:       credhub,
	}
}

func pruneManifest(raw []byte, needsBoshCreatEnv bool) []byte {
	in := map[string]interface{}{}
	err := yaml.Unmarshal(raw, &in)
	Expect(err).ToNot(HaveOccurred())
	allKeys := pruneKeys
	if !needsBoshCreatEnv {
		allKeys = append(allKeys, pruneCreateEnvKeys...)
	}
	keys, err := json.Marshal(allKeys)
	Expect(err).ToNot(HaveOccurred())
	return jq{
		query: `with_entries(
                          select([.key] | inside($keys|fromjson) | not)
                        )`,
		variables: []string{"$keys"},
		values:    []interface{}{string(keys)},
	}.Run(in)
}

func extractBoshVariables(raw []byte) ([]byte, bool) {
	bv := struct {
		Variables     []interface{}          `yaml:"variables",omitempty`
		BoshVariables map[string]interface{} `yaml:"bosh-variables",omitempty`
	}{}
	err := yaml.Unmarshal(raw, &bv)
	Expect(err).ToNot(HaveOccurred())

	bvo, err := yaml.Marshal(bv.BoshVariables)
	Expect(err).ToNot(HaveOccurred())

	return bvo, bv.Variables != nil
}

func (g *genesis) rawManifest() []byte {
	args := []string{
		"manifest",
		"--cwd", g.deploymentsDir(),
		"--no-redact",
		"--no-prune",
	}
	if g.environment.cloudConfigManifest() != "" {
		args = append(args, "--cloud-config", g.environment.cloudConfigManifest())
	}
	args = append(args, g.environment.Name)
	cmd := g.genesis(args...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Run()
	Expect(cmd.ProcessState.ExitCode()).To(Equal(0))
	return buf.Bytes()
}

func (g *genesis) needsBoshCreatEnv() bool {
	for _, f := range g.env().Kit.Features {
		if f == "proto" {
			return true
		}
	}
	return false
}

func (g *genesis) ProvidedSecretsStub() []byte {
	rawFeatures, err := json.Marshal(g.env().Kit.Features)
	Expect(err).ToNot(HaveOccurred())

	return jq{
		query: `with_entries(
                          select([.key] | inside($features|fromjson))
                        )
                        | reduce .[] as $item ({}; . * $item)
                        | with_entries(
                          .key as $p
                          | {
                            key: "\($base)/\(.key)",
                            value: .value.keys | with_entries(
                              .value = "<!{meta.vault}/\($p):\(.key)!>"
                            )
                          }
                        )`,
		variables: []string{"$base", "$features"},
		values:    []interface{}{g.base(), string(rawFeatures)},
	}.Run(g.kit().Provided)
}

func (g *genesis) AddSecrets() {
	args := []string{
		"add-secrets",
		"--cwd", g.deploymentsDir(),
		g.environment.Name,
	}
	cmd := g.genesis(args...)
	cmd.Run()
	Expect(cmd.ProcessState.ExitCode()).To(Equal(0))
}

func (g *genesis) base() string {
	return fmt.Sprintf("secret/%s/%s",
		strings.Replace(g.environment.Name, "-", "/", -1),
		filepath.Base(KitDir))
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
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("HOME=%s", g.workDir),
		fmt.Sprintf("GENESIS_TESTING_BOSH_CPI=%s", g.environment.CPI),
		"GENESIS_TESTING_CHECK_SECRETS_PRESENCE_ONLY=true",
	)
	return cmd
}

func copyFile(src string, dst string) {
	data, err := ioutil.ReadFile(src)
	Expect(err).ToNot(HaveOccurred())
	err = ioutil.WriteFile(dst, data, 0644)
	Expect(err).ToNot(HaveOccurred())
}
