package model

import (
	"errors"
	"testing"
)

func TestCategoryLabel(t *testing.T) {
	if got := CategoryFood.Label(); got != "еда" {
		t.Fatalf("unexpected food label %q", got)
	}
	if got := CategoryPersonalCare.Label(); got != "уход за собой" {
		t.Fatalf("unexpected personal care label %q", got)
	}
	if got := Category("unknown").Label(); got != "unknown" {
		t.Fatalf("unexpected unknown category label %q", got)
	}
}

func TestCategories(t *testing.T) {
	categories := Categories()
	if len(categories) != 14 {
		t.Fatalf("expected 14 categories, got %d", len(categories))
	}
	categories[0] = CategoryOther
	if allCategories[0] != CategoryFood {
		t.Fatal("Categories must return a copy")
	}
}

func TestParseCategory(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    Category
		wantErr bool
	}{
		{name: "valid", value: "food", want: CategoryFood},
		{name: "normalizes case and spaces", value: "  TRANSPORT ", want: CategoryTransport},
		{name: "additional category", value: "personal_care", want: CategoryPersonalCare},
		{name: "unknown", value: "crypto", wantErr: true},
		{name: "empty", value: "", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ParseCategory(test.value)
			if test.wantErr {
				if !errors.Is(err, ErrValidation) {
					t.Fatalf("expected validation error, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("parse category: %v", err)
			}
			if got != test.want {
				t.Fatalf("expected %q, got %q", test.want, got)
			}
		})
	}
}
