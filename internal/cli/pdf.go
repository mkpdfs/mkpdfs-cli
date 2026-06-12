package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/pkg/browser"
	"github.com/sim4gh/mkpdfs-cli/internal/api"
	"github.com/sim4gh/mkpdfs-cli/internal/localmap"
	"github.com/sim4gh/mkpdfs-cli/internal/util"
	"github.com/spf13/cobra"
)

var (
	pdfTemplate string
	pdfData     string
	pdfOut      string
	pdfOpen     bool
	pdfUseKey   bool
)

func addPdfCommands() {
	pdfCmd := &cobra.Command{Use: "pdf", Short: "Generate PDFs"}
	genCmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate a PDF from a template + JSON data",
		RunE:  runPdfGenerate,
	}
	genCmd.Flags().StringVarP(&pdfTemplate, "template", "t", "", "templateId or local .hbs file (required)")
	genCmd.Flags().StringVarP(&pdfData, "data", "d", "", "JSON data file (required)")
	genCmd.Flags().StringVarP(&pdfOut, "out", "o", "", "output PDF path")
	genCmd.Flags().BoolVar(&pdfOpen, "open", false, "open the PDF after download")
	genCmd.Flags().BoolVar(&pdfUseKey, "api-key", false, "use the server-to-server route with your tlfy_ API key")
	_ = genCmd.MarkFlagRequired("template")
	_ = genCmd.MarkFlagRequired("data")
	pdfCmd.AddCommand(genCmd)
	rootCmd.AddCommand(pdfCmd)
}

// resolveTemplateID maps a local file path to its templateId via .mkpdfs.json.
// If ref is not a file on disk it is assumed to already be a template ID.
func resolveTemplateID(ref string) (string, error) {
	if _, err := os.Stat(ref); err != nil {
		return ref, nil // not a file → assume it's already an ID
	}
	cwd, _ := os.Getwd()
	m, err := localmap.Load(cwd)
	if err != nil {
		return "", err
	}
	if e, ok := m.Templates[localmap.Key(ref)]; ok {
		return e.TemplateID, nil
	}
	return "", fmt.Errorf("%s is not in .mkpdfs.json — push it first: mkp templates push %s: %w", ref, ref, ErrUsage)
}

func runPdfGenerate(cmd *cobra.Command, args []string) error {
	env, err := currentEnv()
	if err != nil {
		return err
	}

	templateID, err := resolveTemplateID(pdfTemplate)
	if err != nil {
		return err
	}

	raw, err := os.ReadFile(pdfData)
	if err != nil {
		return err
	}
	var data any
	if err := json.Unmarshal(raw, &data); err != nil {
		return fmt.Errorf("%s is not valid JSON: %v: %w", pdfData, err, ErrUsage)
	}
	if arr, ok := data.([]any); ok && len(arr) > 50 {
		return fmt.Errorf("batch data has %d items; the API limit is 50: %w", len(arr), ErrUsage)
	}

	body := map[string]any{"templateId": templateID, "data": data}

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " Generating PDF..."
	s.Start()

	var (
		resp     *api.Response
		authPath string
	)
	if pdfUseKey {
		authPath = "API key (/v1/pdf/generate)"
		client, cerr := api.New(env).WithAPIKey()
		if cerr != nil {
			s.Stop()
			return cerr
		}
		resp, err = client.PostWithKey("/v1/pdf/generate", body)
	} else {
		authPath = "JWT (/pdf/generate)"
		client, cerr := api.New(env).WithJWT()
		if cerr != nil {
			s.Stop()
			return cerr
		}
		resp, err = client.Post("/pdf/generate", body)
	}
	s.Stop()
	if err != nil {
		return err
	}

	var result struct {
		PdfURL         string `json:"pdfUrl"`
		Size           int64  `json:"size"`
		PagesGenerated int    `json:"pagesGenerated"`
	}
	if err := resp.Unmarshal(&result); err != nil {
		return err
	}

	if flagJSON {
		out, _ := json.Marshal(map[string]any{
			"pdfUrl":   result.PdfURL,
			"size":     result.Size,
			"pages":    result.PagesGenerated,
			"authPath": authPath,
		})
		fmt.Println(string(out))
		return nil
	}

	outPath := pdfOut
	if outPath == "" {
		idPrefix := templateID
		if len(idPrefix) > 8 {
			idPrefix = idPrefix[:8]
		}
		outPath = fmt.Sprintf("%s-%s.pdf", idPrefix, time.Now().Format("2006-01-02"))
	}

	dl, err := http.Get(result.PdfURL) //nolint:noctx
	if err != nil {
		return err
	}
	defer dl.Body.Close()

	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := io.Copy(f, dl.Body); err != nil {
		return err
	}

	fmt.Printf("%s %s (%s, %d page(s)) — via %s\n",
		color.GreenString("✓"), outPath, util.FormatBytes(result.Size), result.PagesGenerated, authPath)

	if pdfOpen {
		_ = browser.OpenFile(outPath)
	}
	return nil
}
