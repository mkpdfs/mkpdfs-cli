package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/sim4gh/mkpdfs-cli/internal/config"
	"github.com/spf13/cobra"
)

var (
	tokenName        string
	tokenExpiresDays int
	tokenSave        bool
)

func addTokensCommands() {
	tokCmd := &cobra.Command{Use: "tokens", Short: "Manage API tokens"}

	listCmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List your API tokens",
		RunE:    runTokensList,
	}

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new API token",
		RunE:  runTokensCreate,
	}
	createCmd.Flags().StringVar(&tokenName, "name", "CLI token", "token name")
	createCmd.Flags().IntVar(&tokenExpiresDays, "expires-days", 0, "days until token expires (0 = never)")
	createCmd.Flags().BoolVar(&tokenSave, "save", false, "save token as the CLI API key for this environment")

	revokeCmd := &cobra.Command{
		Use:   "revoke <tokenId>",
		Short: "Revoke an API token",
		Args:  cobra.ExactArgs(1),
		RunE:  func(cmd *cobra.Command, args []string) error { return runTokensRevoke(args[0]) },
	}

	tokCmd.AddCommand(listCmd, createCmd, revokeCmd)
	requireSubcommand(tokCmd)
	rootCmd.AddCommand(tokCmd)
}

// tokenItem matches the listTokens projection: tokenId, name, createdAt, lastUsed, usageCount.
// The active field is NOT in the projection — the backend omits it from list output.
type tokenItem struct {
	TokenID    string `json:"tokenId"`
	Name       string `json:"name"`
	CreatedAt  string `json:"createdAt"`
	LastUsed   string `json:"lastUsed"`
	UsageCount int    `json:"usageCount"`
	ExpiresAt  string `json:"expiresAt"`
}

func runTokensList(cmd *cobra.Command, args []string) error {
	client, err := jwtClient()
	if err != nil {
		return err
	}
	resp, err := client.Get("/user/tokens")
	if err != nil {
		return err
	}

	if flagJSON {
		fmt.Println(string(resp.Body))
		return nil
	}

	var body struct {
		Tokens []tokenItem `json:"tokens"`
	}
	if err := resp.Unmarshal(&body); err != nil {
		return err
	}

	if len(body.Tokens) == 0 {
		fmt.Println("No tokens found. Use `mkp tokens create` to create one.")
		return nil
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"ID", "Name", "Last Used", "Expires"})
	table.SetBorder(false)
	table.SetColumnSeparator("  ")
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)

	for _, t := range body.Tokens {
		lastUsed := t.LastUsed
		if lastUsed == "" {
			lastUsed = "never"
		}
		expires := t.ExpiresAt
		if expires == "" {
			expires = "never"
		}
		table.Append([]string{
			t.TokenID,
			t.Name,
			lastUsed,
			expires,
		})
	}
	table.Render()
	return nil
}

func runTokensCreate(cmd *cobra.Command, args []string) error {
	env, err := currentEnv()
	if err != nil {
		return err
	}

	client, err := jwtClient()
	if err != nil {
		return err
	}

	body := map[string]any{"name": tokenName}
	if tokenExpiresDays > 0 {
		body["expiresInDays"] = tokenExpiresDays
	}

	resp, err := client.Post("/user/tokens", body)
	if err != nil {
		return err
	}

	var result struct {
		Success bool `json:"success"`
		Data    struct {
			Token     string `json:"token"`
			Name      string `json:"name"`
			ExpiresAt string `json:"expiresAt"`
			Message   string `json:"message"`
		} `json:"data"`
	}
	if err := resp.Unmarshal(&result); err != nil {
		return err
	}

	if flagJSON {
		out, _ := json.Marshal(map[string]any{
			"token":     result.Data.Token,
			"name":      result.Data.Name,
			"expiresAt": result.Data.ExpiresAt,
		})
		fmt.Println(string(out))
	} else {
		fmt.Printf("%s Token created: %s\n", color.GreenString("✓"), result.Data.Name)
		fmt.Printf("\n  %s\n\n", result.Data.Token)
		color.Yellow("  Store this token securely — it cannot be retrieved again.")
		fmt.Println()
		if result.Data.ExpiresAt != "" {
			fmt.Printf("  Expires: %s\n", result.Data.ExpiresAt)
		} else {
			fmt.Printf("  Expires: never\n")
		}
	}

	if tokenSave {
		cfg := config.Get()
		creds := cfg.Creds(env.Name)
		creds.APIKey = result.Data.Token
		if err := config.SetConfig(cfg); err != nil {
			return fmt.Errorf("saving API key to config: %w", err)
		}
		fmt.Printf("\n%s Saved as the CLI API key for %s\n", color.GreenString("✓"), env.Name)
	}

	return nil
}

func runTokensRevoke(tokenID string) error {
	client, err := jwtClient()
	if err != nil {
		return err
	}
	if _, err := client.Delete("/user/tokens/" + tokenID); err != nil {
		return err
	}

	// Check for non-JSON --json flag: if set, just print JSON success
	if flagJSON {
		out, _ := json.Marshal(map[string]any{"success": true, "tokenId": tokenID})
		fmt.Println(string(out))
		return nil
	}

	fmt.Printf("%s Revoked token %s\n", color.GreenString("✓"), tokenID)
	return nil
}
