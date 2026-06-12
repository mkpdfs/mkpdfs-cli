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

	// Best-effort plan limits from GET /user/profile. The getProfile handler
	// returns subscriptionLimits (sourced from SUBSCRIPTION_PLANS in
	// mkpdfs-backend/src/libs/middleware/subscription.ts). On error, skip the
	// "X / Y" columns and just print the raw counts.
	limits, plan, haveLimits := fetchLimits(client)

	bold := color.New(color.Bold)
	if plan != "" {
		bold.Printf("Usage — %s (%s plan)\n\n", period, plan)
	} else {
		bold.Printf("Usage — %s\n\n", period)
	}
	if haveLimits {
		fmt.Printf("  PDFs generated:    %s\n", limitLine(u.PagesGenerated, limits.PagesPerMonth))
		fmt.Printf("  Templates:         %s\n", limitLine(u.TemplatesUploaded, limits.TemplatesAllowed))
		fmt.Printf("  API tokens:        %s\n", limitLine(u.TokensCreated, limits.APITokensAllowed))
	} else {
		fmt.Printf("  PDFs generated:    %d\n", u.PagesGenerated)
		fmt.Printf("  Templates:         %d\n", u.TemplatesUploaded)
		fmt.Printf("  API tokens:        %d\n", u.TokensCreated)
	}
	fmt.Printf("  Data generated:    %s\n", util.FormatBytes(u.BytesGenerated))
	fmt.Println()
	return nil
}

// subscriptionLimits mirrors the SubscriptionLimits interface in
// mkpdfs-backend/src/libs/middleware/subscription.ts. A value of -1 means
// unlimited.
type subscriptionLimits struct {
	PagesPerMonth    int `json:"pagesPerMonth"`
	TemplatesAllowed int `json:"templatesAllowed"`
	APITokensAllowed int `json:"apiTokensAllowed"`
}

// fetchLimits reads subscriptionLimits + plan from GET /user/profile.
func fetchLimits(client *api.Client) (subscriptionLimits, string, bool) {
	resp, err := client.Get("/user/profile")
	if err != nil {
		return subscriptionLimits{}, "", false
	}
	var profile struct {
		Data struct {
			Subscription struct {
				Plan string `json:"plan"`
			} `json:"subscription"`
			SubscriptionLimits subscriptionLimits `json:"subscriptionLimits"`
		} `json:"data"`
	}
	if err := resp.Unmarshal(&profile); err != nil {
		return subscriptionLimits{}, "", false
	}
	return profile.Data.SubscriptionLimits, profile.Data.Subscription.Plan, true
}

// limitLine formats "used / limit", rendering -1 as "unlimited".
func limitLine(used, limit int) string {
	if limit < 0 {
		return fmt.Sprintf("%d / unlimited", used)
	}
	return fmt.Sprintf("%d / %d", used, limit)
}
