package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/palarix/exponential/internal/model"
)

type persona struct {
	Name  string
	Email string
}

func (p persona) Identity() string {
	return fmt.Sprintf("%s <%s>", p.Name, p.Email)
}

var personas = []persona{
	{"Sarah Chen", "sarah@megacorp.io"},
	{"Jake Morrison", "jake@megacorp.io"},
	{"Luna Garcia", "luna@megacorp.io"},
	{"Claude", "agent@claude.ai"},
	{"Marcus Lee", "marcus@megacorp.io"},
	{"Priya Patel", "priya@megacorp.io"},
	{"Alex Kim", "alex@megacorp.io"},
	{"Devin", "agent@devin.ai"},
	{"River Santos", "river@megacorp.io"},
	{"Nyx", "nyx@agents.megacorp.io"},
	{"Taylor Okafor", "taylor@megacorp.io"},
	{"Jordan Mills", "jordan@megacorp.io"},
}

var labels = []string{
	"bug", "feature", "epic", "improvement", "backend", "frontend",
	"mobile", "infra", "design", "security", "performance", "docs",
	"testing", "devops", "api", "database", "ux", "a11y",
}

var epicTitles = []string{
	"User Authentication & SSO", "API Gateway v2", "Billing & Subscriptions",
	"Mobile App Redesign", "Search Infrastructure", "Analytics Pipeline",
	"CI/CD Overhaul", "Database Migration", "Microservice Decomposition",
	"Performance Optimization", "Security Audit Remediation", "Accessibility Compliance",
	"Design System v3", "Notification Engine", "Content Management",
	"Localization & i18n", "Data Export & Reporting", "Third-Party Integrations",
	"Feature Flags System", "Observability Stack", "Event Sourcing Migration",
	"GraphQL Gateway", "Rate Limiting & Throttling", "Caching Layer v2",
	"Webhook Delivery System", "Admin Dashboard Rewrite", "Multi-Tenancy Support",
	"Audit Logging", "File Storage Service", "Real-Time Collaboration",
}

var featureVerbs = []string{
	"Implement", "Build", "Add", "Create", "Design", "Set up",
	"Configure", "Wire up", "Integrate", "Scaffold",
}

var featureNouns = []string{
	"user profile endpoint", "settings page", "dashboard widget",
	"export functionality", "import pipeline", "webhook handler",
	"cron job scheduler", "retry mechanism", "circuit breaker",
	"health check probe", "metrics collector", "log aggregator",
	"cache invalidation", "batch processor", "queue consumer",
	"rate limiter", "auth middleware", "request validator",
	"response serializer", "error handler", "data transformer",
	"notification dispatcher", "event emitter", "state machine",
	"migration script", "seed data generator", "test fixture",
	"API client", "SDK wrapper", "CLI command",
	"form validation", "file upload handler", "image resizer",
	"PDF generator", "email template", "SMS gateway",
	"search indexer", "autocomplete engine", "filter builder",
	"sort comparator", "pagination cursor", "bulk updater",
	"permission checker", "role manager", "session handler",
	"token refresher", "password hasher", "2FA enrollment",
	"audit trail recorder", "changelog generator", "version bumper",
	"dependency checker", "license scanner", "vulnerability scanner",
	"load balancer config", "DNS record manager", "SSL cert rotator",
	"container orchestrator", "service mesh config", "feature toggle",
	"A/B test framework", "analytics tracker", "funnel reporter",
	"cohort analyzer", "retention calculator", "churn predictor",
}

var bugPrefixes = []string{
	"Fix", "Resolve", "Patch", "Debug", "Correct",
}

var bugNouns = []string{
	"null pointer in user service", "race condition in order processing",
	"memory leak in WebSocket handler", "timeout in payment callback",
	"incorrect pagination offset", "broken sort on dashboard",
	"missing index on orders table", "CORS error on staging",
	"stale cache after bulk update", "duplicate events in analytics",
	"flaky test in auth suite", "CSS overflow on mobile",
	"timezone handling in scheduler", "encoding issue in export",
	"deadlock in batch processor", "permission bypass in admin API",
	"infinite loop in retry logic", "truncated response in search",
	"missing validation on input", "wrong status code on 404",
	"session leak on logout", "broken redirect after OAuth",
	"slow query in reporting", "N+1 in listing endpoint",
	"wrong decimal precision in billing", "lost message in queue",
	"misaligned columns in PDF report", "broken dark mode toggle",
	"scroll jump on page load", "focus trap in modal",
}

var improvementPrefixes = []string{
	"Optimize", "Refactor", "Improve", "Simplify", "Upgrade",
	"Clean up", "Modernize", "Consolidate", "Streamline",
}

var improvementNouns = []string{
	"database query performance", "API response time", "build pipeline",
	"test coverage for auth module", "error messages in checkout",
	"logging verbosity in prod", "dependency versions",
	"Docker image size", "startup time for workers",
	"memory usage in report generator", "code duplication in handlers",
	"type safety in API layer", "documentation for onboarding",
	"monitoring alerts threshold", "backup rotation policy",
	"connection pool sizing", "batch job throughput",
	"frontend bundle size", "image lazy loading",
	"form UX on mobile", "keyboard navigation",
}

var commentTemplates = []string{
	"Investigated the root cause — turns out %s was the issue. Fixed by %s.",
	"This is more complex than expected. The main challenge is %s. Will need another day.",
	"Deployed to staging. Looking good so far. Need to monitor %s before promoting.",
	"Code review feedback addressed. Main change: %s.",
	"Blocked on %s — waiting for the dependency to be resolved.",
	"Unblocked. Proceeding with %s.",
	"Found an edge case: %s. Adding a test for it.",
	"Performance benchmarks look solid: %s.",
	"Design review complete. Adjusting %s per feedback.",
	"Tested with %s — all green. Ready for final review.",
	"Merged the prerequisite PR. Now tackling %s.",
	"Pairing with %s on this — the domain logic is tricky.",
	"Splitting this into smaller PRs for easier review. Starting with %s.",
	"Ran load tests: %s. Within acceptable thresholds.",
	"Updated the migration to handle %s gracefully.",
	"CI is green after fixing the flaky %s test.",
	"Rolled back the initial attempt — %s caused regressions. Taking a different approach.",
	"This touches a lot of shared code. Added %s to prevent breakage.",
	"Security review passed. %s was flagged but deemed acceptable risk.",
	"Added feature flag so we can %s without blocking the release.",
}

var commentDetails = []string{
	"the connection pool exhaustion under load",
	"a subtle race between the cache invalidation and the read path",
	"the schema migration for the new columns",
	"incorrect assumptions about the input format",
	"the retry backoff configuration",
	"the WebSocket reconnection logic",
	"the token refresh timing window",
	"the batch size for bulk operations",
	"latency in the p95 bucket",
	"the queue depth during peak hours",
	"the index strategy for the new query pattern",
	"the serialization format for events",
	"memory allocation in the hot path",
	"the validation rules for the new field",
	"edge cases in the date parsing",
	"cross-browser compatibility issues",
	"the error boundary in the React tree",
	"a missing foreign key constraint",
	"the service discovery timeout",
	"the gRPC deadline propagation",
}

func main() {
	rng := rand.New(rand.NewSource(42))

	outputDir := "stresstest"
	if len(os.Args) > 1 {
		outputDir = os.Args[1]
	}
	xpoDir := filepath.Join(outputDir, ".xpo")
	os.MkdirAll(xpoDir, 0755)

	now := time.Now().UTC()
	anchor := now.Add(-730 * 24 * time.Hour) // project started 2 years ago

	var events []model.Event
	issueCount := 0
	commentCount := 0

	targetEvents := 100_000
	numEpics := 30
	numIssuesPerEpic := 150 // ~4500 issues in epics + ~500 orphans

	epicIDs := make([]string, numEpics)
	for i := 0; i < numEpics; i++ {
		issueCount++
		id := fmt.Sprintf("st-%06x", issueCount)
		epicIDs[i] = id

		title := epicTitles[i%len(epicTitles)]
		if i >= len(epicTitles) {
			title = fmt.Sprintf("%s (Phase %d)", title, i/len(epicTitles)+1)
		}

		ts := anchor.Add(time.Duration(rng.Intn(200)) * 24 * time.Hour)
		actor := personas[rng.Intn(len(personas))]

		events = append(events, model.Event{
			ID:        id,
			Type:      model.EventTypeCreate,
			CreatedAt: ts,
			CreatedBy: actor.Identity(),
			Payload: model.CreatePayload{
				Title:       title,
				Description: fmt.Sprintf("Epic tracking all work related to %s.", title),
				Status:      "PLANNED",
				Labels:      []string{"epic"},
				Estimate:    0,
			},
		})

		// Epic status transitions
		if rng.Float64() < 0.8 {
			events = append(events, model.Event{
				ID:        id,
				Type:      model.EventTypeUpdate,
				CreatedAt: ts.Add(time.Duration(rng.Intn(30)+5) * 24 * time.Hour),
				CreatedBy: actor.Identity(),
				Payload:   updateStatus("DOING"),
			})
		}
	}

	allIssueIDs := make([]string, 0, numEpics*numIssuesPerEpic+500)
	allIssueIDs = append(allIssueIDs, epicIDs...)

	// Generate issues under epics
	for _, epicID := range epicIDs {
		count := numIssuesPerEpic + rng.Intn(40) - 20 // 130-170 per epic
		for j := 0; j < count; j++ {
			issueCount++
			id := fmt.Sprintf("st-%06x", issueCount)
			allIssueIDs = append(allIssueIDs, id)

			offsetDays := rng.Intn(700)
			ts := anchor.Add(time.Duration(offsetDays) * 24 * time.Hour)
			ts = ts.Add(time.Duration(rng.Intn(10)) * time.Hour)
			actor := personas[rng.Intn(len(personas))]

			title, issueLabels := randomIssueTitle(rng)

			est := []int{0, 1, 2, 3, 5, 8}[rng.Intn(6)]

			events = append(events, model.Event{
				ID:        id,
				Type:      model.EventTypeCreate,
				CreatedAt: ts,
				CreatedBy: actor.Identity(),
				Payload: model.CreatePayload{
					Title:    title,
					Status:   "BACKLOG",
					ParentID: epicID,
					Estimate: est,
					Labels:   issueLabels,
					Assignee: personas[rng.Intn(len(personas))].Identity(),
				},
			})
		}
	}

	// Orphan issues (no parent)
	for i := 0; i < 500; i++ {
		issueCount++
		id := fmt.Sprintf("st-%06x", issueCount)
		allIssueIDs = append(allIssueIDs, id)

		offsetDays := rng.Intn(700)
		ts := anchor.Add(time.Duration(offsetDays) * 24 * time.Hour)
		actor := personas[rng.Intn(len(personas))]

		title, issueLabels := randomIssueTitle(rng)
		est := []int{0, 1, 2, 3, 5}[rng.Intn(5)]

		events = append(events, model.Event{
			ID:        id,
			Type:      model.EventTypeCreate,
			CreatedAt: ts,
			CreatedBy: actor.Identity(),
			Payload: model.CreatePayload{
				Title:    title,
				Status:   "BACKLOG",
				Estimate: est,
				Labels:   issueLabels,
				Assignee: personas[rng.Intn(len(personas))].Identity(),
			},
		})
	}

	fmt.Printf("Created %d issues (%d events so far)\n", issueCount, len(events))

	// Now generate lifecycle events until we hit 100k
	// Shuffle issue IDs so the activity distribution is random
	shuffled := make([]string, len(allIssueIDs))
	copy(shuffled, allIssueIDs)
	rng.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })

	idx := 0
	for len(events) < targetEvents {
		id := shuffled[idx%len(shuffled)]
		idx++

		// Each round adds 1-6 events for an issue
		baseOffset := rng.Intn(700)
		baseTS := anchor.Add(time.Duration(baseOffset) * 24 * time.Hour)
		actor := personas[rng.Intn(len(personas))]
		hourOffset := 0

		// Status transitions
		statuses := pickStatusPath(rng)
		for _, st := range statuses {
			if len(events) >= targetEvents {
				break
			}
			hourOffset += rng.Intn(48) + 2
			events = append(events, model.Event{
				ID:        id,
				Type:      model.EventTypeUpdate,
				CreatedAt: baseTS.Add(time.Duration(hourOffset) * time.Hour),
				CreatedBy: actor.Identity(),
				Payload:   updateStatus(st),
			})
		}

		// Comments
		numComments := rng.Intn(4)
		for c := 0; c < numComments; c++ {
			if len(events) >= targetEvents {
				break
			}
			hourOffset += rng.Intn(24) + 1
			commentCount++
			events = append(events, model.Event{
				ID:        id,
				Type:      model.EventTypeComment,
				CreatedAt: baseTS.Add(time.Duration(hourOffset) * time.Hour),
				CreatedBy: personas[rng.Intn(len(personas))].Identity(),
				Payload: model.CommentPayload{
					ID:   fmt.Sprintf("cmt-%06d", commentCount),
					Text: randomComment(rng),
				},
			})
		}

		// Occasional dependency links
		if rng.Float64() < 0.1 && len(events) < targetEvents {
			targetIdx := rng.Intn(len(allIssueIDs))
			targetID := allIssueIDs[targetIdx]
			if targetID != id {
				hourOffset += rng.Intn(12) + 1
				events = append(events, model.Event{
					ID:        id,
					Type:      model.EventTypeUpdate,
					CreatedAt: baseTS.Add(time.Duration(hourOffset) * time.Hour),
					CreatedBy: actor.Identity(),
					Payload: model.UpdatePayload{
						Dependencies: []model.Dependency{
							{SourceID: id, TargetID: targetID, Kind: model.DependencyDependsOn},
						},
					},
				})
			}
		}

		// Occasional label/estimate updates
		if rng.Float64() < 0.15 && len(events) < targetEvents {
			hourOffset += rng.Intn(24) + 1
			newEst := []int{1, 2, 3, 5, 8, 13}[rng.Intn(6)]
			events = append(events, model.Event{
				ID:        id,
				Type:      model.EventTypeUpdate,
				CreatedAt: baseTS.Add(time.Duration(hourOffset) * time.Hour),
				CreatedBy: actor.Identity(),
				Payload: model.UpdatePayload{
					Estimate: &newEst,
				},
			})
		}
	}

	// Trim to exactly targetEvents
	events = events[:targetEvents]

	// Sort by timestamp
	sort.Slice(events, func(i, j int) bool {
		return events[i].CreatedAt.Before(events[j].CreatedAt)
	})

	// Write issues.db
	dbPath := filepath.Join(xpoDir, "issues.db")
	f, err := os.Create(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating issues.db: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	encoder := json.NewEncoder(f)
	encoder.SetEscapeHTML(false)
	for _, evt := range events {
		if err := encoder.Encode(evt); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing event: %v\n", err)
			os.Exit(1)
		}
	}

	// Write config.yaml
	writeConfig(xpoDir)

	fi, _ := os.Stat(dbPath)
	fmt.Printf("Done: %d events, %d issues, %d comments\n", len(events), issueCount, commentCount)
	fmt.Printf("File: %s (%.1f MB)\n", dbPath, float64(fi.Size())/(1024*1024))
}

func randomIssueTitle(rng *rand.Rand) (string, []string) {
	roll := rng.Float64()
	if roll < 0.25 {
		// Bug
		prefix := bugPrefixes[rng.Intn(len(bugPrefixes))]
		noun := bugNouns[rng.Intn(len(bugNouns))]
		return fmt.Sprintf("%s %s", prefix, noun), []string{"bug", labels[rng.Intn(len(labels)-3)+3]}
	} else if roll < 0.7 {
		// Feature
		verb := featureVerbs[rng.Intn(len(featureVerbs))]
		noun := featureNouns[rng.Intn(len(featureNouns))]
		return fmt.Sprintf("%s %s", verb, noun), []string{"feature", labels[rng.Intn(len(labels)-3)+3]}
	} else {
		// Improvement
		prefix := improvementPrefixes[rng.Intn(len(improvementPrefixes))]
		noun := improvementNouns[rng.Intn(len(improvementNouns))]
		return fmt.Sprintf("%s %s", prefix, noun), []string{"improvement", labels[rng.Intn(len(labels)-3)+3]}
	}
}

func pickStatusPath(rng *rand.Rand) []string {
	roll := rng.Float64()
	if roll < 0.4 {
		return []string{"PLANNED", "DOING", "DONE"}
	} else if roll < 0.6 {
		return []string{"PLANNED", "DOING"}
	} else if roll < 0.75 {
		return []string{"PLANNED"}
	} else if roll < 0.85 {
		return []string{"PLANNED", "DOING", "BLOCKED", "DOING", "DONE"}
	} else if roll < 0.95 {
		return []string{"PLANNED", "DOING", "DONE"}
	} else {
		return nil // stays BACKLOG
	}
}

func randomComment(rng *rand.Rand) string {
	tmpl := commentTemplates[rng.Intn(len(commentTemplates))]
	d1 := commentDetails[rng.Intn(len(commentDetails))]
	d2 := commentDetails[rng.Intn(len(commentDetails))]
	return fmt.Sprintf(tmpl, d1, d2)
}

func updateStatus(status string) model.UpdatePayload {
	s := status
	return model.UpdatePayload{Status: &s}
}

func writeConfig(xpoDir string) {
	cfg := `name: MegaCorp Platform
prefix: st-
version: 3
estimation_system: fibonacci
count_unestimated: true
automations:
    first_start: false
    last_completed: false
labels:
    bug: "#eb5757"
    feature: "#b36cd9"
    epic: "#5e6ad2"
    improvement: "#4da6e8"
    backend: "#e07058"
    frontend: "#26b5b0"
    mobile: "#e06091"
    design: "#d4a030"
    infra: "#6b7280"
    security: "#f59e0b"
    performance: "#10b981"
    docs: "#8b5cf6"
    testing: "#06b6d4"
    devops: "#f97316"
    api: "#ec4899"
    database: "#14b8a6"
    ux: "#a855f7"
    a11y: "#84cc16"
cycles:
    enabled: true
    duration: 2w
    start_day: monday
contributors:
    - "Sarah Chen <sarah@megacorp.io>"
    - "Jake Morrison <jake@megacorp.io>"
    - "Luna Garcia <luna@megacorp.io>"
    - "Claude <agent@claude.ai>"
    - "Marcus Lee <marcus@megacorp.io>"
    - "Priya Patel <priya@megacorp.io>"
    - "Alex Kim <alex@megacorp.io>"
    - "Devin <agent@devin.ai>"
    - "River Santos <river@megacorp.io>"
    - "Nyx <nyx@agents.megacorp.io>"
    - "Taylor Okafor <taylor@megacorp.io>"
    - "Jordan Mills <jordan@megacorp.io>"
`
	os.WriteFile(filepath.Join(xpoDir, "config.yaml"), []byte(cfg), 0644)
}
