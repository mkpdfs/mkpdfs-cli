package cli

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/sim4gh/mkpdfs-cli/internal/api"
	"github.com/sim4gh/mkpdfs-cli/internal/auth"
	"github.com/sim4gh/mkpdfs-cli/internal/config"
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

var (
	pullOut    string
	pushID     string
	pushNew    bool
	pushForce  bool
	pushDryRun bool
	delForce   bool
	tplAPIKey  bool
)

// maxTemplateBytes caps template source sent to the API. API Gateway REST has a
// 10 MB request limit; base64 inflates ~33%, so 6.5 MiB leaves room for JSON overhead.
const maxTemplateBytes = 6_815_744 // 6.5 * 1024 * 1024

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

	pushCmd := &cobra.Command{
		Use:   "push <file>",
		Short: "Create or update a template from a .hbs file",
		Args:  cobra.ExactArgs(1),
		RunE:  func(cmd *cobra.Command, args []string) error { return runTemplatesPush(args[0]) },
	}
	pushCmd.Flags().StringVar(&pushID, "id", "", "force update of a specific template ID")
	pushCmd.Flags().BoolVar(&pushNew, "new", false, "force create a new template (ignore local mapping)")
	pushCmd.Flags().BoolVar(&pushForce, "force", false, "override account/conflict guards")
	pushCmd.Flags().BoolVar(&pushDryRun, "dry-run", false, "print what would happen without writing")

	deleteCmd := &cobra.Command{
		Use:   "delete <templateId>",
		Short: "Delete a template",
		Args:  cobra.ExactArgs(1),
		RunE:  func(cmd *cobra.Command, args []string) error { return runTemplatesDelete(args[0]) },
	}
	deleteCmd.Flags().BoolVar(&delForce, "force", false, "skip confirmation prompt")

	tplCmd.PersistentFlags().BoolVar(&tplAPIKey, "api-key", false,
		"authenticate with MKPDFS_API_KEY / saved API key (server-to-server, no browser login)")

	tplCmd.AddCommand(listCmd, getCmd, pullCmd, pushCmd, deleteCmd)
	requireSubcommand(tplCmd)
	rootCmd.AddCommand(tplCmd)
}

// sanitizeFilename makes a template name safe to use as a local filename by
// replacing path separators and whitespace with hyphens.
func sanitizeFilename(name string) string {
	r := strings.NewReplacer("/", "-", "\\", "-", " ", "-")
	out := strings.Trim(r.Replace(name), "-")
	if out == "" {
		out = "template"
	}
	return out
}

func confirm(prompt string) bool {
	if flagYes {
		return true
	}
	fmt.Printf("%s [y/N]: ", prompt)
	var resp string
	fmt.Scanln(&resp)
	return resp == "y" || resp == "Y" || resp == "yes"
}

func runTemplatesPush(file string) error {
	env, err := currentEnv()
	if err != nil {
		return err
	}

	content, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("reading %s: %w", file, err)
	}
	if _, err := hbs.Validate(string(content)); err != nil {
		return fmt.Errorf("handlebars syntax error in %s: %v: %w", file, err, ErrUsage)
	}
	if len(content) > maxTemplateBytes {
		return fmt.Errorf("template %s is %s; the API limit is 6.5 MiB: %w",
			file, util.FormatBytes(int64(len(content))), ErrUsage)
	}

	client, prefix, err := templatesClient()
	if err != nil {
		return err
	}

	var userID string
	if !tplAPIKey {
		if creds := config.Get().Creds(env.Name); creds != nil && creds.IDToken != "" {
			if payload, err := auth.DecodeJWT(creds.IDToken); err == nil && payload != nil {
				userID = payload.Sub
			}
		}
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	m, err := localmap.Load(cwd)
	if err != nil {
		return err
	}

	// If the entry is known and we're not forcing create/specific-id, fetch the
	// remote updatedAt so decidePush can detect drift. On fetch error, treat as
	// unknown ("") — decidePush then skips the conflict check.
	var remoteUpdatedAt string
	entry, known := m.Templates[localmap.Key(file)]
	if known && !pushNew && pushID == "" {
		if remote, ferr := fetchTemplate(entry.TemplateID); ferr == nil {
			remoteUpdatedAt = remote.UpdatedAt
		} else if flagVerbose {
			fmt.Printf("warning: could not fetch remote template %s for conflict check: %v\n", entry.TemplateID, ferr)
		}
	}

	decision, err := decidePush(pushInput{
		File:            file,
		Map:             m,
		ActiveEnv:       env.Name,
		UserID:          userID,
		RemoteUpdatedAt: remoteUpdatedAt,
		Force:           pushForce,
		ForceID:         pushID,
		ForceNew:        pushNew,
		APIKeyMode:      tplAPIKey,
	})
	if err != nil {
		return err
	}

	name := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))

	if pushDryRun {
		if decision.Action == pushCreate {
			fmt.Printf("dry-run: would CREATE %q in %s (%s)\n", name, env.Name, util.FormatBytes(int64(len(content))))
		} else {
			fmt.Printf("dry-run: would UPDATE %s %q in %s (%s)\n", decision.TemplateID, name, env.Name, util.FormatBytes(int64(len(content))))
		}
		return nil
	}

	if env.Name == "prod" && !confirm(fmt.Sprintf("Push %q to PRODUCTION?", name)) {
		return nil
	}
	if decision.Action == pushCreate && !confirm(fmt.Sprintf("Create new template %q?", name)) {
		return nil
	}

	body := map[string]string{
		"name":            name,
		"content":         base64.StdEncoding.EncodeToString(content),
		"contentEncoding": "base64",
	}

	var resp *api.Response
	if decision.Action == pushCreate {
		resp, err = client.Post(prefix+"/upload", body)
	} else {
		resp, err = client.Put(prefix+"/"+decision.TemplateID, body)
		if err != nil && resp != nil && resp.StatusCode == 404 {
			return fmt.Errorf("remote template %s no longer exists — your .mkpdfs.json entry is stale. Push with --new to create it again: %w",
				decision.TemplateID, ErrUsage)
		}
	}
	if err != nil {
		return err
	}

	result, err := parsePushResult(resp.Body)
	if err != nil {
		return err
	}

	// Report success BEFORE touching .mkpdfs.json — the push already happened
	// server-side, so a local save failure must not mask it (exit stays 0).
	verb := "Created"
	if decision.Action == pushUpdate {
		verb = "Updated"
	}
	fmt.Printf("%s %s %q (%s) — %s\n",
		color.GreenString("✓"), verb, result.Name, result.TemplateID, util.FormatBytes(result.FileSize))

	if m.Environment == "" {
		m.Environment = env.Name
	}
	if m.UserID == "" {
		m.UserID = userID
	}
	m.Templates[localmap.Key(file)] = localmap.Entry{
		TemplateID:      result.TemplateID,
		Name:            result.Name,
		RemoteUpdatedAt: result.UpdatedAt,
	}
	if err := localmap.Save(cwd, m); err != nil {
		fmt.Printf("warning: push succeeded but .mkpdfs.json could not be updated: %v\n", err)
	}
	return nil
}

// parsePushResult parses the push response body. Both uploadTemplate (201) and
// updateTemplate (200) return the template object at the TOP LEVEL (not nested
// under "template"). An empty templateId means an unexpected shape — recording
// it in .mkpdfs.json would poison future pushes, so reject it.
func parsePushResult(body []byte) (*templateMeta, error) {
	var result templateMeta
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing API response: %w", err)
	}
	if result.TemplateID == "" {
		return nil, fmt.Errorf("unexpected response shape from push (no templateId) — not updating .mkpdfs.json")
	}
	return &result, nil
}

func runTemplatesDelete(id string) error {
	if !delForce && !confirm(fmt.Sprintf("Delete template %s? This cannot be undone.", id)) {
		return nil
	}
	client, prefix, err := templatesClient()
	if err != nil {
		return err
	}
	if _, err := client.Delete(prefix + "/" + id); err != nil {
		return err
	}
	fmt.Printf("%s Deleted %s\n", color.GreenString("✓"), id)

	// Prune any stale local mapping for the deleted template. Load errors are
	// ignored on purpose — the cwd may be unrelated to this template.
	if cwd, err := os.Getwd(); err == nil {
		if m, err := localmap.Load(cwd); err == nil {
			for key, entry := range m.Templates {
				if entry.TemplateID == id {
					delete(m.Templates, key)
					if err := localmap.Save(cwd, m); err == nil {
						fmt.Printf("  (removed stale entry %q from .mkpdfs.json)\n", key)
					}
				}
			}
		}
	}
	return nil
}

func jwtClient() (*api.Client, error) {
	env, err := currentEnv()
	if err != nil {
		return nil, err
	}
	return api.New(env).WithJWT()
}

// templatesPrefix returns the route prefix for templates given the auth mode.
func templatesPrefix(apiKey bool) string {
	if apiKey {
		return "/v1/templates"
	}
	return "/templates"
}

// templatesClient builds the API client + route prefix honoring --api-key.
func templatesClient() (*api.Client, string, error) {
	env, err := currentEnv()
	if err != nil {
		return nil, "", err
	}
	if tplAPIKey {
		c, err := api.New(env).WithAPIKey()
		return c, templatesPrefix(true), err
	}
	c, err := api.New(env).WithJWT()
	return c, templatesPrefix(false), err
}

func fetchTemplate(id string) (*templateMeta, error) {
	client, prefix, err := templatesClient()
	if err != nil {
		return nil, err
	}
	resp, err := client.Get(prefix + "/" + id)
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
	client, prefix, err := templatesClient()
	if err != nil {
		return err
	}
	resp, err := client.Get(prefix)
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
		outPath = sanitizeFilename(tpl.Name) + ".hbs"
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
	if err := guardMapEnv(m, env.Name); err != nil {
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
