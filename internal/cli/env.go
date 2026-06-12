package cli

import (
	"github.com/sim4gh/mkpdfs-cli/internal/config"
	"github.com/sim4gh/mkpdfs-cli/internal/envs"
)

func currentEnv() (envs.Env, error) {
	name := flagEnv
	if name == "" {
		name = config.Get().Environment
	}
	return envs.Resolve(name)
}
