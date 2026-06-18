package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/sim4gh/mkpdfs-cli/internal/api"
)

// fakeAPI is a canned creditsAPI: each method returns the queued response/err for
// its path. It mimics api.Client's contract of returning a non-nil *Response even
// on error (status >= 400).
type fakeAPI struct {
	getResp  *api.Response
	getErr   error
	putResp  *api.Response
	putErr   error
	postResp *api.Response
	postErr  error

	lastPutBody any
}

func (f *fakeAPI) Get(path string) (*api.Response, error) { return f.getResp, f.getErr }
func (f *fakeAPI) Put(path string, body any) (*api.Response, error) {
	f.lastPutBody = body
	return f.putResp, f.putErr
}
func (f *fakeAPI) Post(path string, body any) (*api.Response, error) { return f.postResp, f.postErr }

func resp(status int, body string) *api.Response {
	return &api.Response{StatusCode: status, Body: json.RawMessage(body)}
}

// --- formatting ---

func TestFormatCredits(t *testing.T) {
	cases := map[float64]string{
		0: "0", 100: "100", 1240: "1,240", 1000000: "1,000,000", -30: "-30", -1500: "-1,500",
	}
	for in, want := range cases {
		if got := formatCredits(in); got != want {
			t.Errorf("formatCredits(%v) = %q, want %q", in, got, want)
		}
	}
}

func TestFormatSignedCredits(t *testing.T) {
	cases := map[float64]string{1000: "+1,000", -200: "-200", 0: "+0"}
	for in, want := range cases {
		if got := formatSignedCredits(in); got != want {
			t.Errorf("formatSignedCredits(%v) = %q, want %q", in, got, want)
		}
	}
}

func TestCreditBalanceStringEnterprise(t *testing.T) {
	if got := creditBalanceString(subscriptionInfo{Plan: "enterprise", CreditBalance: 5}); got != "unlimited" {
		t.Errorf("enterprise = %q, want unlimited", got)
	}
	if got := creditBalanceString(subscriptionInfo{Plan: "credits", CreditBalance: 1240}); got != "1,240" {
		t.Errorf("credits = %q, want 1,240", got)
	}
}

// --- balance ---

func TestCreditsBalance(t *testing.T) {
	f := &fakeAPI{getResp: resp(200, `{"data":{"subscription":{"plan":"credits","creditBalance":1240,"autoRecharge":true,"rechargeThreshold":100}}}`)}
	var out bytes.Buffer
	if err := creditsBalance(f, &out, false); err != nil {
		t.Fatal(err)
	}
	s := out.String()
	for _, want := range []string{"1,240 remaining", "Auto-recharge:  on", "balance < 100", "Plan:           credits"} {
		if !strings.Contains(s, want) {
			t.Errorf("balance output missing %q\n%s", want, s)
		}
	}
}

func TestCreditsBalanceEnterprise(t *testing.T) {
	f := &fakeAPI{getResp: resp(200, `{"data":{"subscription":{"plan":"enterprise","creditBalance":0}}}`)}
	var out bytes.Buffer
	if err := creditsBalance(f, &out, false); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "unlimited remaining") {
		t.Errorf("want unlimited, got\n%s", out.String())
	}
}

func TestCreditsBalanceNegativeAndError(t *testing.T) {
	f := &fakeAPI{getResp: resp(200, `{"data":{"subscription":{"plan":"credits","creditBalance":-30,"autoRechargeError":"card declined"}}}`)}
	var out bytes.Buffer
	if err := creditsBalance(f, &out, false); err != nil {
		t.Fatal(err)
	}
	s := out.String()
	if !strings.Contains(s, "-30 remaining") {
		t.Errorf("want negative balance, got\n%s", s)
	}
	if !strings.Contains(s, "Auto-recharge failed: card declined") {
		t.Errorf("want autorecharge error banner, got\n%s", s)
	}
	if !strings.Contains(s, "Buy credits to continue") {
		t.Errorf("want negative-balance hint, got\n%s", s)
	}
}

func TestCreditsBalanceJSON(t *testing.T) {
	f := &fakeAPI{getResp: resp(200, `{"data":{"subscription":{"plan":"credits","creditBalance":500,"autoRecharge":false,"rechargeThreshold":100}}}`)}
	var out bytes.Buffer
	if err := creditsBalance(f, &out, true); err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("not valid JSON: %v\n%s", err, out.String())
	}
	if got["creditBalance"].(float64) != 500 || got["plan"].(string) != "credits" {
		t.Errorf("unexpected json: %v", got)
	}
}

// --- ledger ---

func TestCreditsLedgerPopulated(t *testing.T) {
	body := `{"entries":[
		{"entryId":"a","type":"purchase","amount":1000,"createdAt":"2026-06-15T10:30:00Z"},
		{"entryId":"b","type":"debit","amount":-3,"balanceAfter":997,"description":"pdf_generation","createdAt":"2026-06-16T09:00:00Z"}
	]}`
	f := &fakeAPI{getResp: resp(200, body)}
	var out bytes.Buffer
	if err := creditsLedger(f, &out, false); err != nil {
		t.Fatal(err)
	}
	s := out.String()
	for _, want := range []string{"+1,000", "-3", "997", "pdf_generation", "2026-06-15 10:30"} {
		if !strings.Contains(s, want) {
			t.Errorf("ledger missing %q\n%s", want, s)
		}
	}
}

func TestCreditsLedgerEmpty(t *testing.T) {
	f := &fakeAPI{getResp: resp(200, `{"entries":[]}`)}
	var out bytes.Buffer
	if err := creditsLedger(f, &out, false); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "No ledger entries yet.") {
		t.Errorf("want empty message, got\n%s", out.String())
	}
}

func TestCreditsLedgerFiftyFooter(t *testing.T) {
	var entries []string
	for i := 0; i < 50; i++ {
		entries = append(entries, `{"entryId":"x","type":"debit","amount":-1,"createdAt":"2026-06-16T09:00:00Z"}`)
	}
	f := &fakeAPI{getResp: resp(200, `{"entries":[`+strings.Join(entries, ",")+`]}`)}
	var out bytes.Buffer
	if err := creditsLedger(f, &out, false); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "(showing most recent 50)") {
		t.Errorf("want 50-footer, got\n%s", out.String())
	}
}

// --- auto-recharge ---

func TestValidateAutoRechargeFlags(t *testing.T) {
	cases := []struct {
		name                           string
		enable, disable, threshChanged bool
		threshold                      int
		wantErr                        bool
	}{
		{"show (no flags)", false, false, false, 0, false},
		{"enable", true, false, false, 0, false},
		{"enable+threshold ok", true, false, true, 100, false},
		{"disable", false, true, false, 0, false},
		{"both enable+disable", true, true, false, 0, true},
		{"threshold without enable", false, false, true, 100, true},
		{"disable+threshold", false, true, true, 100, true},
		{"threshold too low", true, false, true, 0, true},
		{"threshold too high", true, false, true, 1001, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateAutoRechargeFlags(tc.enable, tc.disable, tc.threshChanged, tc.threshold)
			if tc.wantErr {
				if err == nil {
					t.Fatal("want error, got nil")
				}
				if !errors.Is(err, ErrUsage) {
					t.Fatalf("want ErrUsage, got %v", err)
				}
			} else if err != nil {
				t.Fatalf("want nil, got %v", err)
			}
		})
	}
}

func TestAutoRechargeEnable(t *testing.T) {
	f := &fakeAPI{putResp: resp(200, `{"autoRecharge":true,"rechargeThreshold":200}`)}
	var out bytes.Buffer
	thr := 200
	if err := autoRechargeSet(f, &out, false, true, &thr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Auto-recharge enabled (threshold 200)") {
		t.Errorf("got\n%s", out.String())
	}
	// request body carried the threshold pointer
	req := f.lastPutBody.(autoRechargeRequest)
	if !req.Enabled || req.Threshold == nil || *req.Threshold != 200 {
		t.Errorf("bad request body: %+v", req)
	}
}

func TestAutoRechargeDisable(t *testing.T) {
	f := &fakeAPI{putResp: resp(200, `{"autoRecharge":false,"rechargeThreshold":100}`)}
	var out bytes.Buffer
	if err := autoRechargeSet(f, &out, false, false, nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Auto-recharge disabled") {
		t.Errorf("got\n%s", out.String())
	}
}

func TestAutoRechargeNoPaymentMethod(t *testing.T) {
	f := &fakeAPI{
		putResp: resp(400, `{"success":false,"error":"NO_PAYMENT_METHOD","message":"x"}`),
		putErr:  errors.New("generic api error"),
	}
	var out bytes.Buffer
	err := autoRechargeSet(f, &out, false, true, nil)
	if err == nil || !strings.Contains(err.Error(), "Buy a credit pack first") {
		t.Fatalf("want NO_PAYMENT_METHOD guidance, got %v", err)
	}
}

func TestAutoRechargeShow(t *testing.T) {
	f := &fakeAPI{getResp: resp(200, `{"data":{"subscription":{"autoRecharge":true,"rechargeThreshold":100}}}`)}
	var out bytes.Buffer
	if err := autoRechargeShow(f, &out, false); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Auto-recharge is on (threshold 100)") {
		t.Errorf("got\n%s", out.String())
	}
}

// --- buy ---

func TestCreditsBuyOpensBrowser(t *testing.T) {
	orig := openURL
	defer func() { openURL = orig }()
	var opened string
	openURL = func(u string) error { opened = u; return nil }

	f := &fakeAPI{postResp: resp(200, `{"success":true,"url":"https://checkout.stripe.com/x","sessionId":"cs_1"}`)}
	var out bytes.Buffer
	if err := creditsBuy(f, &out, false); err != nil {
		t.Fatal(err)
	}
	if opened != "https://checkout.stripe.com/x" {
		t.Errorf("browser opened %q", opened)
	}
	if !strings.Contains(out.String(), "https://checkout.stripe.com/x") {
		t.Errorf("URL not printed:\n%s", out.String())
	}
}

func TestCreditsBuyJSONDoesNotOpen(t *testing.T) {
	orig := openURL
	defer func() { openURL = orig }()
	called := false
	openURL = func(u string) error { called = true; return nil }

	f := &fakeAPI{postResp: resp(200, `{"success":true,"url":"https://checkout.stripe.com/x","sessionId":"cs_1"}`)}
	var out bytes.Buffer
	if err := creditsBuy(f, &out, true); err != nil {
		t.Fatal(err)
	}
	if called {
		t.Error("--json must not open the browser")
	}
	if !strings.Contains(out.String(), `"url":"https://checkout.stripe.com/x"`) {
		t.Errorf("raw JSON expected:\n%s", out.String())
	}
}

func TestCreditsBuyBrowserErrorStillSucceeds(t *testing.T) {
	orig := openURL
	defer func() { openURL = orig }()
	openURL = func(u string) error { return errors.New("no browser") }

	f := &fakeAPI{postResp: resp(200, `{"success":true,"url":"https://checkout.stripe.com/x"}`)}
	var out bytes.Buffer
	if err := creditsBuy(f, &out, false); err != nil {
		t.Fatalf("browser failure must not fail the command: %v", err)
	}
	if !strings.Contains(out.String(), "couldn't open your browser") {
		t.Errorf("want fallback note, got\n%s", out.String())
	}
}
