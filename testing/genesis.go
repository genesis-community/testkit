package testing

import (
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
		"kit", "genesis", "compilation"}
	pruneCreateEnvKeys = []string{"resource_pools", "vm_types",
		"disk_pools", "disk_types", "networks", "azs", "vm_extensions"}
	pruneExodusKeys = []string{"version", "dated", "deployer", "kit_name",
		"kit_version", "vault_base", "kit_is_dev", "upgarding"}
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
	Name     string                 `yaml:"name"`
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
	g.genesis("init",
		"--link-dev-kit", KitDir,
		"--vault", currentVault,
		"--cwd", g.workDir,
		"--directory", "deployments",
		g.kit().Name,
	).Run(nil)

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
	args = append(args, g.configsArgs()...)
	args = append(args, g.environment.Name)
	g.genesis(args...).Run(g.environment.OutputMatchers.GenesisCheck)
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
	filtered := map[string]interface{}{}
	for k, v := range in {
		if k == "exodus" {
			e, ok := v.(map[string]interface{})
			if !ok {
				continue
			}
			exodus := map[string]interface{}{}
			for ek, ev := range e {
				if !contains(pruneExodusKeys, ek) {
					exodus[ek] = ev
				}
			}
			if len(exodus) != 0 {
				filtered[k] = exodus
			}
			continue
		}
		if !contains(allKeys, k) {
			filtered[k] = v
		}
	}
	out, err := yaml.Marshal(filtered)
	Expect(err).ToNot(HaveOccurred())
	return out
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
	args = append(args, g.configsArgs()...)
	args = append(args, g.environment.Name)
	return g.genesis(args...).Run(g.environment.OutputMatchers.GenesisManifest)
}

func (g *genesis) needsBoshCreatEnv() bool {
	for _, f := range g.env().Kit.Features {
		if f == "proto" {
			return true
		}
	}
	return false
}

func (g *genesis) ExodusStub() []byte {
	if g.environment.exodusStub() == "" {
		return []byte(`{}`)
	}

	in := map[string]interface{}{}
	raw, err := ioutil.ReadFile(g.environment.exodusStub())
	Expect(err).ToNot(HaveOccurred())
	err = yaml.Unmarshal(raw, &in)
	return jq{
		query:     `[{ key: "\($base)", value: .}] | from_entries`,
		variables: []string{"$base"},
		values:    []interface{}{g.exodusBase()},
	}.Run(in)
}

func (g *genesis) ProvidedSecretsStub() []byte {
	args := []string{
		"check-secrets",
		"--no-color",
		"-lm", "-v",
		"--cwd", g.deploymentsDir(),
		g.environment.Name,
		"type=provided",
	}
	buf, code := g.genesis(args...).RunWithoutMatcher()
	if code == 0 {
		return []byte(`{}`)
	}

	return jq{
		query: `split("\n") | map(select(startswith("  [")))
			| map(split(" ")[3])
			    | map(split(":") | {
			         key: "\($base)/\(.[0])",
			         value: ([{
                                     key: .[1],
		                     value: "<!{meta.vault}/\(.[0]):\(.[1])!>"
                                 }] | from_entries )
                            })
			| map([.] | from_entries)
			| reduce .[] as $item ({}; . * $item)`,
		variables: []string{"$base"},
		values:    []interface{}{g.base()},
	}.Run(string(buf))
}

func (g *genesis) AddSecrets() {
	args := []string{
		"add-secrets",
		"--cwd", g.deploymentsDir(),
		g.environment.Name,
	}
	g.genesis(args...).Run(g.environment.OutputMatchers.GenesisAddSecrets)
}

func (g *genesis) base() string {
	return fmt.Sprintf("secret/%s/%s",
		strings.Replace(g.environment.Name, "-", "/", -1),
		g.kit().Name)
}

func (g *genesis) exodusBase() string {
	return fmt.Sprintf("secret/exodus/%s/%s",
		g.environment.Name, g.kit().Name)
}

func (g *genesis) git(arg ...string) *exec.Cmd {
	cmd := exec.Command("git", arg...)
	cmd.Stdout = g.logger.Writer()
	cmd.Stderr = g.logger.Writer()
	cmd.Env = append(os.Environ(), fmt.Sprintf("HOME=%s", g.workDir))
	return cmd
}

func (g *genesis) configsArgs() []string {
	args := make([]string, 0)
	if g.environment.cloudConfigManifest() != "" {
		args = append(args, "-c", fmt.Sprintf("cloud=%s", g.environment.cloudConfigManifest()))
	} else {
		args = append(args, "-c", fmt.Sprintf("cloud=%s", g.configStubPath()))
	}
	if g.environment.runtimeConfigManifest() != "" {
		args = append(args, "-c", fmt.Sprintf("runtime=%s", g.environment.runtimeConfigManifest()))
	} else {
		args = append(args, "-c", fmt.Sprintf("runtime=%s", g.configStubPath()))
	}
	return args
}

func (g *genesis) configStubPath() string {
	dst := filepath.Join(g.workDir, "config-stub.yml")
	_, err := os.Stat(dst)
	if os.IsNotExist(err) {
		err := ioutil.WriteFile(dst, []byte(`{}`), 0644)
		Expect(err).ToNot(HaveOccurred())
	}
	return dst
}

func (g *genesis) genesis(arg ...string) *Cmd {
	return NewCmd("genesis", arg, []string{
		fmt.Sprintf("GENESIS_TESTING_BOSH_CPI=%s", g.environment.CPI),
		"GENESIS_TESTING_CHECK_SECRETS_PRESENCE_ONLY=true",
		"GENESIS_TESTING=yes",
		fmt.Sprintf("GENESIS_BOSH_VERIFIED=%s", g.environment.Name),
		fmt.Sprintf("HOME=%s", g.workDir),
	}, g.logger)
}

func copyFile(src string, dst string) {
	data, err := ioutil.ReadFile(src)
	Expect(err).ToNot(HaveOccurred())
	err = ioutil.WriteFile(dst, data, 0644)
	Expect(err).ToNot(HaveOccurred())
}

func contains(in []string, key string) bool {
	for _, fk := range in {
		if fk == key {
			return true
		}
	}
	return false
}
