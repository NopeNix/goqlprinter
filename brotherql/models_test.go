package brotherql

import (
	"testing"
)

func TestGetModel_KnownModels(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                   string
		wantRasterWidthBytes   int
		wantSupportsSwitchMode bool
	}{
		{"QL-500", 90, false},
		{"QL-700", 90, true},
		{"QL-800", 90, true},
		{"QL-1050", 162, true},
		{"QL-1110NWB", 162, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			model, err := GetModel(tc.name)
			if err != nil {
				t.Errorf("GetModel(%q) returned unexpected error: %v", tc.name, err)
			}
			if model.Name != tc.name {
				t.Errorf("GetModel(%q).Name = %q, want %q", tc.name, model.Name, tc.name)
			}
			if model.RasterWidthBytes != tc.wantRasterWidthBytes {
				t.Errorf("GetModel(%q).RasterWidthBytes = %d, want %d", tc.name, model.RasterWidthBytes, tc.wantRasterWidthBytes)
			}
			if model.SupportsSwitchMode != tc.wantSupportsSwitchMode {
				t.Errorf("GetModel(%q).SupportsSwitchMode = %v, want %v", tc.name, model.SupportsSwitchMode, tc.wantSupportsSwitchMode)
			}
		})
	}
}

func TestGetModel_WideModelsHave162Bytes(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"QL-1050", "QL-1110NWB"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			model, err := GetModel(name)
			if err != nil {
				t.Fatalf("GetModel(%q) returned unexpected error: %v", name, err)
			}
			if model.RasterWidthBytes != 162 {
				t.Errorf("GetModel(%q).RasterWidthBytes = %d, want 162", name, model.RasterWidthBytes)
			}
		})
	}
}

func TestGetModel_UnknownModelReturnsDefaultAndError(t *testing.T) {
	t.Parallel()

	model, err := GetModel("QL-UNKNOWN")
	if err == nil {
		t.Error("GetModel(unknown) expected non-nil error, got nil")
	}
	if model.Name != "QL-800" {
		t.Errorf("GetModel(unknown).Name = %q, want QL-800", model.Name)
	}
	if model.RasterWidthBytes != 90 {
		t.Errorf("GetModel(unknown).RasterWidthBytes = %d, want 90", model.RasterWidthBytes)
	}
}

func TestGetLabel_KnownIDs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		id             string
		wantIsDieCut   bool
		wantFeedMargin int
	}{
		{"62", false, 35},
		{"29", false, 35},
		{"17x54", true, 0},
		{"d58", true, 0},
	}

	for _, tc := range tests {
		t.Run(tc.id, func(t *testing.T) {
			t.Parallel()
			label, err := GetLabel(tc.id)
			if err != nil {
				t.Fatalf("GetLabel(%q) returned unexpected error: %v", tc.id, err)
			}
			if label.ID != tc.id {
				t.Errorf("GetLabel(%q).ID = %q, want %q", tc.id, label.ID, tc.id)
			}
			if label.IsDieCut != tc.wantIsDieCut {
				t.Errorf("GetLabel(%q).IsDieCut = %v, want %v", tc.id, label.IsDieCut, tc.wantIsDieCut)
			}
			if label.FeedMargin != tc.wantFeedMargin {
				t.Errorf("GetLabel(%q).FeedMargin = %d, want %d", tc.id, label.FeedMargin, tc.wantFeedMargin)
			}
		})
	}
}

func TestGetLabel_UnknownIDReturnsError(t *testing.T) {
	t.Parallel()

	_, err := GetLabel("notexist")
	if err == nil {
		t.Error("GetLabel(unknown) expected error, got nil")
	}
}

func TestListLabels_NonEmpty(t *testing.T) {
	t.Parallel()

	labels := ListLabels()
	if len(labels) == 0 {
		t.Error("ListLabels() returned empty slice, want non-empty")
	}
}

func TestListLabels_UniqueIDs(t *testing.T) {
	t.Parallel()

	labels := ListLabels()
	seen := make(map[string]bool)
	for _, label := range labels {
		if seen[label.ID] {
			t.Errorf("ListLabels() contains duplicate ID: %q", label.ID)
		}
		seen[label.ID] = true
	}
}

func TestListLabels_EndlessTapeHasPositiveFeedMargin(t *testing.T) {
	t.Parallel()

	for _, label := range ListLabels() {
		if !label.IsDieCut {
			t.Run(label.ID, func(t *testing.T) {
				t.Parallel()
				if label.FeedMargin <= 0 {
					t.Errorf("Endless tape label %q has FeedMargin=%d, want > 0", label.ID, label.FeedMargin)
				}
			})
		}
	}
}

func TestListLabels_DieCutLabelsHaveZeroFeedMargin(t *testing.T) {
	t.Parallel()

	for _, label := range ListLabels() {
		// d12 is the documented exception
		if label.IsDieCut && label.ID != "d12" {
			t.Run(label.ID, func(t *testing.T) {
				t.Parallel()
				if label.FeedMargin != 0 {
					t.Errorf("Die-cut label %q has FeedMargin=%d, want 0", label.ID, label.FeedMargin)
				}
			})
		}
	}
}

func TestListLabels_PrintableWidthFitsInTotal(t *testing.T) {
	t.Parallel()

	for _, label := range ListLabels() {
		t.Run(label.ID, func(t *testing.T) {
			t.Parallel()
			if label.DotsPrintableWidth > label.DotsTotalWidth {
				t.Errorf("Label %q: DotsPrintableWidth(%d) > DotsTotalWidth(%d)",
					label.ID, label.DotsPrintableWidth, label.DotsTotalWidth)
			}
		})
	}
}
