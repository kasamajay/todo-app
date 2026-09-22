package handlers

import "testing"

func TestParseDueDate(t *testing.T) {
	cases := []struct {
		name    string
		raw     string
		wantNil bool
		wantErr bool
	}{
		{name: "empty string means no due date", raw: "", wantNil: true},
		{name: "whitespace only means no due date", raw: "   ", wantNil: true},
		{name: "valid RFC3339", raw: "2026-09-25T00:00:00Z"},
		{name: "date-only is not valid RFC3339", raw: "2026-09-25", wantErr: true},
		{name: "garbage string", raw: "not-a-date", wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseDueDate(tc.raw)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("parseDueDate(%q): expected error, got nil", tc.raw)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseDueDate(%q): unexpected error: %v", tc.raw, err)
			}
			if tc.wantNil && got != nil {
				t.Fatalf("parseDueDate(%q): expected nil, got %v", tc.raw, *got)
			}
			if !tc.wantNil && got == nil {
				t.Fatalf("parseDueDate(%q): expected non-nil result", tc.raw)
			}
		})
	}
}
