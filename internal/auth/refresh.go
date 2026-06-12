package auth

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sim4gh/mkpdfs-cli/internal/config"
	"github.com/sim4gh/mkpdfs-cli/internal/envs"
)

const cognitoEndpoint = "https://cognito-idp.us-east-1.amazonaws.com/"

// RefreshTokens exchanges the stored refresh token for fresh id/access tokens
// using InitiateAuth REFRESH_TOKEN_AUTH (plain JSON API, no Hosted UI).
func RefreshTokens(env envs.Env) (*config.Creds, error) {
	cfg := config.Get()
	creds := cfg.Creds(env.Name)
	if creds.RefreshToken == "" {
		return nil, errors.New(`not authenticated. Run "mkp auth login"`)
	}

	body, _ := json.Marshal(map[string]any{
		"AuthFlow": "REFRESH_TOKEN_AUTH",
		"ClientId": env.ClientID,
		"AuthParameters": map[string]string{
			"REFRESH_TOKEN": creds.RefreshToken,
		},
	})
	req, err := http.NewRequest("POST", cognitoEndpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "AWSCognitoIdentityProviderService.InitiateAuth")

	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf(`session expired. Run "mkp auth login" (%s)`, string(raw))
	}

	var out struct {
		AuthenticationResult struct {
			IdToken     string `json:"IdToken"`
			AccessToken string `json:"AccessToken"`
		} `json:"AuthenticationResult"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	creds.IDToken = out.AuthenticationResult.IdToken
	creds.AccessToken = out.AuthenticationResult.AccessToken
	if err := config.SetConfig(cfg); err != nil {
		return nil, err
	}
	return creds, nil
}
