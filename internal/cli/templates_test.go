package cli

import "testing"

func TestTemplatesPrefix(t *testing.T) {
	if got := templatesPrefix(false); got != "/templates" {
		t.Errorf("jwt prefix = %q", got)
	}
	if got := templatesPrefix(true); got != "/v1/templates" {
		t.Errorf("api-key prefix = %q", got)
	}
}

func TestMaxTemplateBytes(t *testing.T) {
	// 6.5 MiB, under the ~7 MB effective ceiling.
	if maxTemplateBytes != 6_815_744 {
		t.Errorf("maxTemplateBytes = %d", maxTemplateBytes)
	}
}
