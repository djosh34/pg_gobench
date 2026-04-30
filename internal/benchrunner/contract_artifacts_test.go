package benchrunner_test

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

func TestLiveBenchmarkSchemaContractArtifactsUseBench(t *testing.T) {
	root := repoRoot(t)
	files := []string{
		".ralph/tasks/story-03-core-benchmark/task-01-benchmark-schema-scale.md",
		".ralph/tasks/story-03-core-benchmark/task-01-benchmark-schema-scale_plans/2026-04-30-benchmark-schema-scale-plan.md",
		".ralph/tasks/story-06-advanced-workloads/task-01-join-lock-contention-workloads_plans/2026-04-30-join-lock-contention-workloads-plan.md",
		".ralph/tasks/story-99-manual-verify-everything/task-01-manual-verify-everything.md",
	}
	forbidden := []*regexp.Regexp{
		regexp.MustCompile("schema named `pg_gobench`"),
		regexp.MustCompile("`pg_gobench` schema"),
		regexp.MustCompile("schema `pg_gobench`"),
		regexp.MustCompile(`pg_gobench\.(accounts|branches|history|tellers)`),
	}

	for _, rel := range files {
		t.Run(rel, func(t *testing.T) {
			contentBytes, err := os.ReadFile(filepath.Join(root, rel))
			if err != nil {
				t.Fatalf("ReadFile(%q): %v", rel, err)
			}
			content := string(contentBytes)
			if !strings.Contains(content, "`bench`") {
				t.Fatalf("%s does not describe the live benchmark schema as `bench`", rel)
			}
			for _, pattern := range forbidden {
				if match := pattern.FindString(content); match != "" {
					t.Fatalf("%s contains stale benchmark schema contract %q", rel, match)
				}
			}
		})
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
