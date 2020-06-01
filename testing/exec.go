package testing

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"syscall"

	"github.com/onsi/gomega"
	gtypes "github.com/onsi/gomega/types"

	"github.com/egymgmbh/go-prefix-writer/prefixer"
)

type Cmd struct {
	stderr   *bytes.Buffer
	stdout   *bytes.Buffer
	combined *bytes.Buffer
	writer   *io.Writer
	command  string
	cmd      *exec.Cmd
}

func NewCmd(c string, arg []string, env []string, logger *log.Logger) *Cmd {
	cmd := exec.Command(c, arg...)
	cmd.Env = append(os.Environ(), env...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	command := fmt.Sprintf("%s %s", c, arg[0])
	prefix := fmt.Sprintf("[%s-%s] ", c, arg[0])
	prefixWriter := prefixer.New(logger.Writer(),
		func() string { return prefix })
	outBuff := bytes.NewBuffer([]byte{})
	errBuff := bytes.NewBuffer([]byte{})
	combinedBuff := bytes.NewBuffer([]byte{})
	outWriter := io.MultiWriter(combinedBuff,
		io.MultiWriter(outBuff, prefixWriter))
	errWriter := io.MultiWriter(combinedBuff,
		io.MultiWriter(errBuff, prefixWriter))
	cmd.Stdout = outWriter
	cmd.Stderr = errWriter
	return &Cmd{
		command: command, cmd: cmd,
		stdout: outBuff, stderr: errBuff,
		combined: combinedBuff,
	}
}

func (c *Cmd) Run(matcher gtypes.GomegaMatcher) []byte {
	if matcher == nil {
		matcher = gomega.HaveSuffix(commandFooter(c.command, 0))
	}
	fmt.Fprintf(c.combined, "running: %+v\n", c.cmd)
	err := c.cmd.Run()
	code := c.cmd.ProcessState.ExitCode()
	fmt.Fprintf(c.combined, commandFooter(c.command, code))
	if err != nil {
		fmt.Fprintf(c.combined, "got error: %s", err)
	}
	gomega.Expect(c.combined.String()).To(matcher)
	return c.stdout.Bytes()
}

func (c *Cmd) RunWithoutMatcher() ([]byte, int) {
	err := c.cmd.Run()
	code := c.cmd.ProcessState.ExitCode()
	fmt.Fprintf(c.combined, commandFooter(c.command, code))
	if err != nil {
		fmt.Fprintf(c.combined, "got error: %s", err)
	}
	return c.stdout.Bytes(), code
}

func commandFooter(command string, code int) string {
	return fmt.Sprintf("process: '%s' exited with: %d\n", command, code)
}
