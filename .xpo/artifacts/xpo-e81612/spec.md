# Spec: bind every config key to its XPO_* environment variable

## What

`LoadConfig` binds an `XPO_*` environment variable to every scalar and slice field of `Config`, found by walking the struct's `mapstructure` tags. Every documented key can then be set from the environment, not only the keys with a default.

## Why

`AutomaticEnv` only affects `Unmarshal` for keys Viper can list from defaults, config files or `BindEnv`. For example, `XPO_REMOTE_URL` is ignored unless a config file already has `remote.url`. The README promises that `XPO_` variables override both config files.

## Acceptance Criteria

- [ ] For every scalar field of `Config`, setting its `XPO_<PATH>` variable sets the field when no config file mentions the key. Example: `XPO_REMOTE_URL`, `XPO_CHAT_MODEL`, `XPO_DEFAULT_BRANCH`.
- [ ] `[]string` fields accept a comma-separated value, for example `XPO_DEFAULT_LABELS=bug,feature`.
- [ ] Map fields (`labels`, `permissions.roles`, `permissions.users`) are not bound. A set `XPO_LABELS` neither changes the config nor breaks loading.
- [ ] Env values still override config-file values, and config-file values still apply when the env var is unset.
- [ ] Defaults still apply when neither a file nor the env sets a key.
- [ ] A test walks `Config` and fails when a new scalar or slice field cannot be set from the environment.
- [ ] The README says which keys the environment can set and that map keys cannot.
- [ ] `make test` passes.

## Flow

1. Write the tests first, in a new file `internal/config/env_test.go`:
   - `envTestValues` holds a valid env value for every bindable key. `TestEnvBindings_CoverEveryField` fails when a `Config` field has no entry, or an entry names no field.
   - `TestLoadConfig_EnvSetsEveryKey` runs one subtest per key. Each sets only that key's variable, plus any companions validation needs (`cycles.enabled` needs `cycles.duration` and `cycles.start_day`), loads the config in an empty project, and checks the field.
   - Override order: a file value plus an env value gives the env value. A file value alone gives the file value.
   - Defaults apply with no env set.
   - `XPO_LABELS=junk` and `XPO_PERMISSIONS_ROLES=junk` load without error and leave `labels` unchanged.
2. Add `envBindableKeys(t reflect.Type, prefix string) []string` in `internal/config/config.go`. It recurses into struct fields, building the dotted path from each `mapstructure` tag, returns each scalar or slice path, and skips map fields.
3. In `LoadConfig`, after `AutomaticEnv()`, call `v.BindEnv` for each key and return its error.
4. Update the README Configuration section: env names are `XPO_` plus the dotted key in upper case, with `.` replaced by `_`. Lists are comma-separated. Map keys (`labels`, `permissions`) need a config file.

## Decisions

| Question | Decision | Rationale |
|---|---|---|
| How to register keys | Reflect over `Config` and call `BindEnv` | New fields are covered automatically; no experimental Viper API. |
| Map fields | Not bound | One env string cannot express a map. Binding them would make a stray variable a load error. |
| Slices | Bound, comma-separated | Viper's default decode hooks split strings on commas. |
| `version` | Bound like every other key | The README promises all keys. No current `XPO_` variable collides. |
| One key per subtest | Each subtest sets a single variable | With many `XPO_*` variables set at once, Viper sometimes picks up unbound keys anyway, so an all-at-once test passed without the fix. One key at a time fails for all 17 keys without a default when the fix is removed. |

## Edge Cases

- **MEDIUM — a stray env var now takes effect.** For example, an `XPO_NAME` in someone's shell now sets the project name. This is the documented behavior. No `XPO_` variable used by xpo itself (`XPO_CONFIG_DIR`, `XPO_AGENT_IDENTITY`) is a config key.
- **LOW — a bad value for a typed field** (for example `XPO_HIDE_DEFAULT_LABELS=maybe`). `Unmarshal` returns its existing decode error.
- **LOW — empty env value.** Viper treats an empty variable as unset by default, so it does not blank out a file value.
