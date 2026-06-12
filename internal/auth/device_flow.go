package auth

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// DeviceAuthResponse is the payload returned by POST /auth/cli/device.
// Field names match the live backend exactly.
type DeviceAuthResponse struct {
	DeviceCode      string `json:"deviceCode"`
	UserCode        string `json:"userCode"`
	VerificationURI string `json:"verificationUri"`
	ExpiresIn       int    `json:"expiresIn"`
	Interval        int    `json:"interval"`
}

// DeviceTokenResponse is the payload returned by POST /auth/cli/token.
// On success (200) the token fields are populated; on error the Error field
// carries the error code (authorization_pending, slow_down, access_denied,
// expired_token).
type DeviceTokenResponse struct {
	IDToken      string `json:"idToken"`
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	Error        string `json:"error"`
}

// slowDownStep is the additional wait added to the polling interval each time
// the server returns "slow_down". It is a package-level var so tests can
// override it to zero to avoid sleeping.
var slowDownStep = 5 * time.Second

// InitiateDeviceAuth starts the device-authorization flow by calling
// POST {apiBase}/auth/cli/device and returning the server's response.
func InitiateDeviceAuth(apiBase string) (*DeviceAuthResponse, error) {
	resp, err := (&http.Client{Timeout: 30 * time.Second}).Post(
		apiBase+"/auth/cli/device", "application/json", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, errors.New("failed to start login: " + string(raw))
	}
	var out DeviceAuthResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// PollForToken repeatedly calls POST {apiBase}/auth/cli/token until the user
// approves, denies, or the code expires. intervalSec is the initial wait
// between attempts (0 is valid for tests). On "slow_down" the interval grows
// by slowDownStep.
func PollForToken(apiBase, deviceCode string, intervalSec int) (*DeviceTokenResponse, error) {
	interval := time.Duration(intervalSec) * time.Second
	body, _ := json.Marshal(map[string]string{"deviceCode": deviceCode})
	client := &http.Client{Timeout: 30 * time.Second}

	for {
		time.Sleep(interval)
		resp, err := client.Post(apiBase+"/auth/cli/token", "application/json", bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		raw, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		var out DeviceTokenResponse
		if err := json.Unmarshal(raw, &out); err != nil {
			return nil, err
		}
		if resp.StatusCode == 200 {
			return &out, nil
		}
		switch out.Error {
		case "authorization_pending":
			continue
		case "slow_down":
			interval += slowDownStep
			continue
		case "access_denied":
			return nil, errors.New("login denied in the browser")
		case "expired_token":
			return nil, errors.New(`the code expired. Run "mkp auth login" again`)
		default:
			return nil, fmt.Errorf("login failed: %s", string(raw))
		}
	}
}
