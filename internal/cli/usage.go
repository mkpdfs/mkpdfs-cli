package cli

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/sim4gh/mkpdfs-cli/internal/api"
	"github.com/sim4gh/mkpdfs-cli/internal/util"
	"github.com/spf13/cobra"
)

func addUsageCommand() {
	usageCmd := &cobra.Command{
		Use:   "usage",
		Short: "Show your current-month usage stats and credit balance",
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

	// Best-effort profile read for limits + credit balance. On error, skip those
	// lines and just print the raw stats. Note: the prepaid-credits model dropped
	// the old monthly page cap — there is NO per-month PDF limit anymore, so the
	// page counter below is a plain stat, not "used / limit".
	limits, sub, haveProfile := fetchProfile(client)

	bold := color.New(color.Bold)
	bold.Printf("Usage — %s\n\n", period)

	if haveProfile {
		fmt.Printf("  Credit balance:     %s\n\n", creditBalanceString(sub))
	}

	fmt.Printf("  PDF pages generated: %d\n", u.PagesGenerated)
	if haveProfile {
		fmt.Printf("  Templates:           %s\n", limitLine(u.TemplatesUploaded, limits.TemplatesAllowed))
		fmt.Printf("  API tokens:          %s\n", limitLine(u.TokensCreated, limits.APITokensAllowed))
	} else {
		fmt.Printf("  Templates:           %d\n", u.TemplatesUploaded)
		fmt.Printf("  API tokens:          %d\n", u.TokensCreated)
	}
	fmt.Printf("  Data generated:      %s\n", util.FormatBytes(u.BytesGenerated))
	fmt.Println()
	fmt.Println("  Detailed credit history: mkp credits ledger")
	return nil
}

// subscriptionLimits mirrors the limits the getProfile handler returns. A value of
// -1 means unlimited. The old `pagesPerMonth` field was removed in the prepaid-credits
// migration and is intentionally absent here.
type subscriptionLimits struct {
	TemplatesAllowed int `json:"templatesAllowed"`
	APITokensAllowed int `json:"apiTokensAllowed"`
}

// fetchProfile reads limits + subscription (plan, creditBalance) from GET /user/profile.
func fetchProfile(client *api.Client) (subscriptionLimits, subscriptionInfo, bool) {
	resp, err := client.Get("/user/profile")
	if err != nil {
		return subscriptionLimits{}, subscriptionInfo{}, false
	}
	var p profileResponse
	if err := resp.Unmarshal(&p); err != nil {
		return subscriptionLimits{}, subscriptionInfo{}, false
	}
	return p.Data.SubscriptionLimits, p.Data.Subscription, true
}

// limitLine formats "used / limit", rendering -1 as "unlimited".
func limitLine(used, limit int) string {
	if limit < 0 {
		return fmt.Sprintf("%d / unlimited", used)
	}
	return fmt.Sprintf("%d / %d", used, limit)
}
