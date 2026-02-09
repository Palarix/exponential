# Beats Roadmap

- [x] Implement a comprehensive test suite
- [x] Introduce a data model and CLI version
- [x] Add data model version to config.yml
- [x] Add `version` subcommand to print CLI version
- [x] Have `doctor` subcommand flag version mismatch between data model in current CLI and on disk
- [x] Add `migrate` subcommand to upgrade `issues.db` to match latest data model
- [x] Refactor beats domain model: storage domain, issue types domain, business logic domain
- [x] Refactor `BlockedBy` and `BlockReason` into more generic dependencies ('blocks','precedes','duplicates','fixes','parent',... )
- [x] Encode `Epic` logic (data = sum of children if children exist, transition guards, ...) and enforce in flow
- [ ] Add `comment` as an event type and support assembling a comments stream attached to issues
- [ ] Beats `boards` command that starts UI with
  - [ ] Dashboard view
  - [ ] Backlog view
  - [ ] Board view
  - [ ] Dependencies view
  - [ ] Pending changes buffer and sync to `issues.db` on "Save"
- [ ] Support / Enforce walkthroughs as a requirement for closing an issue (allow override with --force)
