package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/pkg/browser"
	"github.com/sim4gh/mkpdfs-cli/internal/api"
	"github.com/spf13/cobra"
)

// creditsAPI is the subset of *api.Client the credits commands use. It exists so
// tests can inject a fake transport without a live server or real credentials.
type creditsAPI interface {
	Get(path string) (*api.Response, error)
	Put(path string, body any) (*api.Response, error)
	Post(path string, body any) (*api.Response, error)
}

// openURL is indirected (vs calling browser.OpenURL directly like auth login does)
// so `credits buy` tests can stub the browser launch.
var openURL = browser.OpenURL

var (
	arEnable    bool
	arDisable   bool
	arThreshold int
)

// subscriptionInfo mirrors data.subscription from GET /user/profile (credits fields).
type subscriptionInfo struct {
	Plan              string  `json:"plan"`
	CreditBalance     float64 `json:"creditBalance"`
	AutoRecharge      bool    `json:"autoRecharge"`
	RechargeThreshold int     `json:"rechargeThreshold"`
	AutoRechargeError string  `json:"autoRechargeError"`
}

// profileResponse decodes GET /user/profile. Shared by credits.go and usage.go so
// the two surfaces don't drift on the same payload.
type profileResponse struct {
	Data struct {
		Subscription       subscriptionInfo   `json:"subscription"`
		SubscriptionLimits subscriptionLimits `json:"subscriptionLimits"`
	} `json:"data"`
}

// ledgerEntry mirrors one entry from GET /billing/ledger. Amount is ALREADY signed
// by the backend (debit/refund negative, purchase/recharge positive) — render as-is.
type ledgerEntry struct {
	EntryID      string   `json:"entryId"`
	Type         string   `json:"type"`
	Amount       float64  `json:"amount"`
	BalanceAfter *float64 `json:"balanceAfter"`
	Description  *string  `json:"description"`
	CreatedAt    string   `json:"createdAt"`
}

type autoRechargeRequest struct {
	Enabled   bool `json:"enabled"`
	Threshold *int `json:"threshold,omitempty"`
}

type autoRechargeResponse struct {
	AutoRecharge      bool `json:"autoRecharge"`
	RechargeThreshold int  `json:"rechargeThreshold"`
}

type checkoutResponse struct {
	Success   bool   `json:"success"`
	URL       string `json:"url"`
	SessionID string `json:"sessionId"`
}

func addCreditsCommands() {
	rootCmd.AddCommand(newCreditsCmd())
}

// newCreditsCmd builds the `credits` command tree. It's a builder (not wired
// straight into rootCmd) so tests can construct a fresh tree per case and avoid
// shared flag state across Cobra executions.
func newCreditsCmd() *cobra.Command {
	// Parent has its OWN action (balance) — so it canNOT use requireSubcommand,
	// which overwrites RunE. Instead runCreditsBalance treats any leftover arg
	// (an unknown subcommand) as a usage error (exit 2).
	creditsCmd := &cobra.Command{
		Use:   "credits",
		Short: "View and manage prepaid credits",
		RunE:  runCreditsBalance,
	}

	ledgerCmd := &cobra.Command{
		Use:   "ledger",
		Short: "Show recent credit ledger entries",
		Args:  cobra.NoArgs,
		RunE:  runCreditsLedger,
	}

	autoRechargeCmd := &cobra.Command{
		Use:   "auto-recharge",
		Short: "Show or change auto-recharge settings",
		Args:  cobra.NoArgs,
		RunE:  runCreditsAutoRecharge,
	}
	autoRechargeCmd.Flags().BoolVar(&arEnable, "enable", false, "enable auto-recharge")
	autoRechargeCmd.Flags().BoolVar(&arDisable, "disable", false, "disable auto-recharge")
	autoRechargeCmd.Flags().IntVar(&arThreshold, "threshold", 0, "recharge when balance falls below N credits (1-1000)")

	buyCmd := &cobra.Command{
		Use:   "buy",
		Short: "Buy a credit pack (opens Stripe checkout in your browser)",
		Args:  cobra.NoArgs,
		RunE:  runCreditsBuy,
	}

	creditsCmd.AddCommand(ledgerCmd, autoRechargeCmd, buyCmd)
	return creditsCmd
}

// --- balance (parent action) ---

func runCreditsBalance(cmd *cobra.Command, args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("unknown command %q for %q: %w", args[0], cmd.CommandPath(), ErrUsage)
	}
	client, err := jwtClient()
	if err != nil {
		return err
	}
	return creditsBalance(client, cmd.OutOrStdout(), flagJSON)
}

func creditsBalance(c creditsAPI, w io.Writer, asJSON bool) error {
	resp, err := c.Get("/user/profile")
	if err != nil {
		return err
	}
	var p profileResponse
	if err := resp.Unmarshal(&p); err != nil {
		return err
	}
	sub := p.Data.Subscription

	if asJSON {
		out, _ := json.Marshal(map[string]any{
			"creditBalance":     sub.CreditBalance,
			"autoRecharge":      sub.AutoRecharge,
			"rechargeThreshold": sub.RechargeThreshold,
			"autoRechargeError": sub.AutoRechargeError,
			"plan":              sub.Plan,
		})
		fmt.Fprintln(w, string(out))
		return nil
	}

	color.New(color.Bold).Fprintf(w, "Credits — %s remaining\n\n", creditBalanceString(sub))

	ar := "off"
	if sub.AutoRecharge {
		ar = fmt.Sprintf("on  (when balance < %d)", sub.RechargeThreshold)
	}
	fmt.Fprintf(w, "  Auto-recharge:  %s\n", ar)
	fmt.Fprintf(w, "  Plan:           %s\n", sub.Plan)

	if sub.AutoRechargeError != "" {
		color.New(color.FgRed).Fprintf(w,
			"\n  Auto-recharge failed: %s. Update your card: mkp credits buy\n", sub.AutoRechargeError)
	}
	if sub.Plan != "enterprise" && sub.CreditBalance < 0 {
		fmt.Fprintln(w, "\n  Buy credits to continue: mkp credits buy")
	}
	return nil
}

// --- ledger ---

func runCreditsLedger(cmd *cobra.Command, args []string) error {
	client, err := jwtClient()
	if err != nil {
		return err
	}
	return creditsLedger(client, cmd.OutOrStdout(), flagJSON)
}

func creditsLedger(c creditsAPI, w io.Writer, asJSON bool) error {
	resp, err := c.Get("/billing/ledger")
	if err != nil {
		return err
	}
	if asJSON {
		fmt.Fprintln(w, string(resp.Body))
		return nil
	}

	var body struct {
		Entries []ledgerEntry `json:"entries"`
	}
	if err := resp.Unmarshal(&body); err != nil {
		return err
	}
	if len(body.Entries) == 0 {
		fmt.Fprintln(w, "No ledger entries yet.")
		return nil
	}

	table := tablewriter.NewWriter(w)
	table.SetHeader([]string{"Date", "Type", "Amount", "Balance", "Description"})
	table.SetBorder(false)
	table.SetColumnSeparator("  ")
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)

	for _, e := range body.Entries {
		bal := ""
		if e.BalanceAfter != nil {
			bal = formatCredits(*e.BalanceAfter)
		}
		desc := ""
		if e.Description != nil {
			desc = *e.Description
		}
		amt := formatSignedCredits(e.Amount)
		if e.Amount > 0 {
			// Credits-in stand out; color.GreenString is a no-op when stdout
			// isn't a TTY (so --json and piped output stay clean).
			amt = color.GreenString(amt)
		}
		table.Append([]string{shortTime(e.CreatedAt), e.Type, amt, bal, desc})
	}
	table.Render()

	if len(body.Entries) == 50 {
		fmt.Fprintln(w, "(showing most recent 50)")
	}
	return nil
}

// --- auto-recharge ---

func runCreditsAutoRecharge(cmd *cobra.Command, args []string) error {
	threshChanged := cmd.Flags().Changed("threshold")
	if err := validateAutoRechargeFlags(arEnable, arDisable, threshChanged, arThreshold); err != nil {
		return err
	}
	client, err := jwtClient()
	if err != nil {
		return err
	}
	w := cmd.OutOrStdout()
	switch {
	case arEnable:
		var t *int
		if threshChanged {
			t = &arThreshold
		}
		return autoRechargeSet(client, w, flagJSON, true, t)
	case arDisable:
		return autoRechargeSet(client, w, flagJSON, false, nil)
	default:
		return autoRechargeShow(client, w, flagJSON)
	}
}

// validateAutoRechargeFlags enforces the flag-combination contract (all usage errors).
func validateAutoRechargeFlags(enable, disable, threshChanged bool, threshold int) error {
	if enable && disable {
		return fmt.Errorf("--enable and --disable are mutually exclusive: %w", ErrUsage)
	}
	if threshChanged && !enable {
		return fmt.Errorf("--threshold can only be used with --enable: %w", ErrUsage)
	}
	if enable && threshChanged && (threshold < 1 || threshold > 1000) {
		return fmt.Errorf("--threshold must be between 1 and 1000: %w", ErrUsage)
	}
	return nil
}

func autoRechargeSet(c creditsAPI, w io.Writer, asJSON, enabled bool, threshold *int) error {
	resp, err := c.Put("/billing/auto-recharge", autoRechargeRequest{Enabled: enabled, Threshold: threshold})
	if err != nil {
		// The backend returns 400 NO_PAYMENT_METHOD when there's no saved card.
		// Inspect resp even though err is set (api.Client returns both on >=400).
		if resp != nil && resp.StatusCode == 400 {
			var e struct {
				Error string `json:"error"`
			}
			if resp.Unmarshal(&e) == nil && e.Error == "NO_PAYMENT_METHOD" {
				return errors.New("No saved card. Buy a credit pack first: mkp credits buy")
			}
		}
		return err
	}

	var out autoRechargeResponse
	if err := resp.Unmarshal(&out); err != nil {
		return err
	}
	if asJSON {
		b, _ := json.Marshal(out)
		fmt.Fprintln(w, string(b))
		return nil
	}
	if out.AutoRecharge {
		fmt.Fprintf(w, "%s Auto-recharge enabled (threshold %d)\n", color.GreenString("✓"), out.RechargeThreshold)
	} else {
		fmt.Fprintf(w, "%s Auto-recharge disabled\n", color.GreenString("✓"))
	}
	return nil
}

func autoRechargeShow(c creditsAPI, w io.Writer, asJSON bool) error {
	resp, err := c.Get("/user/profile")
	if err != nil {
		return err
	}
	var p profileResponse
	if err := resp.Unmarshal(&p); err != nil {
		return err
	}
	sub := p.Data.Subscription
	if asJSON {
		b, _ := json.Marshal(autoRechargeResponse{AutoRecharge: sub.AutoRecharge, RechargeThreshold: sub.RechargeThreshold})
		fmt.Fprintln(w, string(b))
		return nil
	}
	if sub.AutoRecharge {
		fmt.Fprintf(w, "Auto-recharge is on (threshold %d)\n", sub.RechargeThreshold)
	} else {
		fmt.Fprintln(w, "Auto-recharge is off")
	}
	return nil
}

// --- buy ---

func runCreditsBuy(cmd *cobra.Command, args []string) error {
	client, err := jwtClient()
	if err != nil {
		return err
	}
	return creditsBuy(client, cmd.OutOrStdout(), flagJSON)
}

func creditsBuy(c creditsAPI, w io.Writer, asJSON bool) error {
	resp, err := c.Post("/stripe/create-checkout-session", nil)
	if err != nil {
		return err
	}
	if asJSON {
		fmt.Fprintln(w, string(resp.Body))
		return nil
	}
	var out checkoutResponse
	if err := resp.Unmarshal(&out); err != nil {
		return err
	}
	if out.URL == "" {
		return errors.New("checkout session did not return a URL")
	}
	fmt.Fprintf(w, "\n  Opening checkout: %s\n\n", out.URL)
	if err := openURL(out.URL); err != nil {
		fmt.Fprintln(w, "  (couldn't open your browser automatically — copy the URL above)")
	}
	return nil
}

// --- formatting helpers ---

// creditBalanceString renders the balance: "unlimited" for enterprise, otherwise a
// whole integer with thousands separators (the JSON number is float64 but credits
// are whole pages).
func creditBalanceString(sub subscriptionInfo) string {
	if sub.Plan == "enterprise" {
		return "unlimited"
	}
	return formatCredits(sub.CreditBalance)
}

// formatCredits renders a whole-credit amount with thousands separators (negatives
// keep their sign): 1240 -> "1,240", -30 -> "-30".
func formatCredits(n float64) string {
	i := int64(math.Round(n))
	neg := i < 0
	if neg {
		i = -i
	}
	s := strconv.FormatInt(i, 10)
	var b strings.Builder
	pre := len(s) % 3
	if pre > 0 {
		b.WriteString(s[:pre])
		if len(s) > pre {
			b.WriteByte(',')
		}
	}
	for j := pre; j < len(s); j += 3 {
		b.WriteString(s[j : j+3])
		if j+3 < len(s) {
			b.WriteByte(',')
		}
	}
	out := b.String()
	if neg {
		return "-" + out
	}
	return out
}

// formatSignedCredits forces an explicit +/- prefix for ledger amounts.
func formatSignedCredits(n float64) string {
	if int64(math.Round(n)) < 0 {
		return formatCredits(n) // formatCredits already prefixes "-"
	}
	return "+" + formatCredits(n)
}

// shortTime trims an ISO-8601 timestamp to "YYYY-MM-DD HH:MM" for table display.
func shortTime(iso string) string {
	if len(iso) >= 16 {
		return strings.Replace(iso[:16], "T", " ", 1)
	}
	return iso
}
