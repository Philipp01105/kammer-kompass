package ihk_catalog

import "testing"

func TestCatalogHasExpectedEntries(t *testing.T) {
	entries := All()
	if len(entries) != 79 {
		t.Fatalf("expected 79 IHK catalog entries, got %d", len(entries))
	}

	slugs := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		if entry.Slug == "" || entry.Name == "" || entry.City == "" || entry.State == "" || entry.OfficialURL == "" {
			t.Fatalf("catalog entry has empty required field: %+v", entry)
		}
		if _, ok := slugs[entry.Slug]; ok {
			t.Fatalf("duplicate slug %q", entry.Slug)
		}
		slugs[entry.Slug] = struct{}{}
	}
}
