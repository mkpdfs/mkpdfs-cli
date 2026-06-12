package cli

import (
	"fmt"

	"github.com/sim4gh/mkpdfs-cli/internal/config"
	"github.com/sim4gh/mkpdfs-cli/internal/envs"
	"github.com/sim4gh/mkpdfs-cli/internal/localmap"
)

func currentEnv() (envs.Env, error) {
	name := flagEnv
	if name == "" {
		name = config.Get().Environment
	}
	e, err := envs.Resolve(name)
	if err != nil {
		// Bad --env / config value is user input → usage error (exit 2).
		return envs.Env{}, fmt.Errorf("%v: %w", err, ErrUsage)
	}
	return e, nil
}

// guardMapEnv refuses cross-environment operations on a .mkpdfs.json that is
// already bound to a different environment. push/pull/generate all share this
// guard so a dev-bound directory never writes to prod (or vice versa).
func guardMapEnv(m *localmap.Map, activeEnv string) error {
	if m.Environment != "" && m.Environment != activeEnv {
		return fmt.Errorf(
			".mkpdfs.json is bound to %q but the active environment is %q — no cross-env operations. Use --env %s or a different directory: %w",
			m.Environment, activeEnv, m.Environment, ErrUsage)
	}
	return nil
}
