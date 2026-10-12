# Walkthrough: XPO_* environment variables set every config key

## What was wrong

The README says `XPO_` environment variables override both config files. That was only true for keys with a default. `LoadConfig` called `v.AutomaticEnv()`, but Viper only checks the environment during `Unmarshal` for keys it already knows about: keys with a default, keys in a loaded config file, or keys registered with `BindEnv`. `remote.url` has no default, so `XPO_REMOTE_URL` did nothing unless a config file already set `remote.url`. 17 keys had this problem, including `remote.url`, `remote.token`, `chat.model` and `default_branch`.

We found it while testing xpo-a60065. `XPO_REMOTE_URL=… xpo board` stayed in local mode.

## The fix

`internal/config/config.go`:

- `envBindableKeys(t reflect.Type, prefix string) []string` walks `Config` by its `mapstructure` tags. It recurses into nested structs (`remote`, `drive.supervisor`, …), returns the dotted path of every scalar and slice field, and skips maps.
- `LoadConfig` calls `v.BindEnv(key)` for each of those keys right after `AutomaticEnv()`. `SetEnvPrefix("XPO")` and the `.`→`_` key replacer are already set up, so `BindEnv("remote.url")` binds `XPO_REMOTE_URL`.

Reflection means a new field is covered as soon as it is added to `Config`, without anyone having to remember a list.

### Why maps are skipped

`labels`, `permissions.roles` and `permissions.users` are maps. One env string cannot express a map, and binding one would turn a stray `XPO_LABELS` into a decode error that breaks every command. These keys still need a config file, and the README now says so.

### Slices

`default_labels` and `contributors` are `[]string`. Viper's default decode hook splits a string on commas, so `XPO_DEFAULT_LABELS=bug,feature` gives `["bug","feature"]` with no extra code.

## Tests

`internal/config/env_test.go` (new):

- `envTestValues` maps every bindable key to a valid env value.
- `TestEnvBindings_CoverEveryField` compares that map to `envBindableKeys` in both directions, and checks that no map key is bound. A new `Config` field without a test value fails here.
- `TestLoadConfig_EnvSetsEveryKey` runs one subtest per key in an empty project (temp dir, fresh `HOME` and `XPO_CONFIG_DIR`). It sets that key's variable and checks the field through reflection. `cycles.enabled=true` fails validation unless `cycles.duration` and `cycles.start_day` are set too, so `envCompanions` sets them alongside it.
- `TestLoadConfig_EnvOverridesFile`: env beats file for `remote.url`, and the file value of `chat.model` survives when its variable is unset.
- `TestLoadConfig_DefaultsWithoutEnv`: defaults still apply.
- `TestLoadConfig_MapEnvVarsIgnored`: `XPO_LABELS=junk` and `XPO_PERMISSIONS_ROLES=junk` load cleanly and leave the file's labels alone.

### Non-obvious: one key per subtest

The first version set all 31 variables at once and passed **without** the fix. With many `XPO_*` variables set, Viper sometimes picks up unbound keys anyway, nondeterministically. Setting one key per subtest removes that effect. With `BindEnv` commented out, exactly the 17 keys without a default fail, which is the bug. Don't merge the subtests back into one.

## README

The Configuration section now explains the naming rule (`XPO_` plus the key in upper case, `.` → `_`), gives `XPO_REMOTE_URL` and `XPO_CHAT_MODEL` as examples, says lists are comma-separated, and says map keys need a config file.

## Acceptance criteria

- [x] Every scalar field can be set from `XPO_<PATH>` with no config file mentioning it. **Evidence:** `TestLoadConfig_EnvSetsEveryKey`, 31 subtests, all pass. Live: `XPO_REMOTE_URL=http://127.0.0.1:1 xpo board` logs remote mode, and `/api/local/mode` returns `{"remote":true}`.
- [x] `[]string` fields take comma-separated values. **Evidence:** the `default_labels` and `contributors` subtests (`bug,feature`, two contributors).
- [x] Map fields are not bound, and setting one neither changes the config nor breaks loading. **Evidence:** `TestEnvBindings_CoverEveryField` (map-key check) and `TestLoadConfig_MapEnvVarsIgnored`.
- [x] Env overrides a file value, and the file value applies when the env var is unset. **Evidence:** `TestLoadConfig_EnvOverridesFile`.
- [x] Defaults apply when neither a file nor the env sets a key. **Evidence:** `TestLoadConfig_DefaultsWithoutEnv`.
- [x] A test fails when a new scalar or slice field cannot be set from the environment. **Evidence:** `TestEnvBindings_CoverEveryField`. Removing the fix makes 17 subtests of `TestLoadConfig_EnvSetsEveryKey` fail.
- [x] The README documents which keys the environment can set and that map keys cannot. **Evidence:** the README Configuration section.
- [x] `make test` passes.

## Side effect to know about

Any `XPO_<KEY>` variable in someone's shell now takes effect for every key, not just those with defaults. That is the documented behavior. xpo's own non-config variables (`XPO_CONFIG_DIR`, `XPO_AGENT_IDENTITY`) don't collide with any config key.
