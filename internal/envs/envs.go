package envs

import "fmt"

type Env struct {
	Name     string
	APIBase  string
	WebBase  string
	ClientID string // Cognito app client (for token refresh)
}

var All = map[string]Env{
	"prod": {Name: "prod", APIBase: "https://apis.mkpdfs.com", WebBase: "https://mkpdfs.com", ClientID: "3mgah7n76j694e5sb0092fl6hn"},
	"dev":  {Name: "dev", APIBase: "https://dev.apis.mkpdfs.com", WebBase: "https://dev.mkpdfs.com", ClientID: "vis091qbpsj164csp32jketbd"},
}

// Resolve returns the env by name, defaulting to prod.
func Resolve(name string) (Env, error) {
	if name == "" {
		name = "prod"
	}
	e, ok := All[name]
	if !ok {
		return Env{}, fmt.Errorf("unknown environment %q (dev|prod)", name)
	}
	return e, nil
}
