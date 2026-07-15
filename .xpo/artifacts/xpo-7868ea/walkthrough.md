# Walkthrough: MCP update tool missing estimate validation

## What changed

One file: `internal/exponential/client.go`.

## ValidateUpdatePayload

Added estimate validation after the dependency resolution block:

```go
if p.Estimate != nil && *p.Estimate > 0 {
    if err := config.ValidateEstimate(c.Config.EstimationSystem, *p.Estimate); err != nil {
        return err
    }
}
if p.Estimate != nil && *p.Estimate < 0 {
    return fmt.Errorf("estimate must not be negative")
}
```

The `UpdatePayload.Estimate` is a `*int` (pointer), so nil means "not provided" (no change). Zero means "clear the estimate". Positive values are validated against the configured estimation system (e.g. Fibonacci: 1, 2, 3, 5, 8, 13). Negative values are rejected outright.

## ValidateCreatePayload

Added the same negative check for consistency. `CreatePayload.Estimate` is a plain `int`, so the existing `> 0` guard already handled the validation path, but negative values were silently accepted:

```go
if p.Estimate < 0 {
    return fmt.Errorf("estimate must not be negative")
}
```

Both the MCP tools and the `handleDraft` HTTP handler call these methods, so the fix applies to all write paths.
