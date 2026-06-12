package cli

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/sim4gh/mkpdfs-cli/internal/util"
	"github.com/spf13/cobra"
)

func addUsageCommand() {
	usageCmd := &cobra.Command{
		Use:   "usage",
		Short: "Show your usage for the current billing period",
		RunE:  runUsage,
	}
	rootCmd.AddCommand(usageCmd)
}

// usageData matches the getUsage handler response:
//
//	{
//	  usage: {
//	    userId, yearMonth,
//	    pagesGenerated,    // pdfCount in DB mapped here
//	    templatesUploaded, // actual template count
//	    tokensCreated,     // actual token count
//	    bytesGenerated     // totalSizeBytes in DB
//	  },
//	  currentPeriod: "YYYY-MM"
//	}
type usageData struct {
	UserID            string `json:"userId"`
	YearMonth         string `json:"yearMonth"`
	PagesGenerated    int    `json:"pagesGenerated"`
	TemplatesUploaded int    `json:"templatesUploaded"`
	TokensCreated     int    `json:"tokensCreated"`
	BytesGenerated    int64  `json:"bytesGenerated"`
}

func runUsage(cmd *cobra.Command, args []string) error {
	client, err := jwtClient()
	if err != nil {
		return err
	}
	resp, err := client.Get("/user/usage")
	if err != nil {
		return err
	}

	if flagJSON {
		fmt.Println(string(resp.Body))
		return nil
	}

	var body struct {
		Usage         usageData `json:"usage"`
		CurrentPeriod string    `json:"currentPeriod"`
	}
	if err := resp.Unmarshal(&body); err != nil {
		return err
	}

	u := body.Usage
	period := body.CurrentPeriod
	if period == "" {
		period = u.YearMonth
	}

	bold := color.New(color.Bold)
	bold.Printf("Usage — %s\n\n", period)
	fmt.Printf("  PDFs generated:    %d\n", u.PagesGenerated)
	fmt.Printf("  Templates:         %d\n", u.TemplatesUploaded)
	fmt.Printf("  API tokens:        %d\n", u.TokensCreated)
	fmt.Printf("  Data generated:    %s\n", util.FormatBytes(u.BytesGenerated))
	fmt.Println()
	return nil
}
