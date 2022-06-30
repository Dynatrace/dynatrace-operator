package standalone

import (
	"os"
	"path/filepath"

	"github.com/spf13/afero"
)

type Runner struct {
	fs          afero.Fs
	env         *environment
	agentSetup  *oneAgentSetup
	ingestSetup *dataIngestSetup
}

func NewRunner(fs afero.Fs) (runner *Runner, resultedError error) {
	log.Info("creating standalone runner")
	env, err := newEnv()
	if err != nil {
		log.Info("failed to read in the environment")
		return nil, err
	}
	defer env.consumeErrorIfNecessary(&resultedError)

	var agentSetup *oneAgentSetup
	if env.OneAgentInjected {
		agentSetup, err = newOneagentSetup(fs, env)
		if err != nil {
			return nil, err
		}
	}

	var ingestSetup *dataIngestSetup
	if env.DataIngestInjected {
		ingestSetup = newDataIngestSetup(fs, env)
	}

	log.Info("standalone runner created successfully")
	return &Runner{
		fs:          fs,
		env:         env,
		agentSetup:  agentSetup,
		ingestSetup: ingestSetup,
	}, nil
}

func (runner *Runner) Run() (resultedError error) {
	log.Info("standalone agent init started")
	defer runner.env.consumeErrorIfNecessary(&resultedError)

	if runner.agentSetup != nil {
		err := runner.agentSetup.setup()
		if err != nil {
			return err
		}
	}

	if runner.ingestSetup != nil {
		ingestSetup := newDataIngestSetup(runner.fs, runner.env)
		err := ingestSetup.setup()
		if err != nil {
			return err
		}
	}
	return nil
}

func createConfFile(fs afero.Fs, path string, content string) error {
	err := fs.MkdirAll(filepath.Dir(path), onlyReadAllFileMode)
	if err != nil {
		return err
	}

	file, err := fs.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, onlyReadAllFileMode)
	if err != nil {
		return err
	}

	_, err = file.Write([]byte(content))
	if err != nil {
		return err
	}

	log.Info("created file", "filePath", path, "content", content)
	return nil
}
