# Beats Roadmap

- [ ] Refactor beats domain model: storage domain, issue types domain, business logic domain
- [ ] Add `comment` as an event type and support assembling a comments stream attached to issues
- [ ] Refactor `BlockedBy` and `BlockReason` into more generic dependencies ('blocks','precedes','duplicates','fixes','parent',... )
- [ ] Encode `Epic` logic (data = sum of children if children exist, transition guards, ...) and enforce in flow
- [ ] Beats `boards` command that starts UI with
  - [ ] Dashboard view
  - [ ] Backlog view
  - [ ] Board view
  - [ ] Dependencies view
  - [ ] Pending changes buffer and sync to `issues.db` on "Save"
- [ ] Support / Enforce walkthroughs as a requirement for closing an issue (allow override with --force)
