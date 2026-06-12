// Package hbs provides ADVISORY Handlebars validation. The backend's JS
// Handlebars.compile is authoritative; this only catches obvious syntax
// errors locally.
package hbs

import (
	"regexp"

	"github.com/aymerick/raymond"
)

// knownHelpers lists the custom helpers registered by the mkpdfs backend
// (and raymond built-ins) that should not be reported as template variables.
var knownHelpers = map[string]bool{
	"ifEq":           true,
	"gt":             true,
	"formatDate":     true,
	"formatCurrency": true,
	// raymond built-ins
	"each":   true,
	"if":     true,
	"unless": true,
	"with":   true,
	"else":   true,
	"this":   true,
}

var varRe = regexp.MustCompile(`{{[#/]?\s*(?:each\s+|if\s+|unless\s+|with\s+)?([A-Za-z_][A-Za-z0-9_.]*)`)

// Validate parses the template and returns the referenced variable names.
// raymond.Parse only catches structural syntax errors (e.g. unclosed blocks);
// it does not reject unknown helpers — consistent with advisory-only semantics.
func Validate(content string) ([]string, error) {
	if _, err := raymond.Parse(content); err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var vars []string
	for _, m := range varRe.FindAllStringSubmatch(content, -1) {
		name := m[1]
		if knownHelpers[name] || seen[name] {
			continue
		}
		seen[name] = true
		vars = append(vars, name)
	}
	return vars, nil
}
