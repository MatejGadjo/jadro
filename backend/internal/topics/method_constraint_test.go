package topics

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Guards against the classifier emitting a category_method the database
// rejects: every CategoryMethod must appear in the latest migration that
// defines chk_articles_category_method.
func TestCategoryMethodsAllowedByDB(t *testing.T) {
	files, err := filepath.Glob("../storage/migrations/*.sql")
	if err != nil || len(files) == 0 {
		t.Fatalf("no migrations found: %v", err)
	}
	var latest string
	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		up := strings.SplitN(string(raw), "-- +goose Down", 2)[0]
		if strings.Contains(up, "chk_articles_category_method CHECK") {
			latest = up
		}
	}
	for _, m := range []CategoryMethod{
		CategoryMethodForce, CategoryMethodRSSCategory, CategoryMethodURLRule,
		CategoryMethodKeywordRule, CategoryMethodFallback,
	} {
		if !strings.Contains(latest, "'"+string(m)+"'") {
			t.Errorf("category method %q is not allowed by chk_articles_category_method", m)
		}
	}
}
