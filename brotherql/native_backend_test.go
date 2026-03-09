package brotherql

import "testing"

func TestIsBrotherPrinter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want bool
	}{
		{"QL-570", true},
		{"QL-800", true},
		{"ql-800", true},
		{"Brother QL-800", true},
		{"BROTHER QL-1100", true},
		{"Some random printer", false},
		{"HP LaserJet", false},
		{"", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := isBrotherPrinter(tc.name)
			if got != tc.want {
				t.Errorf("isBrotherPrinter(%q) = %v, want %v", tc.name, got, tc.want)
			}
		})
	}
}

func TestExtractModel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"Brother QL-800", "QL-800"},
		{"QL-570", "QL-570"},
		{"ql-1100", "ql-1100"},
		{"BROTHER QL-820NWB Extra", "QL-820NWB"},
		{"Some random printer", "Unknown"},
		{"", "Unknown"},
		{"prefix ql-700", "ql-700"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			got := extractModel(tc.input)
			if got != tc.want {
				t.Errorf("extractModel(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}
