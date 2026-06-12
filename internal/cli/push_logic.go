package cli

import (
	"fmt"

	"github.com/sim4gh/mkpdfs-cli/internal/localmap"
)

type pushAction int

const (
	pushCreate pushAction = iota
	pushUpdate
)

type pushInput struct {
	File            string
	Map             *localmap.Map
	ActiveEnv       string
	UserID          string // current logged-in user
	RemoteUpdatedAt string // fetched from API when entry known; "" if unknown/new
	Force           bool
	ForceID         string // --id
	ForceNew        bool   // --new
}

type pushDecision struct {
	Action     pushAction
	TemplateID string
}

func decidePush(in pushInput) (pushDecision, error) {
	if in.Map.Environment != "" && in.Map.Environment != in.ActiveEnv {
		return pushDecision{}, fmt.Errorf(
			".mkpdfs.json is bound to %q but the active environment is %q — no cross-env writes. Use --env %s or a different directory: %w",
			in.Map.Environment, in.ActiveEnv, in.Map.Environment, ErrUsage)
	}
	if in.ForceID != "" {
		return pushDecision{Action: pushUpdate, TemplateID: in.ForceID}, nil
	}
	entry, known := in.Map.Templates[localmap.Key(in.File)]
	if in.ForceNew || !known {
		return pushDecision{Action: pushCreate}, nil
	}
	if in.Map.UserID != "" && in.Map.UserID != in.UserID && !in.Force {
		return pushDecision{}, fmt.Errorf(
			".mkpdfs.json was created by another account (%s). Use --force to push anyway: %w",
			in.Map.UserID, ErrUsage)
	}
	if entry.RemoteUpdatedAt != "" && in.RemoteUpdatedAt != "" &&
		entry.RemoteUpdatedAt != in.RemoteUpdatedAt && !in.Force {
		return pushDecision{}, fmt.Errorf(
			"remote template changed since your last sync (remote %s, local record %s). Pull first or push --force: %w",
			in.RemoteUpdatedAt, entry.RemoteUpdatedAt, ErrUsage)
	}
	return pushDecision{Action: pushUpdate, TemplateID: entry.TemplateID}, nil
}
