package hbs

import "testing"

func TestValidAndVariables(t *testing.T) {
	vars, err := Validate(`<h1>{{title}}</h1>{{#each items}}{{name}}{{/each}}`)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"title": true, "items": true, "name": true}
	for _, v := range vars {
		delete(want, v)
	}
	if len(want) != 0 {
		t.Fatalf("missing vars: %v (got %v)", want, vars)
	}
}

func TestInvalidSyntax(t *testing.T) {
	if _, err := Validate(`{{#each items}}no close`); err == nil {
		t.Fatal("expected parse error")
	}
}

func TestBackendCustomHelpersParse(t *testing.T) {
	// the backend registers ifEq/formatDate/formatCurrency/gt — these must not fail local validation
	if _, err := Validate(`{{#ifEq a b}}x{{/ifEq}} {{formatDate d}} {{formatCurrency m}}`); err != nil {
		t.Fatalf("custom helpers must parse: %v", err)
	}
}
