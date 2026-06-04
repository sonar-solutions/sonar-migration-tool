// Copyright (C) SonarSource Sàrl
// For more information, see https://sonarsource.com/legal/
// mailto:info AT sonarsource DOT com

package migrate

import (
	"encoding/json"
	"testing"
)

// FlexibleBool drives the `issue-sync` config field (#299). The config
// file owner is a non-developer SQS operator, so we accept the natural
// boolean aliases (true/false, on/off, yes/no, 1/0) in any case.
func TestFlexibleBool_AcceptedForms(t *testing.T) {
	wrap := func(raw string) string { return `{"v":` + raw + `}` }
	cases := []struct {
		in    string
		wantV bool
	}{
		// JSON bare forms.
		{wrap("true"), true},
		{wrap("false"), false},
		{wrap("1"), true},
		{wrap("0"), false},
		// JSON string forms with case variations.
		{wrap(`"true"`), true},
		{wrap(`"FALSE"`), false},
		{wrap(`"on"`), true},
		{wrap(`"Off"`), false},
		{wrap(`"yes"`), true},
		{wrap(`"NO"`), false},
		{wrap(`"  Yes  "`), true}, // surrounding whitespace tolerated
	}
	for _, c := range cases {
		var s struct {
			V FlexibleBool `json:"v"`
		}
		if err := json.Unmarshal([]byte(c.in), &s); err != nil {
			t.Errorf("%s: unexpected error: %v", c.in, err)
			continue
		}
		if !s.V.Set {
			t.Errorf("%s: expected Set=true", c.in)
		}
		if s.V.Value != c.wantV {
			t.Errorf("%s: want %v, got %v", c.in, c.wantV, s.V.Value)
		}
	}
}

func TestFlexibleBool_NullIsAbsent(t *testing.T) {
	var s struct {
		V FlexibleBool `json:"v"`
	}
	if err := json.Unmarshal([]byte(`{"v": null}`), &s); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.V.Set {
		t.Error("null must leave Set=false")
	}
}

func TestFlexibleBool_Rejects(t *testing.T) {
	// Anything that isn't one of the supported aliases must error,
	// not silently default — operators need to notice typos.
	cases := []string{
		`{"v": "maybe"}`,
		`{"v": "truthy"}`,
		`{"v": 2}`,
		`{"v": []}`,
	}
	for _, c := range cases {
		var s struct {
			V FlexibleBool `json:"v"`
		}
		if err := json.Unmarshal([]byte(c), &s); err == nil {
			t.Errorf("%s: expected error, got nil", c)
		}
	}
}
