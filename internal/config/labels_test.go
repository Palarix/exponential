package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupConfigDir(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".beats"), 0755)
	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	t.Cleanup(func() { os.Chdir(oldWd) })
}

func writeConfig(t *testing.T, content string) {
	t.Helper()
	os.WriteFile(filepath.Join(".beats", "config.yaml"), []byte(content), 0644)
}

func readConfig(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(".beats", "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestAddLabel_NewToExisting(t *testing.T) {
	setupConfigDir(t)
	writeConfig(t, "prefix: test-\nversion: 2\nlabels:\n  bug: \"#ff0000\"\n")

	err := AddLabel("feature", "#00ff00")
	if err != nil {
		t.Fatal(err)
	}

	got := readConfig(t)
	if !strings.Contains(got, "feature") || !strings.Contains(got, "#00ff00") {
		t.Errorf("label not added: %s", got)
	}
	if !strings.Contains(got, "bug") {
		t.Error("existing label should be preserved")
	}
}

func TestAddLabel_CreatesLabelsSection(t *testing.T) {
	setupConfigDir(t)
	writeConfig(t, "prefix: test-\nversion: 2\n")

	err := AddLabel("bug", "#ff0000")
	if err != nil {
		t.Fatal(err)
	}

	got := readConfig(t)
	if !strings.Contains(got, "labels:") || !strings.Contains(got, "bug") {
		t.Errorf("labels section not created: %s", got)
	}
}

func TestAddLabel_NoConfigFile(t *testing.T) {
	setupConfigDir(t)

	err := AddLabel("bug", "#ff0000")
	if err != nil {
		t.Fatal(err)
	}

	got := readConfig(t)
	if !strings.Contains(got, "bug") {
		t.Errorf("should create config with label: %s", got)
	}
}

func TestAddLabel_UpdateExisting(t *testing.T) {
	setupConfigDir(t)
	writeConfig(t, "prefix: test-\nlabels:\n  bug: \"#ff0000\"\n")

	AddLabel("bug", "#00ff00")

	got := readConfig(t)
	if !strings.Contains(got, "#00ff00") {
		t.Errorf("color should be updated: %s", got)
	}
	if strings.Count(got, "bug") != 1 {
		t.Errorf("should not duplicate label key: %s", got)
	}
}

func TestDeleteLabel_Existing(t *testing.T) {
	setupConfigDir(t)
	writeConfig(t, "prefix: test-\nlabels:\n  bug: \"#ff0000\"\n  feature: \"#00ff00\"\n")

	err := DeleteLabel("bug")
	if err != nil {
		t.Fatal(err)
	}

	got := readConfig(t)
	if strings.Contains(got, "bug") {
		t.Errorf("bug should be deleted: %s", got)
	}
	if !strings.Contains(got, "feature") {
		t.Error("other labels should be preserved")
	}
}

func TestDeleteLabel_NonExistent(t *testing.T) {
	setupConfigDir(t)
	writeConfig(t, "prefix: test-\nlabels:\n  bug: \"#ff0000\"\n")

	err := DeleteLabel("nonexistent")
	if err != nil {
		t.Fatal("deleting non-existent label should not error")
	}
}

func TestDeleteLabel_NoConfigFile(t *testing.T) {
	setupConfigDir(t)
	err := DeleteLabel("bug")
	if err != nil {
		t.Fatal("missing config should not error")
	}
}

func TestUpdateLabel_Rename(t *testing.T) {
	setupConfigDir(t)
	writeConfig(t, "prefix: test-\nlabels:\n  bug: \"#ff0000\"\n")

	err := UpdateLabel("bug", "defect", "#ee0000")
	if err != nil {
		t.Fatal(err)
	}

	got := readConfig(t)
	if strings.Contains(got, "bug") {
		t.Errorf("old name should be removed: %s", got)
	}
	if !strings.Contains(got, "defect") || !strings.Contains(got, "#ee0000") {
		t.Errorf("new name and color should be present: %s", got)
	}
}

func TestUpdateLabel_ColorOnly(t *testing.T) {
	setupConfigDir(t)
	writeConfig(t, "prefix: test-\nlabels:\n  bug: \"#ff0000\"\n")

	err := UpdateLabel("bug", "bug", "#00ff00")
	if err != nil {
		t.Fatal(err)
	}

	got := readConfig(t)
	if !strings.Contains(got, "#00ff00") {
		t.Errorf("color should be updated: %s", got)
	}
}
