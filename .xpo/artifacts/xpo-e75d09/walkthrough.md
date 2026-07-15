# Walkthrough: CycleID never validated during create or update

## What changed

One file: `internal/exponential/client.go`.

## ValidateCreatePayload

Added after the estimate validation block. `CreatePayload.CycleID` is a plain `string`:

```go
if p.CycleID != "" {
    if !c.Config.Cycles.Enabled {
        return fmt.Errorf("cycles are not enabled")
    }
    if _, err := c.Config.Cycles.CycleForID(p.CycleID); err != nil {
        return err
    }
}
```

## ValidateUpdatePayload

Same logic but for `*string`:

```go
if p.CycleID != nil && *p.CycleID != "" {
    if !c.Config.Cycles.Enabled {
        return fmt.Errorf("cycles are not enabled")
    }
    if _, err := c.Config.Cycles.CycleForID(*p.CycleID); err != nil {
        return err
    }
}
```

`CycleForID` parses the ID as `YYYY-MM-DD` via `time.Parse("2006-01-02", id)` and returns an error for malformed dates. It then resolves the date to the cycle that contains it using `cycleForDate`.

Empty cycle_id values pass through without validation — this allows clearing a cycle assignment (nil `*string` = not provided, empty string = clear).
