package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/sim4gh/mkpdfs-cli/internal/api"
	"github.com/sim4gh/mkpdfs-cli/internal/hbs"
	"github.com/sim4gh/mkpdfs-cli/internal/localmap"
	"github.com/sim4gh/mkpdfs-cli/internal/util"
	"github.com/spf13/cobra"
)

type templateMeta struct {
	TemplateID  string `json:"templateId"`
	Name        string `json:"name"`
	Description string `json:"description"`
	FileSize    int64  `json:"fileSize"`
	UpdatedAt   string `json:"updatedAt"`
	Content     string `json:"content,omitempty"`
}

var pullOut string

func addTemplatesCommands() {
	tplCmd := &cobra.Command{Use: "templates", Aliases: []string{"tpl"}, Short: "Manage templates"}

	listCmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List your templates",
		RunE:    runTemplatesList,
	}
	getCmd := &cobra.Command{
		Use:   "get <templateId>",
		Short: "Show template metadata + variables",
		Args:  cobra.ExactArgs(1),
		RunE:  runTemplatesGet,
	}
	pullCmd := &cobra.Command{
		Use:   "pull <templateId>",
		Short: "Download a template",
		Args:  cobra.ExactArgs(1),
		RunE:  runTemplatesPull,
	}
	pullCmd.Flags().StringVarP(&pullOut, "out", "o", "", "output file (default <name>.hbs)")

	tplCmd.AddCommand(listCmd, getCmd, pullCmd)
	rootCmd.AddCommand(tplCmd)
}

func jwtClient() (*api.Client, error) {
	env, err := currentEnv()
	if err != nil {
		return nil, err
	}
	return api.New(env).WithJWT()
}

func fetchTemplate(id string) (*templateMeta, error) {
	client, err := jwtClient()
	if err != nil {
		return nil, err
	}
	resp, err := client.Get("/templates/" + id)
	if err != nil {
		return nil, err
	}
	var out struct {
		Template templateMeta `json:"template"`
	}
	if err := resp.Unmarshal(&out); err != nil {
		return nil, err
	}
	return &out.Template, nil
}

func runTemplatesList(cmd *cobra.Command, args []string) error {
	client, err := jwtClient()
	if err != nil {
		return err
	}
	resp, err := client.Get("/templates")
	if err != nil {
		return err
	}

	if flagJSON {
		fmt.Println(string(resp.Body))
		return nil
	}

	var body struct {
		Templates []templateMeta `json:"templates"`
	}
	if err := resp.Unmarshal(&body); err != nil {
		return err
	}

	if len(body.Templates) == 0 {
		fmt.Println("No templates found. Use `mkp templates push <file>` to upload one.")
		return nil
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"ID", "Name", "Size", "Updated"})
	table.SetBorder(false)
	table.SetColumnSeparator("  ")
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)

	for _, t := range body.Templates {
		table.Append([]string{
			util.Truncate(t.TemplateID, 36),
			util.Truncate(t.Name, 40),
			util.FormatBytes(t.FileSize),
			t.UpdatedAt,
		})
	}
	table.Render()
	return nil
}

func runTemplatesGet(cmd *cobra.Command, args []string) error {
	tpl, err := fetchTemplate(args[0])
	if err != nil {
		return err
	}

	vars, _ := hbs.Validate(tpl.Content)

	if flagJSON {
		out := map[string]any{
			"templateId": tpl.TemplateID,
			"name":       tpl.Name,
			"updatedAt":  tpl.UpdatedAt,
			"variables":  vars,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}

	bold := color.New(color.Bold)
	bold.Printf("Template: %s\n", tpl.Name)
	fmt.Printf("  ID:          %s\n", tpl.TemplateID)
	fmt.Printf("  Description: %s\n", tpl.Description)
	fmt.Printf("  Size:        %s\n", util.FormatBytes(tpl.FileSize))
	fmt.Printf("  Updated:     %s\n", tpl.UpdatedAt)
	if len(vars) > 0 {
		fmt.Printf("  Variables:   %v\n", vars)
	} else {
		fmt.Printf("  Variables:   (none detected)\n")
	}
	return nil
}

func runTemplatesPull(cmd *cobra.Command, args []string) error {
	tpl, err := fetchTemplate(args[0])
	if err != nil {
		return err
	}

	outPath := pullOut
	if outPath == "" {
		outPath = tpl.Name + ".hbs"
	}

	if err := os.WriteFile(outPath, []byte(tpl.Content), 0644); err != nil {
		return fmt.Errorf("writing %s: %w", outPath, err)
	}

	// Record entry in .mkpdfs.json in cwd
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	m, err := localmap.Load(cwd)
	if err != nil {
		return err
	}
	env, err := currentEnv()
	if err != nil {
		return err
	}
	if m.Environment == "" {
		m.Environment = env.Name
	}
	m.Templates[localmap.Key(outPath)] = localmap.Entry{
		TemplateID:      tpl.TemplateID,
		Name:            tpl.Name,
		RemoteUpdatedAt: tpl.UpdatedAt,
	}
	if err := localmap.Save(cwd, m); err != nil {
		return err
	}

	fmt.Printf("%s Pulled %q → %s\n", color.GreenString("✓"), tpl.Name, outPath)
	return nil
}
