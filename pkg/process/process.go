package process

import (
	"io/ioutil"
	"os/exec"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
)

type Process struct {
	Logger log.Logger
}

func (p *Process) Fork(cmd *exec.Cmd) ([]byte, []byte, error) {
	debug := level.Debug(log.With(p.Logger, "struct", "patcher", "handler", "fork"))

	var stdout, stderr []byte
	stdoutReader, err := cmd.StdoutPipe()
	if err != nil {
		return stdout, stderr, errors.Wrapf(err, "pipe stdout")
	}
	stderrReader, err := cmd.StderrPipe()
	if err != nil {
		return stdout, stderr, errors.Wrapf(err, "pipe stderr")
	}

	debug.Log("event", "cmd.start")
	err = cmd.Start()
	if err != nil {
		return stdout, stderr, errors.Wrap(err, "start cmd")
	}
	debug.Log("event", "cmd.started")

	stdout, err = ioutil.ReadAll(stdoutReader)
	if err != nil {
		debug.Log("event", "stdout.read.fail", "err", err)
		return stdout, stderr, errors.Wrap(err, "read stdout")
	}
	debug.Log("event", "stdout.read", "value", string(stdout))

	stderr, err = ioutil.ReadAll(stderrReader)
	if err != nil {
		debug.Log("event", "stderr.read.fail", "err", err)
		return stdout, stderr, errors.Wrap(err, "read stderr")
	}
	debug.Log("event", "stderr.read", "value", string(stderr))

	debug.Log("event", "cmd.wait")
	err = cmd.Wait()
	debug.Log("event", "cmd.waited")

	debug.Log("event", "cmd.streams.read.done")

	return stdout, stderr, err
}
