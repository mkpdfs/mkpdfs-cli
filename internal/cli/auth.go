package cli

import (
	"fmt"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/pkg/browser"
	"github.com/sim4gh/mkpdfs-cli/internal/api"
	"github.com/sim4gh/mkpdfs-cli/internal/auth"
	"github.com/sim4gh/mkpdfs-cli/internal/config"
	"github.com/spf13/cobra"
)

func addAuthCommands() {
	authCmd := &cobra.Command{Use: "auth", Short: "Authentication"}
	authCmd.AddCommand(
		&cobra.Command{Use: "login", Short: "Log in via your browser", RunE: runLogin},
		&cobra.Command{Use: "logout", Short: "Clear credentials for the current environment", RunE: runLogout},
		&cobra.Command{Use: "whoami", Short: "Show current user and plan", RunE: runWhoami},
	)
	requireSubcommand(authCmd)
	rootCmd.AddCommand(authCmd)
}

func runLogin(cmd *cobra.Command, args []string) error {
	env, err := currentEnv()
	if err != nil {
		return err
	}

	da, err := auth.InitiateDeviceAuth(env.APIBase)
	if err != nil {
		return err
	}

	pretty := da.UserCode
	if len(pretty) == 8 {
		pretty = da.UserCode[:4] + "-" + da.UserCode[4:]
	}
	fmt.Printf("\n  Opening %s\n\n", da.VerificationURI)
	fmt.Printf("  Verification code: %s\n\n", color.New(color.Bold, color.FgCyan).Sprint(pretty))
	fmt.Println("  If the browser doesn't open, visit the URL and enter the code manually.")
	_ = browser.OpenURL(da.VerificationURI)

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " Waiting for authorization..."
	s.Start()
	tok, err := auth.PollForToken(env.APIBase, da.DeviceCode, da.Interval)
	s.Stop()
	if err != nil {
		return err
	}

	cfg := config.Get()
	cfg.Environment = env.Name
	creds := cfg.Creds(env.Name)
	creds.IDToken = tok.IDToken
	creds.AccessToken = tok.AccessToken
	creds.RefreshToken = tok.RefreshToken
	creds.LoggedInAt = time.Now().Format(time.RFC3339)
	if err := config.SetConfig(cfg); err != nil {
		return err
	}

	payload, _ := auth.DecodeJWT(tok.IDToken)
	email := ""
	if payload != nil {
		email = payload.Email
	}
	fmt.Printf("\n  %s Logged in as %s (%s)\n\n  Ready. Try: mkp templates list\n",
		color.GreenString("✓"), email, env.Name)
	return nil
}

func runLogout(cmd *cobra.Command, args []string) error {
	env, err := currentEnv()
	if err != nil {
		return err
	}
	cfg := config.Get()
	*cfg.Creds(env.Name) = config.Creds{}
	if err := config.SetConfig(cfg); err != nil {
		return err
	}
	fmt.Printf("Logged out of %s.\n", env.Name)
	return nil
}

func runWhoami(cmd *cobra.Command, args []string) error {
	env, err := currentEnv()
	if err != nil {
		return err
	}
	client, err := api.New(env).WithJWT()
	if err != nil {
		return err
	}
	resp, err := client.Get("/user/profile")
	if err != nil {
		return err
	}
	if flagJSON {
		fmt.Println(string(resp.Body))
		return nil
	}
	// The backend returns { success: true, data: { email, subscription: { plan, status, ... }, ... } }
	var profile struct {
		Data struct {
			Email        string `json:"email"`
			Subscription struct {
				Plan   string `json:"plan"`
				Status string `json:"status"`
			} `json:"subscription"`
		} `json:"data"`
	}
	_ = resp.Unmarshal(&profile)
	fmt.Printf("Email: %s\nPlan:  %s\nEnv:   %s\n", profile.Data.Email, profile.Data.Subscription.Plan, env.Name)

	// Best-effort current-month usage one-liner. On any error, skip silently.
	if uresp, uerr := client.Get("/user/usage"); uerr == nil {
		var usage struct {
			Usage struct {
				PagesGenerated    int `json:"pagesGenerated"`
				TemplatesUploaded int `json:"templatesUploaded"`
				TokensCreated     int `json:"tokensCreated"`
			} `json:"usage"`
		}
		if uresp.Unmarshal(&usage) == nil {
			fmt.Printf("Usage: %d PDFs, %d templates, %d tokens this month\n",
				usage.Usage.PagesGenerated, usage.Usage.TemplatesUploaded, usage.Usage.TokensCreated)
		}
	}
	return nil
}
