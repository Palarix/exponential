package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
)

// envTestValues holds a valid value for every config key the environment
// can set. TestEnvBindings_CoverEveryField fails when a field is added to
// Config without a value here.
var envTestValues = map[string]string{
	"name":                       "Env Project",
	"prefix":                     "envp",
	"user":                       "Env User <env@example.com>",
	"default_labels":             "bug,feature",
	"hide_default_labels":        "true",
	"editor":                     "nano",
	"auto_commit":                "true",
	"style.theme":                "dark",
	"version":                    "3",
	"estimation_system":          "linear",
	"count_unestimated":          "false",
	"automations.first_start":    "true",
	"automations.last_completed": "true",
	"cycles.enabled":             "true",
	"cycles.duration":            "2w",
	"cycles.start_day":           "monday",
	"cycles.anchor_date":         "2026-01-05",
	"contributors":               "Ann <ann@example.com>,Bob <bob@example.com>",
	"remote.url":                 "https://env.example.com",
	"remote.token":               "env-token",
	"drive.supervisor.agent":     "codex",
	"drive.supervisor.model":     "opus",
	"drive.coder.agent":          "codex",
	"drive.coder.model":          "haiku",
	"drive.max_retries":          "7",
	"drive.test_cmd":             "make check",
	"drive.timeout":              "45m",
	"chat.agent":                 "claude",
	"chat.model":                 "sonnet",
	"worktree_setup":             "make setup",
	"default_branch":             "trunk",
}

func envName(path string) string {
	return "XPO_" + strings.ToUpper(strings.ReplaceAll(path, ".", "_"))
}

// envProject creates an empty project with a fresh HOME and user config
// dir, so no file sets any key.
func envProject(t *testing.T, configYAML string) {
	t.Helper()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".xpo"), 0755)
	if configYAML != "" {
		os.WriteFile(filepath.Join(dir, ".xpo", "config.yaml"), []byte(configYAML), 0644)
	}
	t.Setenv("HOME", filepath.Join(dir, "home"))
	t.Setenv("XPO_CONFIG_DIR", filepath.Join(dir, "home", "xpo"))
	wd, _ := os.Getwd()
	os.Chdir(dir)
	t.Cleanup(func() { os.Chdir(wd) })
}

// fieldByPath returns the Config field at a dotted mapstructure path.
func fieldByPath(v reflect.Value, path string) (reflect.Value, bool) {
	for _, part := range strings.Split(path, ".") {
		found := false
		for i := 0; i < v.NumField(); i++ {
			if strings.Split(v.Type().Field(i).Tag.Get("mapstructure"), ",")[0] == part {
				v, found = v.Field(i), true
				break
			}
		}
		if !found {
			return reflect.Value{}, false
		}
	}
	return v, true
}

func TestEnvBindings_CoverEveryField(t *testing.T) {
	keys := envBindableKeys(reflect.TypeOf(Config{}), "")
	for _, k := range keys {
		if _, ok := envTestValues[k]; !ok {
			t.Errorf("config key %q has no entry in envTestValues", k)
		}
	}
	for k := range envTestValues {
		if !slices.Contains(keys, k) {
			t.Errorf("envTestValues has %q, which is not a bindable config key", k)
		}
	}
	for _, mapKey := range []string{"labels", "permissions.roles", "permissions.users"} {
		if slices.Contains(keys, mapKey) {
			t.Errorf("map key %q must not be bound to the environment", mapKey)
		}
	}
}

// envCompanions are keys a key needs to pass validation on its own.
var envCompanions = map[string][]string{
	"cycles.enabled": {"cycles.duration", "cycles.start_day"},
}

// Each key is set on its own: with several XPO_* variables set at once,
// Viper sometimes picks up unbound keys anyway, which would hide the bug.
func TestLoadConfig_EnvSetsEveryKey(t *testing.T) {
	for path, want := range envTestValues {
		t.Run(path, func(t *testing.T) {
			envProject(t, "")
			t.Setenv(envName(path), want)
			for _, c := range envCompanions[path] {
				t.Setenv(envName(c), envTestValues[c])
			}

			cfg, err := LoadConfig()
			if err != nil {
				t.Fatalf("LoadConfig: %v", err)
			}
			f, ok := fieldByPath(reflect.ValueOf(*cfg), path)
			if !ok {
				t.Fatalf("no field for %s", path)
			}
			var got string
			if f.Kind() == reflect.Slice {
				parts := make([]string, f.Len())
				for i := range parts {
					parts[i] = fmt.Sprint(f.Index(i).Interface())
				}
				got = strings.Join(parts, ",")
			} else {
				got = fmt.Sprint(f.Interface())
			}
			if got != want {
				t.Errorf("%s=%q: field is %q", envName(path), want, got)
			}
		})
	}
}

func TestLoadConfig_EnvOverridesFile(t *testing.T) {
	envProject(t, "remote:\n  url: https://file.example.com\nchat:\n  model: opus\n")
	t.Setenv("XPO_REMOTE_URL", "https://env.example.com")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Remote.URL != "https://env.example.com" {
		t.Errorf("remote.url = %q, want the env value", cfg.Remote.URL)
	}
	if cfg.Chat.Model != "opus" {
		t.Errorf("chat.model = %q, want the file value when no env var is set", cfg.Chat.Model)
	}
}

func TestLoadConfig_DefaultsWithoutEnv(t *testing.T) {
	envProject(t, "")
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Prefix != "issue" || cfg.EstimationSystem != "fibonacci" || cfg.Remote.URL != "" || cfg.Drive.MaxRetries != 3 {
		t.Errorf("defaults not applied: prefix=%q estimation=%q remote=%q retries=%d",
			cfg.Prefix, cfg.EstimationSystem, cfg.Remote.URL, cfg.Drive.MaxRetries)
	}
}

func TestLoadConfig_MapEnvVarsIgnored(t *testing.T) {
	envProject(t, "labels:\n  Bug: \"#ff0000\"\n")
	t.Setenv("XPO_LABELS", "junk")
	t.Setenv("XPO_PERMISSIONS_ROLES", "junk")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig with map env vars set: %v", err)
	}
	if cfg.Labels["Bug"] != "#ff0000" {
		t.Errorf("labels = %v, want the file value", cfg.Labels)
	}
}
