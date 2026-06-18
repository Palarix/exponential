package demo

func allIssues() []seedIssue {
	var out []seedIssue
	out = append(out, epics()...)
	out = append(out, phase1Kickoff()...)
	out = append(out, phase2Sprint()...)
	out = append(out, phase3Incident()...)
	out = append(out, phase4Descope()...)
	out = append(out, phase5Refactor()...)
	out = append(out, phase6Launch()...)
	out = append(out, phase7PostLaunch()...)
	return out
}

func allEvents() []seedEvent {
	var out []seedEvent
	out = append(out, phase1Events()...)
	out = append(out, phase2Events()...)
	out = append(out, phase3Events()...)
	out = append(out, phase4Events()...)
	out = append(out, phase5Events()...)
	out = append(out, phase6Events()...)
	out = append(out, phase7Events()...)
	return out
}

// ──────────────────────────────────────────────────────────────────────
// Epics
// ──────────────────────────────────────────────────────────────────────

func epics() []seedIssue {
	return []seedIssue{
		{Ref: "E1", Title: "Core Platform", Desc: "Foundation services: auth, user management, API gateway, and shared infrastructure for the Upurr Eats platform.", Labels: []string{"epic"}, Actor: pSarah, Offset: 0},
		{Ref: "E2", Title: "Menu & Ordering System", Desc: "End-to-end ordering flow: restaurant menus, cat-friendly dietary filters, cart management, and order lifecycle.", Labels: []string{"epic"}, Actor: pSarah, Offset: 0},
		{Ref: "E3", Title: "Delivery & Logistics", Desc: "Real-time delivery tracking, driver assignment, route optimization, and delivery confirmation with photo proof.", Labels: []string{"epic"}, Actor: pJake, Offset: 0},
		{Ref: "E4", Title: "Payments & Billing", Desc: "Payment processing, invoicing, refunds, promotional credits, and subscription billing for Upurr Premium.", Labels: []string{"epic"}, Actor: pSarah, Offset: 1},
		{Ref: "E5", Title: "Cat Profiles & Preferences", Desc: "Cat owner and cat profiles, dietary preferences, allergy tracking, feeding schedules, and personalized recommendations.", Labels: []string{"epic"}, Actor: pLuna, Offset: 1},
		{Ref: "E6", Title: "Cat Mood Detection", Desc: "AI-powered feature to detect a cat's mood from photos and suggest comfort food accordingly. Uses vision model + custom fine-tune.\n\n**Note:** This is a stretch goal for v1. May be descoped if timeline is tight.", Labels: []string{"epic"}, Actor: pLuna, Offset: 2},
		{Ref: "E7", Title: "Tech Debt & Reliability", Desc: "Reliability improvements, monitoring, and tech debt paydown spawned from the payment incident postmortem.", Labels: []string{"epic"}, Actor: pSarah, Offset: 720},
	}
}

// ──────────────────────────────────────────────────────────────────────
// Phase 1: Project Kickoff (offset 0-168h, W17 target ~30pts)
// ──────────────────────────────────────────────────────────────────────

func phase1Kickoff() []seedIssue {
	return []seedIssue{
		// E1: Core Platform
		{Ref: "CP-1", Title: "Set up monorepo and CI pipeline", Desc: "Initialize Go monorepo structure with GitHub Actions CI. Include linting, testing, and build steps.", Labels: []string{"feature", "infra"}, Parent: "E1", Estimate: 3, Actor: pClaude, Offset: 2},
		{Ref: "CP-2", Title: "Implement JWT authentication service", Desc: "Auth service with JWT token issuance, refresh, and revocation. Support email/password and OAuth (Google, Apple).", Labels: []string{"feature", "backend"}, Parent: "E1", Estimate: 3, Actor: pJake, Offset: 3},
		{Ref: "CP-3", Title: "Design API gateway routing layer", Desc: "Kong-based API gateway with rate limiting, request logging, and service discovery for backend microservices.", Labels: []string{"feature", "backend"}, Parent: "E1", Estimate: 3, Actor: pSarah, Offset: 3},
		{Ref: "CP-4", Title: "PostgreSQL schema and migration framework", Desc: "Set up database schema with golang-migrate. Initial tables: users, cats, restaurants, menu_items.", Labels: []string{"feature", "backend"}, Parent: "E1", Estimate: 3, Actor: pDevin, Offset: 4},
		{Ref: "CP-5", Title: "Implement user registration flow", Desc: "Registration endpoint with email verification, password strength validation, and duplicate detection.", Labels: []string{"feature", "backend"}, Parent: "E1", Estimate: 3, Actor: pClaude, Offset: 24},
		{Ref: "CP-6", Title: "Set up staging environment on AWS", Desc: "ECS Fargate cluster with RDS, ElastiCache, and S3. Terraform modules for reproducible infra.", Labels: []string{"feature", "infra"}, Parent: "E1", Estimate: 3, Actor: pJake, Offset: 10},
		{Ref: "CP-7", Title: "Implement RBAC permission system", Desc: "Role-based access control: owner, admin, driver, customer roles with fine-grained permissions per API endpoint.", Labels: []string{"feature", "backend"}, Parent: "E1", Estimate: 2, Actor: pSarah, Offset: 12},
		{Ref: "CP-8", Title: "Set up centralized logging with ELK", Desc: "Elasticsearch + Kibana stack for centralized log aggregation. Structured JSON logging from all services.", Labels: []string{"feature", "infra"}, Parent: "E1", Estimate: 2, Actor: pDevin, Offset: 20},
		{Ref: "CP-9", Title: "Add health check and readiness endpoints", Desc: "Standard /healthz and /readyz endpoints for all services. Include dependency checks (DB, cache, external APIs).", Labels: []string{"improvement", "backend"}, Parent: "E1", Estimate: 1, Actor: pDevin, Offset: 30},
		{Ref: "CP-10", Title: "Create shared error handling middleware", Desc: "Unified error response format across all APIs. Structured error codes, request IDs, and client-friendly messages.", Labels: []string{"improvement", "backend"}, Parent: "E1", Estimate: 2, Actor: pClaude, Offset: 36},

		// E2: Menu & Ordering
		{Ref: "MO-1", Title: "Design menu data model", Desc: "Restaurant menu schema supporting categories, items, modifiers, dietary tags (grain-free, fish-based, etc.), and pricing tiers.", Labels: []string{"feature", "backend"}, Parent: "E2", Estimate: 3, Actor: pLuna, Offset: 6},
		{Ref: "MO-2", Title: "Build restaurant onboarding API", Desc: "CRUD endpoints for restaurant profiles: name, address, operating hours, delivery radius, and menu upload.", Labels: []string{"feature", "backend"}, Parent: "E2", Estimate: 3, Actor: pDevin, Offset: 8},
		{Ref: "MO-3", Title: "Implement menu browsing UI", Desc: "React components for browsing restaurants, filtering by dietary needs, viewing menus with photos and descriptions.", Labels: []string{"feature", "frontend"}, Parent: "E2", Estimate: 3, Actor: pLuna, Offset: 24},

		// E3: Delivery
		{Ref: "DL-1", Title: "Design delivery driver data model", Desc: "Driver profiles, vehicle info, availability status, current location, rating, and delivery history.", Labels: []string{"feature", "backend"}, Parent: "E3", Estimate: 2, Actor: pClaude, Offset: 8},

		// E4: Payments
		{Ref: "PM-1", Title: "Integrate Stripe payment processing", Desc: "Stripe integration for card payments. Support for payment intents, saved cards, and 3D Secure authentication.", Labels: []string{"feature", "backend"}, Parent: "E4", Estimate: 3, Actor: pJake, Offset: 10},

		// E5: Cat Profiles
		{Ref: "CAT-1", Title: "Cat profile creation and management", Desc: "Cat profiles with name, breed, age, weight, photo, and personality traits. Support multiple cats per household.", Labels: []string{"feature", "backend"}, Parent: "E5", Estimate: 2, Actor: pWhiskers, Offset: 12},
		{Ref: "CAT-2", Title: "Dietary preference and allergy tracking", Desc: "Per-cat dietary preferences: protein preferences, allergies (chicken, grain, dairy), and veterinarian notes.", Labels: []string{"feature", "backend"}, Parent: "E5", Estimate: 2, Actor: pClaude, Offset: 30},

		// E6: Mood Detection (stretch goal, starts research)
		{Ref: "MD-1", Title: "Research vision models for cat mood classification", Desc: "Evaluate GPT-4V, Claude Vision, and custom fine-tuned models for classifying cat moods from photos: happy, hungry, sleepy, playful, grumpy.", Labels: []string{"feature", "backend"}, Parent: "E6", Estimate: 3, Actor: pSarah, Offset: 24},

		// Infrastructure / shared work
		{Ref: "CP-11", Title: "Write integration tests for auth endpoints", Desc: "Full test suite for login, registration, token refresh, and OAuth flows. Use httptest for in-process testing.", Labels: []string{"improvement", "backend"}, Parent: "E1", Estimate: 3, Actor: pWhiskers, Offset: 40},
		{Ref: "CP-12", Title: "Create typed API client for React frontend", Desc: "Auto-generated TypeScript client from OpenAPI spec. Includes request/response types, error handling, and auth token injection.", Labels: []string{"feature", "frontend"}, Parent: "E1", Estimate: 2, Actor: pDevin, Offset: 48},
		{Ref: "CP-13", Title: "Implement request validation middleware", Desc: "JSON schema validation on all POST/PUT endpoints. Return structured error messages with field-level details.", Labels: []string{"improvement", "backend"}, Parent: "E1", Estimate: 2, Actor: pClaude, Offset: 56},
		{Ref: "CP-14", Title: "Configure Redis caching layer for menu data", Desc: "Cache restaurant menus in Redis with 15-min TTL. Invalidate on menu update. Expected to cut DB reads by 70%.", Labels: []string{"improvement", "backend"}, Parent: "E1", Estimate: 2, Actor: pDevin, Offset: 64},
		{Ref: "CP-15", Title: "Build shared UI component library", Desc: "Reusable React components: buttons, cards, modals, form inputs, loading skeletons. Storybook for documentation.", Labels: []string{"feature", "frontend"}, Estimate: 3, Actor: pLuna, Offset: 72},
		{Ref: "CP-16", Title: "Implement WebSocket infrastructure", Desc: "Shared WebSocket server with rooms/channels pattern. Used by: order tracking, driver location, restaurant order feed.", Labels: []string{"feature", "backend"}, Parent: "E1", Estimate: 3, Actor: pClaude, Offset: 80},
		{Ref: "CP-17", Title: "Set up end-to-end test framework with Playwright", Desc: "Playwright test infrastructure for critical user flows. Run in CI on every PR. Initial tests: login, browse menu, add to cart.", Labels: []string{"improvement", "frontend"}, Estimate: 3, Actor: pWhiskers, Offset: 90},
	}
}

func phase1Events() []seedEvent {
	return []seedEvent{
		// Epics move to PLANNED
		{Ref: "E1", Offset: 1, Actor: pSarah, Kind: evStatus, Status: "PLANNED"},
		{Ref: "E2", Offset: 1, Actor: pSarah, Kind: evStatus, Status: "PLANNED"},
		{Ref: "E3", Offset: 1, Actor: pJake, Kind: evStatus, Status: "PLANNED"},
		{Ref: "E4", Offset: 2, Actor: pSarah, Kind: evStatus, Status: "PLANNED"},
		{Ref: "E5", Offset: 2, Actor: pLuna, Kind: evStatus, Status: "PLANNED"},

		// Sarah's kickoff comment
		{Ref: "E1", Offset: 1, Actor: pSarah, Kind: evComment, Text: "Kickoff meeting done. We're targeting 8 weeks to MVP. Core platform is the critical path -- nothing else can ship without auth, the API gateway, and the database layer.\n\nTeam assignments:\n- Jake: Stripe integration, full-stack\n- Luna: design, product, frontend\n- Claude + Devin + Whiskers: implementation workhorse\n\nThe AI agents will handle the bulk of the feature implementation. Jake owns the critical payment path. Luna owns the user experience. I'll handle architecture decisions and the recommendation engine.\n\nLet's build something cats will love."},
		{Ref: "E2", Offset: 1, Actor: pSarah, Kind: evComment, Text: "Menu & ordering is the core product experience. We need to nail the dietary filtering -- that's our main differentiator from regular food delivery apps. Cat owners care deeply about what goes into their pet's food."},

		// CP-1: monorepo setup (agent, fast)
		{Ref: "CP-1", Offset: 3, Actor: pClaude, Kind: evStatus, Status: "PLANNED"},
		{Ref: "CP-1", Offset: 4, Actor: pClaude, Kind: evStatus, Status: "DOING"},
		{Ref: "CP-1", Offset: 8, Actor: pClaude, Kind: evComment, Text: "Monorepo structure initialized:\n\n```\n/cmd      - service entry points\n/internal - shared packages\n/web      - React frontend\n/deploy   - Terraform + Docker\n```\n\nCI pipeline runs on every PR: lint, test, build. Using GitHub Actions with Go 1.22 and Node 20 LTS. Build cache enabled for faster runs."},
		{Ref: "CP-1", Offset: 9, Actor: pClaude, Kind: evStatus, Status: "DONE"},

		// CP-4: database (agent, fast)
		{Ref: "CP-4", Offset: 5, Actor: pDevin, Kind: evStatus, Status: "PLANNED"},
		{Ref: "CP-4", Offset: 6, Actor: pDevin, Kind: evStatus, Status: "DOING"},
		{Ref: "CP-4", Offset: 12, Actor: pDevin, Kind: evComment, Text: "Initial schema is up. Using golang-migrate for versioned migrations. Tables created:\n- `users` - account info + auth metadata\n- `cats` - cat profiles linked to users\n- `restaurants` - restaurant profiles\n- `menu_items` - items with dietary tags as a JSONB array\n\nAll tables have `created_at`/`updated_at` timestamps and soft-delete via `deleted_at`."},
		{Ref: "CP-4", Offset: 13, Actor: pDevin, Kind: evStatus, Status: "DONE"},

		// CP-2: JWT auth (human, 1-2 days)
		{Ref: "CP-2", Offset: 5, Actor: pJake, Kind: evStatus, Status: "PLANNED"},
		{Ref: "CP-2", Offset: 10, Actor: pJake, Kind: evStatus, Status: "DOING"},
		{Ref: "CP-2", Offset: 28, Actor: pJake, Kind: evComment, Text: "JWT implementation complete. Using RS256 with key rotation. Access tokens expire in 15 min, refresh tokens in 7 days. Revocation via Redis blacklist.\n\nStill need to add OAuth providers -- will do that in a follow-up."},
		{Ref: "CP-2", Offset: 30, Actor: pJake, Kind: evStatus, Status: "DONE"},

		// MO-1: menu data model (Luna, human pace)
		{Ref: "MO-1", Offset: 8, Actor: pLuna, Kind: evStatus, Status: "PLANNED"},
		{Ref: "MO-1", Offset: 12, Actor: pLuna, Kind: evStatus, Status: "DOING"},
		{Ref: "MO-1", Offset: 32, Actor: pLuna, Kind: evComment, Text: "Menu data model finalized:\n\n- Restaurant -> many Categories -> many Items\n- Items have: name, description, price, photo_url, dietary_tags[], allergens[], availability_schedule\n- Modifiers (e.g. \"extra tuna\", \"no bones\") as a separate table linked to items\n- Dietary tags are an enum: grain_free, fish_based, organic, raw, senior, kitten, hydration\n\nThis schema supports everything we need for the dietary filtering UX."},
		{Ref: "MO-1", Offset: 34, Actor: pLuna, Kind: evStatus, Status: "DONE"},

		// DL-1: driver model (agent)
		{Ref: "DL-1", Offset: 10, Actor: pClaude, Kind: evStatus, Status: "PLANNED"},
		{Ref: "DL-1", Offset: 12, Actor: pClaude, Kind: evStatus, Status: "DOING"},
		{Ref: "DL-1", Offset: 16, Actor: pClaude, Kind: evComment, Text: "Driver model implemented:\n- Profile: name, phone, vehicle_type, license_plate, photo\n- Status enum: available, busy, offline\n- Location: lat/lng updated every 5s via WebSocket\n- Stats: total_deliveries, avg_rating, completion_rate\n\nMigration and CRUD endpoints ready."},
		{Ref: "DL-1", Offset: 17, Actor: pClaude, Kind: evStatus, Status: "DONE"},

		// MO-2: restaurant onboarding (agent)
		{Ref: "MO-2", Offset: 10, Actor: pDevin, Kind: evStatus, Status: "PLANNED"},
		{Ref: "MO-2", Offset: 14, Actor: pDevin, Kind: evStatus, Status: "DOING"},
		{Ref: "MO-2", Offset: 20, Actor: pDevin, Kind: evComment, Text: "Restaurant onboarding API implemented. Endpoints: create, read, update, delete restaurant profiles. Includes bulk menu import via CSV for restaurants migrating from other platforms. Validation on operating hours format."},
		{Ref: "MO-2", Offset: 21, Actor: pDevin, Kind: evStatus, Status: "DONE"},

		// CP-3: API gateway (Sarah, architect work)
		{Ref: "CP-3", Offset: 6, Actor: pSarah, Kind: evStatus, Status: "PLANNED"},
		{Ref: "CP-3", Offset: 14, Actor: pSarah, Kind: evStatus, Status: "DOING"},
		{Ref: "CP-3", Offset: 38, Actor: pSarah, Kind: evComment, Text: "API gateway is live on staging. Using Kong with:\n- Rate limiting: 100 req/min per user, 1000 req/min per service\n- Request logging to ELK\n- JWT validation plugin for auth\n- Service discovery via Consul\n\nAll backend services route through the gateway now."},
		{Ref: "CP-3", Offset: 40, Actor: pSarah, Kind: evStatus, Status: "DONE"},

		// CAT-1: cat profiles (Whiskers, agent)
		{Ref: "CAT-1", Offset: 14, Actor: pWhiskers, Kind: evStatus, Status: "PLANNED"},
		{Ref: "CAT-1", Offset: 16, Actor: pWhiskers, Kind: evStatus, Status: "DOING"},
		{Ref: "CAT-1", Offset: 22, Actor: pWhiskers, Kind: evComment, Text: "Cat profiles API is ready! As the team's resident cat expert, I made sure we got the data model right:\n- POST /cats - create cat profile\n- GET /cats/:id - get profile\n- PUT /cats/:id - update\n- GET /users/:id/cats - list user's cats\n\nEach cat has: name, breed, birth_date, weight_kg, photo_url, personality_traits[]. Max 10 cats per user (for now). I wanted unlimited but Sarah said we need to be practical."},
		{Ref: "CAT-1", Offset: 23, Actor: pWhiskers, Kind: evStatus, Status: "DONE"},

		// PM-1: Stripe (Jake, starts early, long task)
		{Ref: "PM-1", Offset: 14, Actor: pJake, Kind: evStatus, Status: "PLANNED"},
		{Ref: "PM-1", Offset: 32, Actor: pJake, Kind: evStatus, Status: "DOING"},

		// CP-6: staging env (Jake, infra)
		{Ref: "CP-6", Offset: 12, Actor: pJake, Kind: evStatus, Status: "PLANNED"},
		{Ref: "CP-6", Offset: 16, Actor: pJake, Kind: evStatus, Status: "DOING"},
		{Ref: "CP-6", Offset: 44, Actor: pJake, Kind: evComment, Text: "Staging environment is live!\n\nInfra stack:\n- ECS Fargate (2 services: api, workers)\n- RDS PostgreSQL 15 (db.t3.medium)\n- ElastiCache Redis 7 (cache.t3.micro)\n- S3 for media uploads\n- CloudFront CDN\n\nAll provisioned via Terraform. Deploys from `main` branch automatically."},
		{Ref: "CP-6", Offset: 46, Actor: pJake, Kind: evStatus, Status: "DONE"},

		// CP-7: RBAC (Sarah)
		{Ref: "CP-7", Offset: 16, Actor: pSarah, Kind: evStatus, Status: "PLANNED"},
		{Ref: "CP-7", Offset: 42, Actor: pSarah, Kind: evStatus, Status: "DOING"},
		{Ref: "CP-7", Offset: 58, Actor: pSarah, Kind: evComment, Text: "RBAC system done. Four roles:\n- **customer**: browse, order, manage own cats\n- **driver**: view assigned orders, update delivery status\n- **restaurant_admin**: manage own restaurant, menus, orders\n- **admin**: full access\n\nPermissions checked at API gateway level via JWT claims."},
		{Ref: "CP-7", Offset: 60, Actor: pSarah, Kind: evStatus, Status: "DONE"},

		// CP-8: logging (agent)
		{Ref: "CP-8", Offset: 22, Actor: pDevin, Kind: evStatus, Status: "PLANNED"},
		{Ref: "CP-8", Offset: 24, Actor: pDevin, Kind: evStatus, Status: "DOING"},
		{Ref: "CP-8", Offset: 30, Actor: pDevin, Kind: evComment, Text: "ELK stack deployed. All services now emit structured JSON logs. Kibana dashboards set up for error rate by service, request latency percentiles, and auth failure tracking. Retention: 30 days in Elasticsearch, archived to S3 after."},
		{Ref: "CP-8", Offset: 31, Actor: pDevin, Kind: evStatus, Status: "DONE"},

		// MD-1: mood research (Sarah, long research)
		{Ref: "MD-1", Offset: 26, Actor: pSarah, Kind: evStatus, Status: "PLANNED"},
		{Ref: "MD-1", Offset: 48, Actor: pSarah, Kind: evStatus, Status: "DOING"},

		// CP-5: user registration (agent)
		{Ref: "CP-5", Offset: 26, Actor: pClaude, Kind: evStatus, Status: "PLANNED"},
		{Ref: "CP-5", Offset: 28, Actor: pClaude, Kind: evStatus, Status: "DOING"},
		{Ref: "CP-5", Offset: 34, Actor: pClaude, Kind: evComment, Text: "Registration flow implemented. Email verification uses a signed token with 24h expiry. Password hashing with argon2id. Duplicate detection checks email and phone number. Rate limited to prevent abuse."},
		{Ref: "CP-5", Offset: 35, Actor: pClaude, Kind: evStatus, Status: "DONE"},

		// CAT-2: dietary tracking (agent)
		{Ref: "CAT-2", Offset: 32, Actor: pClaude, Kind: evStatus, Status: "PLANNED"},
		{Ref: "CAT-2", Offset: 34, Actor: pClaude, Kind: evStatus, Status: "DOING"},
		{Ref: "CAT-2", Offset: 40, Actor: pClaude, Kind: evComment, Text: "Dietary preference and allergy tracking implemented. Per-cat allergies auto-filter menu results -- if a cat is allergic to chicken, chicken-based items are dimmed with a warning badge. Also added vet notes field for special dietary instructions."},
		{Ref: "CAT-2", Offset: 41, Actor: pClaude, Kind: evStatus, Status: "DONE"},

		// CP-9: health checks (agent, quick)
		{Ref: "CP-9", Offset: 32, Actor: pDevin, Kind: evStatus, Status: "PLANNED"},
		{Ref: "CP-9", Offset: 34, Actor: pDevin, Kind: evStatus, Status: "DOING"},
		{Ref: "CP-9", Offset: 38, Actor: pDevin, Kind: evComment, Text: "Added `/healthz` and `/readyz` to all services. Health checks verify database connectivity, Redis connectivity, and downstream service reachability. Kubernetes-compatible response format."},
		{Ref: "CP-9", Offset: 39, Actor: pDevin, Kind: evStatus, Status: "DONE"},

		// CP-10: error handling (agent)
		{Ref: "CP-10", Offset: 38, Actor: pClaude, Kind: evStatus, Status: "PLANNED"},
		{Ref: "CP-10", Offset: 40, Actor: pClaude, Kind: evStatus, Status: "DOING"},
		{Ref: "CP-10", Offset: 44, Actor: pClaude, Kind: evComment, Text: "Unified error response format:\n```json\n{\n  \"error\": {\n    \"code\": \"VALIDATION_ERROR\",\n    \"message\": \"Human-readable message\",\n    \"request_id\": \"req_abc123\",\n    \"details\": [...]\n  }\n}\n```\nAll services now use the shared `apierror` package. Panic recovery middleware prevents stack traces from leaking to clients."},
		{Ref: "CP-10", Offset: 45, Actor: pClaude, Kind: evStatus, Status: "DONE"},

		// CP-11: auth integration tests (Whiskers, agent)
		{Ref: "CP-11", Offset: 42, Actor: pWhiskers, Kind: evStatus, Status: "PLANNED"},
		{Ref: "CP-11", Offset: 44, Actor: pWhiskers, Kind: evStatus, Status: "DOING"},
		{Ref: "CP-11", Offset: 50, Actor: pWhiskers, Kind: evComment, Text: "Auth test suite complete! 34 test cases covering all auth endpoints. Found two edge cases during testing: expired refresh tokens weren't returning the right error code, and rate limiting wasn't applied to the token endpoint. Both fixed and tests pass."},
		{Ref: "CP-11", Offset: 51, Actor: pWhiskers, Kind: evStatus, Status: "DONE"},

		// MO-3: menu browsing UI (Luna, human pace)
		{Ref: "MO-3", Offset: 36, Actor: pLuna, Kind: evStatus, Status: "PLANNED"},
		{Ref: "MO-3", Offset: 48, Actor: pLuna, Kind: evStatus, Status: "DOING"},

		// CP-12: typed API client (agent)
		{Ref: "CP-12", Offset: 50, Actor: pDevin, Kind: evStatus, Status: "PLANNED"},
		{Ref: "CP-12", Offset: 52, Actor: pDevin, Kind: evStatus, Status: "DOING"},
		{Ref: "CP-12", Offset: 58, Actor: pDevin, Kind: evStatus, Status: "DONE"},

		// CP-13: request validation (agent)
		{Ref: "CP-13", Offset: 58, Actor: pClaude, Kind: evStatus, Status: "PLANNED"},
		{Ref: "CP-13", Offset: 60, Actor: pClaude, Kind: evStatus, Status: "DOING"},
		{Ref: "CP-13", Offset: 66, Actor: pClaude, Kind: evStatus, Status: "DONE"},

		// CP-14: Redis caching (agent)
		{Ref: "CP-14", Offset: 66, Actor: pDevin, Kind: evStatus, Status: "PLANNED"},
		{Ref: "CP-14", Offset: 68, Actor: pDevin, Kind: evStatus, Status: "DOING"},
		{Ref: "CP-14", Offset: 74, Actor: pDevin, Kind: evComment, Text: "Redis caching live for menu data. Cache-aside pattern with 15-min TTL. Invalidation on writes. DB read volume dropped by 68% in staging load tests."},
		{Ref: "CP-14", Offset: 75, Actor: pDevin, Kind: evStatus, Status: "DONE"},

		// CP-15: UI component library (Luna)
		{Ref: "CP-15", Offset: 74, Actor: pLuna, Kind: evStatus, Status: "PLANNED"},
		{Ref: "CP-15", Offset: 80, Actor: pLuna, Kind: evStatus, Status: "DOING"},

		// CP-16: WebSocket infra (agent)
		{Ref: "CP-16", Offset: 82, Actor: pClaude, Kind: evStatus, Status: "PLANNED"},
		{Ref: "CP-16", Offset: 84, Actor: pClaude, Kind: evStatus, Status: "DOING"},
		{Ref: "CP-16", Offset: 90, Actor: pClaude, Kind: evComment, Text: "WebSocket infrastructure ready. Using gorilla/websocket with a pub/sub hub. Supports authenticated connections via JWT, room-based channels (per-order, per-restaurant, per-driver), automatic reconnection with message replay, and heartbeat every 30s."},
		{Ref: "CP-16", Offset: 91, Actor: pClaude, Kind: evStatus, Status: "DONE"},

		// CP-17: E2E test framework (Whiskers, agent)
		{Ref: "CP-17", Offset: 92, Actor: pWhiskers, Kind: evStatus, Status: "PLANNED"},
		{Ref: "CP-17", Offset: 94, Actor: pWhiskers, Kind: evStatus, Status: "DOING"},
		{Ref: "CP-17", Offset: 100, Actor: pWhiskers, Kind: evComment, Text: "Playwright framework ready with 12 initial E2E tests! Running in CI takes about 4 minutes. Covers: landing page load, login, menu browsing, dietary filter, and cart add. Parallelized across 3 workers for speed."},
		{Ref: "CP-17", Offset: 101, Actor: pWhiskers, Kind: evStatus, Status: "DONE"},

		// Links
		{Ref: "CP-2", Offset: 5, Actor: pJake, Kind: evLink, LinkType: "depends_on", TargetRef: "CP-1"},
		{Ref: "CP-5", Offset: 26, Actor: pClaude, Kind: evLink, LinkType: "depends_on", TargetRef: "CP-2"},
		{Ref: "MO-2", Offset: 10, Actor: pDevin, Kind: evLink, LinkType: "depends_on", TargetRef: "MO-1"},
		{Ref: "MO-3", Offset: 36, Actor: pLuna, Kind: evLink, LinkType: "depends_on", TargetRef: "MO-2"},
	}
}

// ──────────────────────────────────────────────────────────────────────
// Phase 2: First Sprint Push (offset 168-528h, W18 ~40pts, W19 ~50pts)
// ──────────────────────────────────────────────────────────────────────

func phase2Sprint() []seedIssue {
	return []seedIssue{
		// Core platform continued
		{Ref: "CP-18", Title: "OAuth2 login: Google and Apple", Desc: "Add Google and Apple OAuth providers to the auth service. Map external accounts to internal users. Handle first-login flow.", Labels: []string{"feature", "backend"}, Parent: "E1", Estimate: 3, Actor: pJake, Offset: 192},
		{Ref: "CP-19", Title: "Implement email notification service", Desc: "Transactional email via SendGrid: order confirmations, delivery updates, password resets, and welcome emails.", Labels: []string{"feature", "backend"}, Parent: "E1", Estimate: 2, Actor: pClaude, Offset: 198},
		{Ref: "CP-20", Title: "Add request tracing with OpenTelemetry", Desc: "Distributed tracing across all services. Propagate trace IDs through HTTP headers. Export to Jaeger.", Labels: []string{"improvement", "infra"}, Parent: "E1", Estimate: 3, Actor: pDevin, Offset: 204},
		{Ref: "CP-21", Title: "Implement rate limiting per API key", Desc: "Per-key rate limits configurable in Kong. Support burst allowance. Return X-RateLimit headers.", Labels: []string{"improvement", "backend"}, Parent: "E1", Estimate: 1, Actor: pDevin, Offset: 216},

		// Menu & Ordering
		{Ref: "MO-4", Title: "Build cart management service", Desc: "Shopping cart with add/remove/update quantities. Persistent across sessions. Validates item availability and restaurant hours.", Labels: []string{"feature", "backend"}, Parent: "E2", Estimate: 3, Actor: pClaude, Offset: 200},
		{Ref: "MO-5", Title: "Design dietary filter system", Desc: "Filterable tags: grain-free, fish-lover, senior-friendly, kitten-safe, hydration-boost, raw-diet, organic. Backend filtering + UI chips.", Labels: []string{"feature", "backend", "design"}, Parent: "E2", Estimate: 3, Actor: pLuna, Offset: 210},
		{Ref: "MO-6", Title: "Order placement and confirmation flow", Desc: "Full order lifecycle: cart -> checkout -> payment -> confirmation -> restaurant notification. Idempotent order creation.", Labels: []string{"feature", "backend"}, Parent: "E2", Estimate: 3, Actor: pClaude, Offset: 230},
		{Ref: "MO-7", Title: "Build order status tracking page", Desc: "Real-time order status page with WebSocket updates: confirmed, preparing, picked up, delivering, delivered.", Labels: []string{"feature", "frontend"}, Parent: "E2", Estimate: 3, Actor: pLuna, Offset: 260},
		{Ref: "MO-8", Title: "Implement restaurant admin dashboard", Desc: "Dashboard for restaurants to manage menu items, view incoming orders, update order status, and see analytics.", Labels: []string{"feature", "frontend"}, Parent: "E2", Estimate: 3, Actor: pWhiskers, Offset: 280},
		{Ref: "MO-9", Title: "Add menu item photo upload", Desc: "S3-backed photo upload for menu items. Auto-resize to thumbnails and detail views. CDN distribution.", Labels: []string{"feature", "backend"}, Parent: "E2", Estimate: 2, Actor: pDevin, Offset: 290},
		{Ref: "MO-10", Title: "Implement menu search with Elasticsearch", Desc: "Full-text search across restaurant names, menu items, and descriptions. Typo tolerance and cat-food keyword boosting.", Labels: []string{"feature", "backend"}, Parent: "E2", Estimate: 3, Actor: pClaude, Offset: 300},
		{Ref: "MO-11", Title: "Add restaurant rating and review system", Desc: "5-star ratings with text reviews. Aggregate rating display. Verified-purchase badge. Report/flag functionality.", Labels: []string{"feature", "backend"}, Parent: "E2", Estimate: 3, Actor: pDevin, Offset: 320},
		{Ref: "MO-12", Title: "Build reorder from history feature", Desc: "Quick reorder button on past orders. Pre-fill cart with previous items. Check availability before adding.", Labels: []string{"feature", "frontend"}, Parent: "E2", Estimate: 2, Actor: pWhiskers, Offset: 340},
		{Ref: "MO-13", Title: "Implement order cancellation flow", Desc: "Allow cancellation within 5 minutes of placement. After that, require restaurant approval. Auto-refund on successful cancel.", Labels: []string{"feature", "backend"}, Parent: "E2", Estimate: 3, Actor: pClaude, Offset: 360},
		{Ref: "MO-14", Title: "Restaurant hours and availability management", Desc: "Support for regular hours, holiday hours, and temporary closures. Auto-hide unavailable restaurants from search.", Labels: []string{"feature", "backend"}, Parent: "E2", Estimate: 2, Actor: pDevin, Offset: 380},
		{Ref: "MO-15", Title: "Build multi-restaurant cart (split orders)", Desc: "Allow items from multiple restaurants in one cart. Split into separate orders per restaurant. Combined checkout.", Labels: []string{"feature", "backend"}, Parent: "E2", Estimate: 3, Actor: pClaude, Offset: 400},
		{Ref: "MO-16", Title: "Add search autocomplete with debounced suggestions", Desc: "Type-ahead search with 300ms debounce. Show restaurant names, menu items, and dietary filters as suggestions.", Labels: []string{"feature", "frontend"}, Parent: "E2", Estimate: 2, Actor: pWhiskers, Offset: 420},
		{Ref: "MO-17", Title: "Implement order notification email templates", Desc: "Responsive email templates for: order confirmed, order preparing, driver assigned, delivered, and review request.", Labels: []string{"feature", "backend"}, Parent: "E2", Estimate: 2, Actor: pClaude, Offset: 440},
		{Ref: "MO-18", Title: "Database query performance audit", Desc: "Review slow query log from OpenTelemetry. Add missing indexes. Optimize N+1 queries in menu listing and order history.", Labels: []string{"improvement", "backend"}, Parent: "E2", Estimate: 3, Actor: pDevin, Offset: 460},

		// Delivery
		{Ref: "DL-2", Title: "Implement driver assignment algorithm", Desc: "Assign nearest available driver to new orders. Consider distance, current load, and driver rating. Fallback to broadcast if no match.", Labels: []string{"feature", "backend"}, Parent: "E3", Estimate: 3, Actor: pSarah, Offset: 220},
		{Ref: "DL-3", Title: "Build real-time delivery tracking", Desc: "WebSocket-based live tracking of driver location. Update every 5 seconds. Map view with ETA calculation.", Labels: []string{"feature", "frontend"}, Parent: "E3", Estimate: 2, Actor: pLuna, Offset: 270},
		{Ref: "DL-4", Title: "Delivery photo confirmation", Desc: "Drivers upload photo proof of delivery. Stored in S3, linked to order. Visible to customer in order history.", Labels: []string{"feature", "mobile"}, Parent: "E3", Estimate: 2, Actor: pDevin, Offset: 310},
		{Ref: "DL-5", Title: "Route optimization with Google Maps API", Desc: "Optimize multi-stop delivery routes using Google Directions API. Batch nearby deliveries. Show estimated times.", Labels: []string{"feature", "backend"}, Parent: "E3", Estimate: 2, Actor: pClaude, Offset: 330},
		{Ref: "DL-6", Title: "Driver earnings and payout dashboard", Desc: "Dashboard showing delivery history, earnings breakdown, tips, and payout schedule. Weekly automatic payouts via Stripe Connect.", Labels: []string{"feature", "frontend"}, Parent: "E3", Estimate: 3, Actor: pWhiskers, Offset: 350},
		{Ref: "DL-7", Title: "Implement delivery fee calculation", Desc: "Distance-based fee: $2.99 base + $0.50/km. Surge pricing during peak hours. Free delivery for Upurr Premium.", Labels: []string{"feature", "backend"}, Parent: "E3", Estimate: 2, Actor: pDevin, Offset: 370},
		{Ref: "DL-8", Title: "Build driver mobile app MVP", Desc: "React Native app for drivers: order queue, navigation integration, delivery status updates, and earnings view.", Labels: []string{"feature", "mobile"}, Parent: "E3", Estimate: 5, Actor: pJake, Offset: 390},
		{Ref: "DL-9", Title: "Implement delivery time estimation", Desc: "Estimate delivery time from: restaurant prep time + driver travel time. Show on order page. Update as status changes.", Labels: []string{"feature", "backend"}, Parent: "E3", Estimate: 3, Actor: pClaude, Offset: 430},
		{Ref: "DL-10", Title: "Add driver push notifications", Desc: "Push notifications for new order assignments, customer messages, and delivery reminders.", Labels: []string{"feature", "mobile"}, Parent: "E3", Estimate: 2, Actor: pWhiskers, Offset: 450},

		// Payments
		{Ref: "PM-2", Title: "Build checkout UI with payment form", Desc: "Stripe Elements-based checkout. Card input, order summary, tip selection, and promo code entry.", Labels: []string{"feature", "frontend"}, Parent: "E4", Estimate: 3, Actor: pLuna, Offset: 240},
		{Ref: "PM-3", Title: "Implement refund processing", Desc: "Full and partial refunds via Stripe. Admin UI for refund approval. Automatic refund for cancelled orders within 5 min window.", Labels: []string{"feature", "backend"}, Parent: "E4", Estimate: 3, Actor: pClaude, Offset: 290},
		{Ref: "PM-4", Title: "Promotional credit system", Desc: "Issue and redeem promotional credits. Support for referral bonuses, first-order discounts, and seasonal campaigns.", Labels: []string{"feature", "backend"}, Parent: "E4", Estimate: 3, Actor: pDevin, Offset: 330},
		{Ref: "PM-5", Title: "Upurr Premium subscription billing", Desc: "Monthly subscription: $9.99/month for free delivery + priority support. Stripe Billing integration with trial periods.", Labels: []string{"feature", "backend"}, Parent: "E4", Estimate: 5, Actor: pJake, Offset: 400},
		{Ref: "PM-6", Title: "Generate PDF invoices", Desc: "Auto-generate PDF invoices for each order. Email to customer. Include itemized breakdown, tax, delivery fee, and tip.", Labels: []string{"feature", "backend"}, Parent: "E4", Estimate: 2, Actor: pClaude, Offset: 450},
		{Ref: "PM-7", Title: "Tax calculation service", Desc: "Calculate applicable sales tax by delivery address. Integration with TaxJar API for accurate rates.", Labels: []string{"feature", "backend"}, Parent: "E4", Estimate: 2, Actor: pDevin, Offset: 470},
		{Ref: "PM-8", Title: "Implement tipping flow", Desc: "Pre-set tip amounts (15%, 18%, 20%, custom) at checkout. Option to add tip after delivery. Tips go directly to driver.", Labels: []string{"feature", "frontend"}, Parent: "E4", Estimate: 2, Actor: pLuna, Offset: 250},
		{Ref: "PM-9", Title: "Build payment history and receipts page", Desc: "List of all past payments with receipt details. Filter by date range. Download PDF receipts.", Labels: []string{"feature", "frontend"}, Parent: "E4", Estimate: 2, Actor: pWhiskers, Offset: 380},
		{Ref: "PM-10", Title: "Implement Stripe webhook handling", Desc: "Handle Stripe webhooks for payment_intent.succeeded, charge.refunded, invoice.paid, subscription.updated. Idempotent processing.", Labels: []string{"feature", "backend"}, Parent: "E4", Estimate: 3, Actor: pJake, Offset: 410},

		// Cat Profiles
		{Ref: "CAT-3", Title: "Design cat profile UI with avatar upload", Desc: "Adorable cat profile cards with circular avatar, stats, and dietary badges. Photo upload with crop/rotate.", Labels: []string{"feature", "frontend", "design"}, Parent: "E5", Estimate: 3, Actor: pLuna, Offset: 230},
		{Ref: "CAT-4", Title: "Personalized food recommendations engine", Desc: "Recommend menu items based on cat's age, breed, dietary needs, and order history. Collaborative filtering for new cats.", Labels: []string{"feature", "backend"}, Parent: "E5", Estimate: 5, Actor: pSarah, Offset: 350},
		{Ref: "CAT-5", Title: "Feeding schedule reminders", Desc: "Configurable feeding schedule with push notifications. Support breakfast/lunch/dinner/snack slots. Integration with cat profiles.", Labels: []string{"feature", "mobile"}, Parent: "E5", Estimate: 3, Actor: pClaude, Offset: 390},
		{Ref: "CAT-6", Title: "Cat birthday celebration feature", Desc: "Special birthday menu suggestions and a free treat with orders on a cat's birthday. Confetti animation in the app.", Labels: []string{"feature", "frontend"}, Parent: "E5", Estimate: 2, Actor: pWhiskers, Offset: 420},
		{Ref: "CAT-7", Title: "Weight tracking and health trends", Desc: "Log cat weight over time. Chart showing trends. Alert if significant weight change detected. Link to vet recommendations.", Labels: []string{"feature", "frontend"}, Parent: "E5", Estimate: 2, Actor: pLuna, Offset: 440},
		{Ref: "CAT-8", Title: "Multi-cat household order splitting", Desc: "When ordering for multiple cats, split order items by cat. Track per-cat spending and dietary compliance.", Labels: []string{"feature", "backend"}, Parent: "E5", Estimate: 2, Actor: pDevin, Offset: 460},
		{Ref: "CAT-9", Title: "Cat profile sharing between family members", Desc: "Share cat profiles across user accounts. Invite via email or link. Configurable permissions (view-only, can-order, admin).", Labels: []string{"feature", "backend"}, Parent: "E5", Estimate: 2, Actor: pClaude, Offset: 480},

		// Mood Detection (stretch goal)
		{Ref: "MD-2", Title: "Build mood detection training pipeline", Desc: "Data pipeline for labeling cat mood images. Integration with Label Studio. Export to fine-tuning format.", Labels: []string{"feature", "backend"}, Parent: "E6", Estimate: 5, Actor: pClaude, Offset: 250},
		{Ref: "MD-3", Title: "Design mood-based food suggestion UI", Desc: "After mood detection: show comfort food suggestions for grumpy cats, energy food for playful, etc. Fun animations per mood.", Labels: []string{"feature", "frontend", "design"}, Parent: "E6", Estimate: 3, Actor: pLuna, Offset: 300},
		{Ref: "MD-4", Title: "Implement photo capture and upload flow", Desc: "In-app camera integration for mood detection photos. Crop to face, basic quality checks, upload to processing queue.", Labels: []string{"feature", "mobile"}, Parent: "E6", Estimate: 3, Actor: pDevin, Offset: 350},
		{Ref: "MD-5", Title: "Build mood detection API endpoint", Desc: "REST endpoint that accepts cat photo, runs inference, returns mood classification with confidence score.", Labels: []string{"feature", "backend"}, Parent: "E6", Estimate: 5, Actor: pClaude, Offset: 400},
		{Ref: "MD-6", Title: "Collect and curate cat mood training dataset", Desc: "Source 10k+ labeled cat mood images. Mix of public datasets and user-submitted photos (with consent). Balance across all mood categories.", Labels: []string{"feature"}, Parent: "E6", Estimate: 3, Actor: pWhiskers, Offset: 420},

		// Additional dev work
		{Ref: "MO-19", Title: "API versioning strategy and v1 URL prefix", Desc: "Add /api/v1 prefix to all endpoints. Version negotiation via Accept header. Document migration path for future versions.", Labels: []string{"improvement", "backend"}, Estimate: 2, Actor: pDevin, Offset: 490},
		{Ref: "MO-20", Title: "Write E2E tests for full ordering flow", Desc: "Playwright tests covering: browse -> filter -> add to cart -> checkout -> confirm. Includes payment with Stripe test cards.", Labels: []string{"improvement"}, Estimate: 3, Actor: pWhiskers, Offset: 500},
		{Ref: "MO-21", Title: "Implement image optimization pipeline", Desc: "Auto-resize uploaded images to standard sizes (thumbnail, card, detail). Convert to WebP. Lazy load below-fold images.", Labels: []string{"improvement", "backend"}, Estimate: 2, Actor: pDevin, Offset: 510},
		{Ref: "MO-22", Title: "Add pagination to all list endpoints", Desc: "Cursor-based pagination for: restaurants, menu items, orders, reviews. Default page size 20, max 100.", Labels: []string{"improvement", "backend"}, Estimate: 2, Actor: pClaude, Offset: 520},

		// Bugs during development
		{Ref: "BUG-1", Title: "Menu items with special characters break search", Desc: "Elasticsearch throws when menu item names contain ampersands or angle brackets. Needs input sanitization.", Labels: []string{"bug", "backend"}, Estimate: 1, Actor: pJake, Offset: 310},
		{Ref: "BUG-2", Title: "Cart total doesn't update when item quantity changes", Desc: "Frontend cart component doesn't recalculate total when using the +/- quantity buttons. Need to re-trigger price calculation.", Labels: []string{"bug", "frontend"}, Estimate: 1, Actor: pLuna, Offset: 340},
		{Ref: "BUG-3", Title: "Driver location updates stop after app backgrounding", Desc: "On iOS, WebSocket disconnects when the driver app goes to background. Location stops updating for customers.", Labels: []string{"bug", "mobile"}, Estimate: 2, Actor: pJake, Offset: 430},
		{Ref: "BUG-4", Title: "Restaurant photos don't load on slow connections", Desc: "Menu photos time out on 3G connections. Need progressive loading and lower-res fallbacks.", Labels: []string{"bug", "frontend"}, Estimate: 1, Actor: pLuna, Offset: 470},
	}
}

func phase2Events() []seedEvent {
	return []seedEvent{
		// E1 -> DOING
		{Ref: "E1", Offset: 170, Actor: pSarah, Kind: evStatus, Status: "DOING"},
		// E2 -> DOING
		{Ref: "E2", Offset: 170, Actor: pSarah, Kind: evStatus, Status: "DOING"},
		// E3 -> DOING
		{Ref: "E3", Offset: 220, Actor: pJake, Kind: evStatus, Status: "DOING"},
		// E4 -> DOING
		{Ref: "E4", Offset: 240, Actor: pSarah, Kind: evStatus, Status: "DOING"},
		// E5 -> DOING
		{Ref: "E5", Offset: 230, Actor: pLuna, Kind: evStatus, Status: "DOING"},

		// CP-15: UI component library (Luna, continues from phase 1)
		{Ref: "CP-15", Offset: 106, Actor: pLuna, Kind: evComment, Text: "Component library done with Storybook. 24 components including Button, Card, Modal, Input, Select, Avatar, Skeleton, Toast, Dropdown, Tabs, and more. All with consistent theming and responsive behavior."},
		{Ref: "CP-15", Offset: 108, Actor: pLuna, Kind: evStatus, Status: "DONE"},

		// MO-3: menu browsing UI (Luna, continues)
		{Ref: "MO-3", Offset: 120, Actor: pLuna, Kind: evComment, Text: "Design review feedback: the dietary filter chips look great. Using paw-print icons for the filter categories instead of generic tags makes it feel more on-brand. Added a favorites section at the top for returning customers."},
		{Ref: "MO-3", Offset: 122, Actor: pLuna, Kind: evStatus, Status: "DONE"},

		// MD-1: mood research (Sarah, continues from phase 1)
		{Ref: "MD-1", Offset: 160, Actor: pSarah, Kind: evComment, Text: "Evaluated three approaches for cat mood detection:\n\n1. **GPT-4V** - Good accuracy but expensive at scale ($0.03/image)\n2. **Claude Vision** - Similar accuracy, slightly better at distinguishing \"sleepy\" vs \"relaxed\"\n3. **Custom fine-tune** - Would need 50k+ labeled images, 3-4 weeks training time\n\nRecommendation: start with Claude Vision for MVP, collect user data, fine-tune later. The API cost is manageable for our projected volume."},
		{Ref: "MD-1", Offset: 164, Actor: pSarah, Kind: evStatus, Status: "DONE"},

		// PM-1: Stripe integration (Jake, continues from phase 1)
		{Ref: "PM-1", Offset: 170, Actor: pJake, Kind: evComment, Text: "Stripe integration is more complex than estimated. The Payment Intents API has a lot of edge cases around 3D Secure and card authentication. Bumping estimate."},
		{Ref: "PM-1", Offset: 200, Actor: pJake, Kind: evComment, Text: "Stripe integration complete. Supports:\n- Card payments via Payment Intents\n- Saved cards (Stripe Customer objects)\n- 3D Secure authentication\n- Idempotent charges\n\nTest mode working. Will need to finish Stripe webhook handling (PM-10) before going live."},
		{Ref: "PM-1", Offset: 202, Actor: pJake, Kind: evStatus, Status: "DONE"},

		// CP-18: OAuth (Jake)
		{Ref: "CP-18", Offset: 194, Actor: pJake, Kind: evStatus, Status: "PLANNED"},
		{Ref: "CP-18", Offset: 204, Actor: pJake, Kind: evStatus, Status: "DOING"},
		{Ref: "CP-18", Offset: 228, Actor: pJake, Kind: evComment, Text: "Google OAuth working. Apple Sign In is a pain -- their documentation is... something. But it's working now. Both providers create/link internal user accounts seamlessly."},
		{Ref: "CP-18", Offset: 230, Actor: pJake, Kind: evStatus, Status: "DONE"},

		// CP-19: email notifications (agent)
		{Ref: "CP-19", Offset: 200, Actor: pClaude, Kind: evStatus, Status: "PLANNED"},
		{Ref: "CP-19", Offset: 202, Actor: pClaude, Kind: evStatus, Status: "DOING"},
		{Ref: "CP-19", Offset: 208, Actor: pClaude, Kind: evComment, Text: "Email service operational. Templates created for welcome email, email verification, order confirmation, delivery updates, and password reset. Using SendGrid with template versioning. HTML and plain text variants for all templates."},
		{Ref: "CP-19", Offset: 209, Actor: pClaude, Kind: evStatus, Status: "DONE"},

		// CP-20: OpenTelemetry (agent)
		{Ref: "CP-20", Offset: 206, Actor: pDevin, Kind: evStatus, Status: "PLANNED"},
		{Ref: "CP-20", Offset: 208, Actor: pDevin, Kind: evStatus, Status: "DOING"},
		{Ref: "CP-20", Offset: 214, Actor: pDevin, Kind: evComment, Text: "OpenTelemetry tracing deployed. Every request gets a trace ID that propagates across services. Jaeger UI available at jaeger.internal.upurreats.com. Already spotted a slow query in the menu search endpoint."},
		{Ref: "CP-20", Offset: 215, Actor: pDevin, Kind: evStatus, Status: "DONE"},

		// CP-21: rate limiting (agent, quick)
		{Ref: "CP-21", Offset: 218, Actor: pDevin, Kind: evStatus, Status: "PLANNED"},
		{Ref: "CP-21", Offset: 220, Actor: pDevin, Kind: evStatus, Status: "DOING"},
		{Ref: "CP-21", Offset: 224, Actor: pDevin, Kind: evStatus, Status: "DONE"},

		// MO-4: cart management (agent)
		{Ref: "MO-4", Offset: 202, Actor: pClaude, Kind: evStatus, Status: "PLANNED"},
		{Ref: "MO-4", Offset: 210, Actor: pClaude, Kind: evStatus, Status: "DOING"},
		{Ref: "MO-4", Offset: 216, Actor: pClaude, Kind: evComment, Text: "Cart service implemented with Redis-backed persistence. Cart survives browser refresh and app restart. 24h TTL on abandoned carts. Validates item availability on every cart view."},
		{Ref: "MO-4", Offset: 217, Actor: pClaude, Kind: evStatus, Status: "DONE"},

		// MO-5: dietary filters (Luna)
		{Ref: "MO-5", Offset: 212, Actor: pLuna, Kind: evStatus, Status: "PLANNED"},
		{Ref: "MO-5", Offset: 220, Actor: pLuna, Kind: evStatus, Status: "DOING"},
		{Ref: "MO-5", Offset: 244, Actor: pLuna, Kind: evComment, Text: "Dietary filter system complete. Seven filter categories with fun cat-themed icons:\n- Fish Lover\n- Grain Free\n- Raw Diet\n- Organic\n- Senior Friendly\n- Kitten Safe\n- Hydration Boost\n\nFilters are AND-combined. Results update in real-time."},
		{Ref: "MO-5", Offset: 246, Actor: pLuna, Kind: evStatus, Status: "DONE"},

		// DL-2: driver assignment (Sarah)
		{Ref: "DL-2", Offset: 222, Actor: pSarah, Kind: evStatus, Status: "PLANNED"},
		{Ref: "DL-2", Offset: 230, Actor: pSarah, Kind: evStatus, Status: "DOING"},
		{Ref: "DL-2", Offset: 262, Actor: pSarah, Kind: evComment, Text: "Driver assignment algorithm:\n1. Find all available drivers within 5km of restaurant\n2. Score by: distance (40%), rating (30%), current load (30%)\n3. Offer to top candidate with 60s accept window\n4. If declined/expired, offer to next\n5. After 3 rejections, broadcast to all nearby drivers\n\nAverage assignment time in testing: 12 seconds."},
		{Ref: "DL-2", Offset: 264, Actor: pSarah, Kind: evStatus, Status: "DONE"},

		// CAT-3: cat profile UI (Luna)
		{Ref: "CAT-3", Offset: 232, Actor: pLuna, Kind: evStatus, Status: "PLANNED"},
		{Ref: "CAT-3", Offset: 248, Actor: pLuna, Kind: evStatus, Status: "DOING"},
		{Ref: "CAT-3", Offset: 276, Actor: pLuna, Kind: evComment, Text: "Cat profile cards are adorable. Circular avatar with a subtle shadow, breed badge, and dietary icons. The photo upload supports crop and rotate with a paw-shaped crop overlay (because why not)."},
		{Ref: "CAT-3", Offset: 278, Actor: pLuna, Kind: evStatus, Status: "DONE"},

		// MO-6: order placement (agent)
		{Ref: "MO-6", Offset: 234, Actor: pClaude, Kind: evStatus, Status: "PLANNED"},
		{Ref: "MO-6", Offset: 236, Actor: pClaude, Kind: evStatus, Status: "DOING"},
		{Ref: "MO-6", Offset: 244, Actor: pClaude, Kind: evComment, Text: "Order placement flow implemented. Key design decisions:\n- Idempotency key in the request header to prevent duplicate orders\n- Order state machine: PENDING -> CONFIRMED -> PREPARING -> READY -> PICKED_UP -> DELIVERED\n- Restaurant gets a push notification + email on new orders\n- 15-minute auto-cancel if restaurant doesn't confirm"},
		{Ref: "MO-6", Offset: 245, Actor: pClaude, Kind: evStatus, Status: "DONE"},

		// PM-2: checkout UI (Luna)
		{Ref: "PM-2", Offset: 242, Actor: pLuna, Kind: evStatus, Status: "PLANNED"},
		{Ref: "PM-2", Offset: 250, Actor: pLuna, Kind: evStatus, Status: "DOING"},
		{Ref: "PM-2", Offset: 274, Actor: pLuna, Kind: evComment, Text: "Checkout UI done with Stripe Elements. Clean design with order summary, saved card selection, and promo code input. The tip selector design came out great -- preset amounts with custom option."},
		{Ref: "PM-2", Offset: 276, Actor: pLuna, Kind: evStatus, Status: "DONE"},

		// PM-8: tipping flow (Luna)
		{Ref: "PM-8", Offset: 252, Actor: pLuna, Kind: evStatus, Status: "PLANNED"},
		{Ref: "PM-8", Offset: 278, Actor: pLuna, Kind: evStatus, Status: "DOING"},
		{Ref: "PM-8", Offset: 298, Actor: pLuna, Kind: evStatus, Status: "DONE"},

		// MD-2: mood training pipeline (agent, PLANNED only)
		{Ref: "MD-2", Offset: 252, Actor: pClaude, Kind: evStatus, Status: "PLANNED"},

		// MO-7: order tracking page (Luna)
		{Ref: "MO-7", Offset: 262, Actor: pLuna, Kind: evStatus, Status: "PLANNED"},
		{Ref: "MO-7", Offset: 280, Actor: pLuna, Kind: evStatus, Status: "DOING"},
		{Ref: "MO-7", Offset: 306, Actor: pLuna, Kind: evComment, Text: "Order tracking page live. Features:\n- Real-time status bar with cute cat animations per stage\n- Live map showing driver location\n- ETA countdown\n- Chat with driver\n\nUsing WebSocket for live updates. Falls back to polling on unreliable connections."},
		{Ref: "MO-7", Offset: 308, Actor: pLuna, Kind: evStatus, Status: "DONE"},

		// DL-3: real-time tracking (Luna)
		{Ref: "DL-3", Offset: 272, Actor: pLuna, Kind: evStatus, Status: "PLANNED"},
		{Ref: "DL-3", Offset: 310, Actor: pLuna, Kind: evStatus, Status: "DOING"},
		{Ref: "DL-3", Offset: 338, Actor: pLuna, Kind: evComment, Text: "Live tracking is working beautifully. The little cat paw moving along the map is my favorite feature in the whole app. ETA updates every 30 seconds based on real traffic data."},
		{Ref: "DL-3", Offset: 340, Actor: pLuna, Kind: evStatus, Status: "DONE"},

		// MO-8: restaurant dashboard (Whiskers, agent)
		{Ref: "MO-8", Offset: 282, Actor: pWhiskers, Kind: evStatus, Status: "PLANNED"},
		{Ref: "MO-8", Offset: 284, Actor: pWhiskers, Kind: evStatus, Status: "DOING"},
		{Ref: "MO-8", Offset: 292, Actor: pWhiskers, Kind: evComment, Text: "Restaurant dashboard MVP done! Features:\n- Live order feed with accept/reject buttons\n- Menu editor with drag-and-drop reordering\n- Daily/weekly sales charts\n- Customer review management\n\nRestaurant owners can manage everything from a single page. Pretty proud of how the order feed turned out."},
		{Ref: "MO-8", Offset: 293, Actor: pWhiskers, Kind: evStatus, Status: "DONE"},

		// MO-9: photo upload (agent)
		{Ref: "MO-9", Offset: 292, Actor: pDevin, Kind: evStatus, Status: "PLANNED"},
		{Ref: "MO-9", Offset: 294, Actor: pDevin, Kind: evStatus, Status: "DOING"},
		{Ref: "MO-9", Offset: 300, Actor: pDevin, Kind: evStatus, Status: "DONE"},

		// PM-3: refund processing (agent)
		{Ref: "PM-3", Offset: 292, Actor: pClaude, Kind: evStatus, Status: "PLANNED"},
		{Ref: "PM-3", Offset: 294, Actor: pClaude, Kind: evStatus, Status: "DOING"},
		{Ref: "PM-3", Offset: 302, Actor: pClaude, Kind: evComment, Text: "Refund processing implemented. Supports full and partial refunds. Admin dashboard shows pending refund requests with order details and reason. Auto-refund kicks in for orders cancelled within the grace period."},
		{Ref: "PM-3", Offset: 303, Actor: pClaude, Kind: evStatus, Status: "DONE"},

		// MO-10: menu search (agent)
		{Ref: "MO-10", Offset: 302, Actor: pClaude, Kind: evStatus, Status: "PLANNED"},
		{Ref: "MO-10", Offset: 304, Actor: pClaude, Kind: evStatus, Status: "DOING"},
		{Ref: "MO-10", Offset: 310, Actor: pClaude, Kind: evComment, Text: "Elasticsearch search is live. Indexed all restaurants and menu items. Supports fuzzy matching so \"tuna\" finds \"Tuna Delight\" and \"Fresh Tuna Pate\". Boosted results for items matching the user's cat's dietary preferences."},
		{Ref: "MO-10", Offset: 311, Actor: pClaude, Kind: evStatus, Status: "DONE"},

		// BUG-1: search special chars
		{Ref: "BUG-1", Offset: 312, Actor: pClaude, Kind: evStatus, Status: "DOING"},
		{Ref: "BUG-1", Offset: 314, Actor: pClaude, Kind: evComment, Text: "Root cause: Elasticsearch query_string parser chokes on unescaped special chars. Fixed by switching to match query with analyzer that strips special characters."},
		{Ref: "BUG-1", Offset: 315, Actor: pClaude, Kind: evStatus, Status: "DONE"},

		// DL-4: delivery photo (agent)
		{Ref: "DL-4", Offset: 312, Actor: pDevin, Kind: evStatus, Status: "PLANNED"},
		{Ref: "DL-4", Offset: 314, Actor: pDevin, Kind: evStatus, Status: "DOING"},
		{Ref: "DL-4", Offset: 320, Actor: pDevin, Kind: evStatus, Status: "DONE"},

		// MO-11: ratings (agent)
		{Ref: "MO-11", Offset: 322, Actor: pDevin, Kind: evStatus, Status: "PLANNED"},
		{Ref: "MO-11", Offset: 324, Actor: pDevin, Kind: evStatus, Status: "DOING"},
		{Ref: "MO-11", Offset: 332, Actor: pDevin, Kind: evComment, Text: "Rating system implemented. Aggregate ratings use a Bayesian average to avoid one-review restaurants dominating. Reviews are verified-purchase only. Added a helpful vote on reviews."},
		{Ref: "MO-11", Offset: 333, Actor: pDevin, Kind: evStatus, Status: "DONE"},

		// PM-4: promo credits (agent)
		{Ref: "PM-4", Offset: 332, Actor: pDevin, Kind: evStatus, Status: "PLANNED"},
		{Ref: "PM-4", Offset: 334, Actor: pDevin, Kind: evStatus, Status: "DOING"},
		{Ref: "PM-4", Offset: 342, Actor: pDevin, Kind: evComment, Text: "Promo credit system implemented. Supports: fixed amount credits ($5 off), percentage discounts (20% off), free delivery credits, and referral bonuses (both referrer and referee get $10). Credits stack but capped at 50% of order total."},
		{Ref: "PM-4", Offset: 343, Actor: pDevin, Kind: evStatus, Status: "DONE"},

		// DL-5: route optimization (agent)
		{Ref: "DL-5", Offset: 332, Actor: pClaude, Kind: evStatus, Status: "PLANNED"},
		{Ref: "DL-5", Offset: 334, Actor: pClaude, Kind: evStatus, Status: "DOING"},
		{Ref: "DL-5", Offset: 340, Actor: pClaude, Kind: evComment, Text: "Route optimization with Google Maps working. Batches up to 3 nearby deliveries per driver. Estimated 22% reduction in total driving distance vs sequential delivery. API cost about $0.005 per route calculation."},
		{Ref: "DL-5", Offset: 341, Actor: pClaude, Kind: evStatus, Status: "DONE"},

		// BUG-2: cart total
		{Ref: "BUG-2", Offset: 342, Actor: pLuna, Kind: evStatus, Status: "DOING"},
		{Ref: "BUG-2", Offset: 346, Actor: pLuna, Kind: evComment, Text: "React state issue -- the quantity change handler was updating local state but not triggering the price recalculation effect. Moved to useReducer for the cart state and it's solid now."},
		{Ref: "BUG-2", Offset: 348, Actor: pLuna, Kind: evStatus, Status: "DONE"},

		// MO-12: reorder (Whiskers, agent)
		{Ref: "MO-12", Offset: 342, Actor: pWhiskers, Kind: evStatus, Status: "PLANNED"},
		{Ref: "MO-12", Offset: 344, Actor: pWhiskers, Kind: evStatus, Status: "DOING"},
		{Ref: "MO-12", Offset: 350, Actor: pWhiskers, Kind: evStatus, Status: "DONE"},

		// CAT-4: recommendations engine (Sarah)
		{Ref: "CAT-4", Offset: 352, Actor: pSarah, Kind: evStatus, Status: "PLANNED"},

		// DL-6: driver earnings (Whiskers, agent)
		{Ref: "DL-6", Offset: 352, Actor: pWhiskers, Kind: evStatus, Status: "PLANNED"},
		{Ref: "DL-6", Offset: 354, Actor: pWhiskers, Kind: evStatus, Status: "DOING"},
		{Ref: "DL-6", Offset: 362, Actor: pWhiskers, Kind: evComment, Text: "Driver earnings dashboard is done! Shows today's earnings with live updates, weekly breakdown chart, tip history, and payout schedule (next Wednesday). Sarah reviewed the code and suggested some improvements to the chart rendering that I've incorporated."},
		{Ref: "DL-6", Offset: 363, Actor: pWhiskers, Kind: evStatus, Status: "DONE"},

		// MO-13: order cancellation (agent)
		{Ref: "MO-13", Offset: 362, Actor: pClaude, Kind: evStatus, Status: "PLANNED"},
		{Ref: "MO-13", Offset: 364, Actor: pClaude, Kind: evStatus, Status: "DOING"},
		{Ref: "MO-13", Offset: 372, Actor: pClaude, Kind: evComment, Text: "Cancellation flow implemented with 5-min grace period. After that, restaurant must approve. Refund automatically issued on successful cancellation. Wired carefully with the Stripe refund API to handle edge cases."},
		{Ref: "MO-13", Offset: 373, Actor: pClaude, Kind: evStatus, Status: "DONE"},

		// DL-7: delivery fee (agent)
		{Ref: "DL-7", Offset: 372, Actor: pDevin, Kind: evStatus, Status: "PLANNED"},
		{Ref: "DL-7", Offset: 374, Actor: pDevin, Kind: evStatus, Status: "DOING"},
		{Ref: "DL-7", Offset: 380, Actor: pDevin, Kind: evStatus, Status: "DONE"},

		// MO-14: restaurant hours (agent)
		{Ref: "MO-14", Offset: 382, Actor: pDevin, Kind: evStatus, Status: "PLANNED"},
		{Ref: "MO-14", Offset: 384, Actor: pDevin, Kind: evStatus, Status: "DOING"},
		{Ref: "MO-14", Offset: 390, Actor: pDevin, Kind: evStatus, Status: "DONE"},

		// PM-9: payment history (Whiskers, agent)
		{Ref: "PM-9", Offset: 382, Actor: pWhiskers, Kind: evStatus, Status: "PLANNED"},
		{Ref: "PM-9", Offset: 384, Actor: pWhiskers, Kind: evStatus, Status: "DOING"},
		{Ref: "PM-9", Offset: 392, Actor: pWhiskers, Kind: evStatus, Status: "DONE"},

		// DL-8: driver app MVP (Jake)
		{Ref: "DL-8", Offset: 392, Actor: pJake, Kind: evStatus, Status: "PLANNED"},
		{Ref: "DL-8", Offset: 396, Actor: pJake, Kind: evStatus, Status: "DOING"},

		// CAT-5: feeding schedule (agent)
		{Ref: "CAT-5", Offset: 392, Actor: pClaude, Kind: evStatus, Status: "PLANNED"},
		{Ref: "CAT-5", Offset: 394, Actor: pClaude, Kind: evStatus, Status: "DOING"},
		{Ref: "CAT-5", Offset: 402, Actor: pClaude, Kind: evComment, Text: "Feeding schedule reminders implemented. Owners can set breakfast/lunch/dinner/snack times per cat. Push notifications fire 10 minutes before schedule. Added a \"time to order\" deep link that opens the app to the cat's recommended items."},
		{Ref: "CAT-5", Offset: 403, Actor: pClaude, Kind: evStatus, Status: "DONE"},

		// PM-5: subscription billing (Jake)
		{Ref: "PM-5", Offset: 402, Actor: pJake, Kind: evStatus, Status: "PLANNED"},
		{Ref: "PM-5", Offset: 410, Actor: pJake, Kind: evStatus, Status: "DOING"},

		// MO-15: multi-restaurant cart (agent)
		{Ref: "MO-15", Offset: 402, Actor: pClaude, Kind: evStatus, Status: "PLANNED"},
		{Ref: "MO-15", Offset: 404, Actor: pClaude, Kind: evStatus, Status: "DOING"},
		{Ref: "MO-15", Offset: 414, Actor: pClaude, Kind: evComment, Text: "Multi-restaurant cart implemented. Each restaurant's items become a separate sub-order. Combined checkout with a single payment, but the receipt shows per-restaurant breakdown. Delivery logistics handled by DL-2's driver assignment running independently for each sub-order."},
		{Ref: "MO-15", Offset: 415, Actor: pClaude, Kind: evStatus, Status: "DONE"},

		// PM-10: Stripe webhooks (Jake)
		{Ref: "PM-10", Offset: 412, Actor: pJake, Kind: evStatus, Status: "PLANNED"},

		// MO-16: search autocomplete (Whiskers)
		{Ref: "MO-16", Offset: 422, Actor: pWhiskers, Kind: evStatus, Status: "PLANNED"},
		{Ref: "MO-16", Offset: 424, Actor: pWhiskers, Kind: evStatus, Status: "DOING"},
		{Ref: "MO-16", Offset: 430, Actor: pWhiskers, Kind: evComment, Text: "Autocomplete search done! 300ms debounce, shows up to 8 suggestions in 3 categories: restaurants, menu items, dietary filters. Keyboard navigation works. Results update as you type -- feels snappy."},
		{Ref: "MO-16", Offset: 431, Actor: pWhiskers, Kind: evStatus, Status: "DONE"},

		// CAT-6: birthday celebration (Whiskers)
		{Ref: "CAT-6", Offset: 422, Actor: pWhiskers, Kind: evStatus, Status: "PLANNED"},
		{Ref: "CAT-6", Offset: 432, Actor: pWhiskers, Kind: evStatus, Status: "DOING"},
		{Ref: "CAT-6", Offset: 440, Actor: pWhiskers, Kind: evComment, Text: "Birthday feature is adorable! When it's a cat's birthday: confetti animation on the home screen, special Birthday Menu section with party-themed food, free birthday treat added to any order that day, and a little birthday hat on the cat's avatar. Tested with my cat Mittens' birthday data."},
		{Ref: "CAT-6", Offset: 441, Actor: pWhiskers, Kind: evStatus, Status: "DONE"},

		// BUG-3: driver location
		{Ref: "BUG-3", Offset: 432, Actor: pJake, Kind: evStatus, Status: "DOING"},
		{Ref: "BUG-3", Offset: 446, Actor: pJake, Kind: evComment, Text: "iOS background mode is tricky. Switched from WebSocket to a background location service that posts updates via HTTP every 10s when backgrounded. Battery impact is acceptable per testing."},
		{Ref: "BUG-3", Offset: 448, Actor: pJake, Kind: evStatus, Status: "DONE"},

		// DL-9: delivery time estimation (agent)
		{Ref: "DL-9", Offset: 432, Actor: pClaude, Kind: evStatus, Status: "PLANNED"},
		{Ref: "DL-9", Offset: 434, Actor: pClaude, Kind: evStatus, Status: "DOING"},
		{Ref: "DL-9", Offset: 442, Actor: pClaude, Kind: evStatus, Status: "DONE"},

		// MO-17: email templates (agent)
		{Ref: "MO-17", Offset: 442, Actor: pClaude, Kind: evStatus, Status: "PLANNED"},
		{Ref: "MO-17", Offset: 444, Actor: pClaude, Kind: evStatus, Status: "DOING"},
		{Ref: "MO-17", Offset: 450, Actor: pClaude, Kind: evStatus, Status: "DONE"},

		// CAT-7: weight tracking (Luna)
		{Ref: "CAT-7", Offset: 442, Actor: pLuna, Kind: evStatus, Status: "PLANNED"},
		{Ref: "CAT-7", Offset: 448, Actor: pLuna, Kind: evStatus, Status: "DOING"},
		{Ref: "CAT-7", Offset: 472, Actor: pLuna, Kind: evStatus, Status: "DONE"},

		// DL-10: driver push notifications (Whiskers)
		{Ref: "DL-10", Offset: 452, Actor: pWhiskers, Kind: evStatus, Status: "PLANNED"},
		{Ref: "DL-10", Offset: 454, Actor: pWhiskers, Kind: evStatus, Status: "DOING"},
		{Ref: "DL-10", Offset: 462, Actor: pWhiskers, Kind: evStatus, Status: "DONE"},

		// PM-6: PDF invoices (agent)
		{Ref: "PM-6", Offset: 452, Actor: pClaude, Kind: evStatus, Status: "PLANNED"},
		{Ref: "PM-6", Offset: 454, Actor: pClaude, Kind: evStatus, Status: "DOING"},
		{Ref: "PM-6", Offset: 462, Actor: pClaude, Kind: evComment, Text: "PDF invoice generation working. Using go-wkhtmltopdf for rendering. Each invoice includes itemized order breakdown, tax calculation, delivery fee, tip (if any), applied credits/discounts, and company details with order reference. Auto-generated on order completion and emailed to customer."},
		{Ref: "PM-6", Offset: 463, Actor: pClaude, Kind: evStatus, Status: "DONE"},

		// MO-18: query performance (agent)
		{Ref: "MO-18", Offset: 462, Actor: pDevin, Kind: evStatus, Status: "PLANNED"},
		{Ref: "MO-18", Offset: 464, Actor: pDevin, Kind: evStatus, Status: "DOING"},
		{Ref: "MO-18", Offset: 472, Actor: pDevin, Kind: evComment, Text: "Query audit complete. Fixed 4 N+1 issues in menu listing (was fetching modifiers per item -- batched to single query). Added composite index on orders(user_id, status). Slowest query went from 420ms to 18ms."},
		{Ref: "MO-18", Offset: 473, Actor: pDevin, Kind: evStatus, Status: "DONE"},

		// CAT-8: multi-cat order splitting (agent)
		{Ref: "CAT-8", Offset: 462, Actor: pDevin, Kind: evStatus, Status: "PLANNED"},
		{Ref: "CAT-8", Offset: 474, Actor: pDevin, Kind: evStatus, Status: "DOING"},
		{Ref: "CAT-8", Offset: 482, Actor: pDevin, Kind: evStatus, Status: "DONE"},

		// PM-7: tax calculation (agent)
		{Ref: "PM-7", Offset: 472, Actor: pDevin, Kind: evStatus, Status: "PLANNED"},
		{Ref: "PM-7", Offset: 484, Actor: pDevin, Kind: evStatus, Status: "DOING"},
		{Ref: "PM-7", Offset: 492, Actor: pDevin, Kind: evComment, Text: "TaxJar integration complete. Tax rates pulled by delivery zip code. Cached for 24 hours to minimize API calls. Handles state and local tax correctly for all US jurisdictions."},
		{Ref: "PM-7", Offset: 493, Actor: pDevin, Kind: evStatus, Status: "DONE"},

		// BUG-4: slow photos
		{Ref: "BUG-4", Offset: 472, Actor: pLuna, Kind: evStatus, Status: "DOING"},
		{Ref: "BUG-4", Offset: 480, Actor: pLuna, Kind: evComment, Text: "Added progressive image loading: show blurred 20px thumbnail immediately, then fade in the full resolution. Also added WebP format with JPEG fallback. Load time on 3G dropped from 8s to 1.2s."},
		{Ref: "BUG-4", Offset: 482, Actor: pLuna, Kind: evStatus, Status: "DONE"},

		// CAT-9: profile sharing (agent)
		{Ref: "CAT-9", Offset: 482, Actor: pClaude, Kind: evStatus, Status: "PLANNED"},
		{Ref: "CAT-9", Offset: 484, Actor: pClaude, Kind: evStatus, Status: "DOING"},
		{Ref: "CAT-9", Offset: 492, Actor: pClaude, Kind: evComment, Text: "Profile sharing implemented. Uses invite links with 7-day expiry. Three permission levels: viewer, orderer, admin. Admin can manage the cat's profile and dietary settings. Revokable at any time."},
		{Ref: "CAT-9", Offset: 493, Actor: pClaude, Kind: evStatus, Status: "DONE"},

		// MO-19: API versioning (agent)
		{Ref: "MO-19", Offset: 492, Actor: pDevin, Kind: evStatus, Status: "PLANNED"},
		{Ref: "MO-19", Offset: 494, Actor: pDevin, Kind: evStatus, Status: "DOING"},
		{Ref: "MO-19", Offset: 500, Actor: pDevin, Kind: evStatus, Status: "DONE"},

		// MO-20: E2E tests (Whiskers)
		{Ref: "MO-20", Offset: 502, Actor: pWhiskers, Kind: evStatus, Status: "PLANNED"},
		{Ref: "MO-20", Offset: 504, Actor: pWhiskers, Kind: evStatus, Status: "DOING"},
		{Ref: "MO-20", Offset: 512, Actor: pWhiskers, Kind: evComment, Text: "Full ordering E2E test suite done! 8 tests covering browse -> filter -> add to cart -> update quantity -> checkout -> enter card -> confirm -> verify confirmation page. Uses Stripe test card 4242. Takes about 45s to run."},
		{Ref: "MO-20", Offset: 513, Actor: pWhiskers, Kind: evStatus, Status: "DONE"},

		// MO-21: image optimization (agent)
		{Ref: "MO-21", Offset: 512, Actor: pDevin, Kind: evStatus, Status: "PLANNED"},
		{Ref: "MO-21", Offset: 514, Actor: pDevin, Kind: evStatus, Status: "DOING"},
		{Ref: "MO-21", Offset: 520, Actor: pDevin, Kind: evStatus, Status: "DONE"},

		// MO-22: pagination (agent)
		{Ref: "MO-22", Offset: 522, Actor: pClaude, Kind: evStatus, Status: "PLANNED"},
		{Ref: "MO-22", Offset: 524, Actor: pClaude, Kind: evStatus, Status: "DOING"},
		{Ref: "MO-22", Offset: 528, Actor: pClaude, Kind: evStatus, Status: "DONE"},

		// DL-8 continues (Jake)
		{Ref: "DL-8", Offset: 450, Actor: pJake, Kind: evComment, Text: "Driver app coming along. React Native with expo. Core screens done: order queue, active delivery map, earnings. Still need the photo upload and earnings history views."},

		// PM-5: subscription billing (Jake) -- continues
		{Ref: "PM-5", Offset: 448, Actor: pJake, Kind: evComment, Text: "Stripe Billing integration is complex but coming together. Trial period handling and subscription lifecycle events need careful testing."},

		// Sprint retro comment
		{Ref: "E2", Offset: 500, Actor: pSarah, Kind: evComment, Text: "Sprint 2 retro notes:\n\n**What went well:**\n- Menu browsing and ordering flow is feature-complete\n- Dietary filtering is getting great feedback from beta testers\n- Team velocity is picking up -- agents are doing incredible work\n\n**Concerns:**\n- Payment webhook handling still in progress\n- Driver app is behind schedule\n- Mood detection is still in research phase -- may need to descope for v1"},

		// Link PM-2 to PM-1
		{Ref: "PM-2", Offset: 242, Actor: pLuna, Kind: evLink, LinkType: "depends_on", TargetRef: "PM-1"},
	}
}

// ──────────────────────────────────────────────────────────────────────
// Phase 3: Production Incident (offset 552-696h, W20 ~30pts)
// ──────────────────────────────────────────────────────────────────────

func phase3Incident() []seedIssue {
	return []seedIssue{
		{Ref: "INC-1", Title: "INCIDENT: Double-charging customers on retry", Desc: "**Severity: P0**\n\nCustomers are being charged twice when the payment confirmation times out and the frontend retries. The retry creates a new PaymentIntent instead of confirming the existing one.\n\nImpact: 23 customers double-charged over the last 4 hours. Total overcharge: $847.50.\n\nTimeline:\n- 14:32 UTC - First customer complaint via support\n- 14:45 UTC - Pattern identified, 5 more complaints\n- 14:52 UTC - Engineering paged\n- 15:10 UTC - Root cause identified\n- 15:45 UTC - Hotfix deployed\n- 16:00 UTC - Refunds initiated for all affected customers", Labels: []string{"bug", "hotfix", "backend"}, Estimate: 1, Actor: pJake, Offset: 552},
		{Ref: "INC-2", Title: "Hotfix: add idempotency key to payment retry", Desc: "The payment flow must include an idempotency key tied to the order ID. If a PaymentIntent already exists for this order, confirm the existing one instead of creating a new one.", Labels: []string{"hotfix", "backend"}, Estimate: 1, Actor: pJake, Offset: 553},
		{Ref: "INC-3", Title: "Hotfix: add client-side retry backoff", Desc: "Frontend payment submission should use exponential backoff with jitter. Currently retries immediately on timeout, which compounds the double-charge issue.", Labels: []string{"hotfix", "frontend"}, Estimate: 1, Actor: pLuna, Offset: 553},
		{Ref: "INC-4", Title: "Batch refund all double-charged customers", Desc: "Write a script to identify all double-charged orders from the incident window and issue Stripe refunds. Send apology email with a $15 credit.", Labels: []string{"hotfix", "backend"}, Estimate: 2, Actor: pClaude, Offset: 554},
		{Ref: "INC-5", Title: "Write incident postmortem", Desc: "Full postmortem following the template: timeline, root cause, impact, remediation, action items. Share with the team and stakeholders.", Labels: []string{"improvement"}, Estimate: 2, Actor: pSarah, Offset: 564},
		{Ref: "INC-6", Title: "Add payment monitoring alerts", Desc: "Set up alerts for:\n- Duplicate charges per customer within 5 minutes\n- Payment error rate > 5%\n- Refund rate > 10%\n- Average payment latency > 3s", Labels: []string{"improvement", "infra"}, Estimate: 2, Actor: pDevin, Offset: 570},

		// Ongoing work during incident (slowed pace)
		{Ref: "MO-23", Title: "Mobile responsive design pass for all pages", Desc: "Review and fix responsive layout on all customer-facing pages. Test on iPhone SE, iPhone 14, Pixel 7, and iPad.", Labels: []string{"improvement", "frontend"}, Estimate: 3, Actor: pLuna, Offset: 580},
		{Ref: "MO-24", Title: "Build admin user management dashboard", Desc: "Internal admin dashboard for: user lookup, order history, credit adjustments, account suspension. Requires admin role.", Labels: []string{"feature", "frontend"}, Parent: "E1", Estimate: 5, Actor: pWhiskers, Offset: 590},
		{Ref: "MO-25", Title: "Implement soft delete and data retention policy", Desc: "Soft delete for users, cats, and orders. Hard delete after 90 days. GDPR-compliant data export endpoint.", Labels: []string{"improvement", "backend"}, Estimate: 3, Actor: pClaude, Offset: 600},
		{Ref: "MO-26", Title: "Build customer support contact form", Desc: "In-app help form with category selection (order issue, payment, account, other). Auto-attach order context if applicable.", Labels: []string{"feature", "frontend"}, Estimate: 2, Actor: pWhiskers, Offset: 620},
		{Ref: "MO-27", Title: "Implement order receipt printing for restaurants", Desc: "Thermal printer-friendly receipt format. Auto-print on order acceptance. Includes QR code for driver pickup verification.", Labels: []string{"feature", "backend"}, Parent: "E2", Estimate: 3, Actor: pJake, Offset: 630},
		{Ref: "MO-28", Title: "Add delivery radius validation on order placement", Desc: "Validate customer address is within restaurant's delivery radius before order submission. Show clear error with nearest alternatives.", Labels: []string{"feature", "backend"}, Parent: "E3", Estimate: 2, Actor: pDevin, Offset: 640},
		{Ref: "MO-29", Title: "Build notification preferences settings page", Desc: "User settings for: email notifications, push notifications, SMS. Per-category toggles: orders, promotions, recommendations.", Labels: []string{"feature", "frontend"}, Parent: "E5", Estimate: 3, Actor: pLuna, Offset: 650},
		{Ref: "MO-30", Title: "Implement restaurant menu scheduling", Desc: "Support for time-based menus: breakfast, lunch, dinner. Auto-switch visible menu based on time of day.", Labels: []string{"feature", "backend"}, Parent: "E2", Estimate: 3, Actor: pClaude, Offset: 660},
		{Ref: "MO-31", Title: "Add structured logging to all services", Desc: "Migrate from fmt.Printf to structured JSON logging with log levels. Include request_id, user_id, and trace_id in all log entries.", Labels: []string{"improvement", "backend"}, Estimate: 2, Actor: pDevin, Offset: 670},
		{Ref: "MO-32", Title: "Write unit tests for payment service", Desc: "Unit tests with mocked Stripe client for: charge creation, refund processing, webhook validation, subscription management.", Labels: []string{"improvement", "backend"}, Parent: "E4", Estimate: 3, Actor: pWhiskers, Offset: 680},
		{Ref: "MO-33", Title: "Implement driver rating system", Desc: "Customers rate drivers 1-5 stars after delivery. Running average displayed on driver profile. Low ratings trigger review.", Labels: []string{"feature", "backend"}, Parent: "E3", Estimate: 3, Actor: pClaude, Offset: 690},
	}
}

func phase3Events() []seedEvent {
	return []seedEvent{
		// Incident fires
		{Ref: "INC-1", Offset: 552, Actor: pJake, Kind: evStatus, Status: "DOING"},
		{Ref: "INC-1", Offset: 552, Actor: pSarah, Kind: evComment, Text: "All hands on deck. Jake is leading the investigation. Luna, check the frontend retry logic. Devin, pull the payment logs from the last 6 hours."},
		{Ref: "INC-1", Offset: 553, Actor: pJake, Kind: evComment, Text: "Found it. The checkout component creates a new `PaymentIntent` on every submission attempt. When the first intent succeeds but the response times out, the retry creates a second intent that also succeeds. Both charge the customer.\n\nThe fix is straightforward: use the order ID as Stripe's idempotency key, and on retry, retrieve the existing PaymentIntent instead of creating a new one."},
		{Ref: "INC-1", Offset: 554, Actor: pDevin, Kind: evComment, Text: "Payment logs confirm 23 affected orders between 10:15 and 14:52 UTC. All have exactly 2 successful charges. Total overcharge: $847.50."},

		// Hotfixes
		{Ref: "INC-2", Offset: 553, Actor: pJake, Kind: evStatus, Status: "DOING"},
		{Ref: "INC-2", Offset: 555, Actor: pJake, Kind: evComment, Text: "Fix deployed. Payment flow now:\n1. Create PaymentIntent with `idempotency_key = order_id`\n2. On retry, `GET /payment-intents?order_id=X` first\n3. If exists and status is `succeeded`, skip to confirmation\n4. If exists and status is `requires_confirmation`, confirm it\n5. Only create new intent if none exists"},
		{Ref: "INC-2", Offset: 556, Actor: pJake, Kind: evStatus, Status: "DONE"},
		{Ref: "INC-2", Offset: 556, Actor: pJake, Kind: evLink, LinkType: "blocks", TargetRef: "INC-1"},

		{Ref: "INC-3", Offset: 553, Actor: pLuna, Kind: evStatus, Status: "DOING"},
		{Ref: "INC-3", Offset: 555, Actor: pLuna, Kind: evComment, Text: "Added exponential backoff: 1s, 2s, 4s, 8s with +/-25% jitter. Max 3 retries. Also added a loading state that disables the pay button during submission."},
		{Ref: "INC-3", Offset: 556, Actor: pLuna, Kind: evStatus, Status: "DONE"},

		{Ref: "INC-4", Offset: 554, Actor: pClaude, Kind: evStatus, Status: "DOING"},
		{Ref: "INC-4", Offset: 558, Actor: pClaude, Kind: evComment, Text: "All 23 refunds processed via Stripe. Sent apology emails with $15 Upurr Credit to each affected customer. Automated script verified no additional double-charges outside the incident window."},
		{Ref: "INC-4", Offset: 559, Actor: pClaude, Kind: evStatus, Status: "DONE"},

		{Ref: "INC-1", Offset: 560, Actor: pJake, Kind: evStatus, Status: "DONE"},

		// Postmortem (Sarah)
		{Ref: "INC-5", Offset: 566, Actor: pSarah, Kind: evStatus, Status: "DOING"},
		{Ref: "INC-5", Offset: 578, Actor: pSarah, Kind: evComment, Text: "## Postmortem: Double-Charge Incident (Day 23)\n\n**Root Cause:** The payment checkout flow created a new Stripe PaymentIntent on every form submission, including retries. When the initial payment succeeded but the HTTP response timed out, the frontend retried and created a duplicate charge.\n\n**Contributing Factors:**\n- No idempotency key on payment creation\n- Frontend retry with no backoff (immediate retry on timeout)\n- No monitoring for duplicate charges\n- Payment webhook handling (PM-10) was still in progress -- we were relying on synchronous confirmation\n\n**Action Items:**\n1. Add idempotency keys to all payment operations (done)\n2. Implement retry backoff on frontend (done)\n3. Refund all affected customers (done)\n4. Prioritize payment webhook handling (PM-10)\n5. Add payment monitoring and alerting\n6. Review all other API calls for missing idempotency\n7. Implement end-to-end payment integration tests\n\n**Lessons Learned:** We should have caught this in testing. Adding payment scenarios to our integration test suite is critical before launch."},
		{Ref: "INC-5", Offset: 580, Actor: pSarah, Kind: evStatus, Status: "DONE"},

		// Payment monitoring (agent)
		{Ref: "INC-6", Offset: 572, Actor: pDevin, Kind: evStatus, Status: "PLANNED"},
		{Ref: "INC-6", Offset: 574, Actor: pDevin, Kind: evStatus, Status: "DOING"},
		{Ref: "INC-6", Offset: 582, Actor: pDevin, Kind: evComment, Text: "Payment monitoring alerts configured in PagerDuty:\n- Duplicate charge detection: alerts if same customer charged 2x within 5 min\n- Error rate: pages if payment errors > 5% over 10 min window\n- Refund rate: alerts if refund rate > 10% over 1 hour\n- Latency: alerts if p99 payment latency > 5s\n\nAll alerts route to the #payments-oncall Slack channel."},
		{Ref: "INC-6", Offset: 583, Actor: pDevin, Kind: evStatus, Status: "DONE"},

		// PM-10 gets prioritized (Jake)
		{Ref: "PM-10", Offset: 564, Actor: pSarah, Kind: evComment, Text: "Bumping priority on this. The payment incident showed we can't rely on synchronous payment confirmation. Webhook handling needs to be done before we go live."},
		{Ref: "PM-10", Offset: 566, Actor: pJake, Kind: evStatus, Status: "DOING"},
		{Ref: "PM-10", Offset: 590, Actor: pJake, Kind: evComment, Text: "Stripe webhook handler is live. Handles:\n- `payment_intent.succeeded` - mark order as paid\n- `payment_intent.payment_failed` - mark order as failed, notify customer\n- `charge.refunded` - update order status\n- `invoice.paid` - subscription billing confirmation\n\nIdempotent processing with event dedup by webhook event ID. Signature verification on all incoming webhooks."},
		{Ref: "PM-10", Offset: 592, Actor: pJake, Kind: evStatus, Status: "DONE"},

		// DL-8 completes (was in progress from phase 2)
		{Ref: "DL-8", Offset: 560, Actor: pJake, Kind: evComment, Text: "Driver app MVP is done but shipping is paused while we deal with the payment incident. Will pick back up after."},
		{Ref: "DL-8", Offset: 588, Actor: pJake, Kind: evStatus, Status: "DONE"},

		// PM-5 completes (Jake, was in progress)
		{Ref: "PM-5", Offset: 596, Actor: pJake, Kind: evComment, Text: "Upurr Premium subscription billing is live on staging. $9.99/month via Stripe Billing. Includes free delivery on all orders, priority driver assignment, early access to new restaurants, and 14-day free trial. Webhook handling for subscription events all working."},
		{Ref: "PM-5", Offset: 598, Actor: pJake, Kind: evStatus, Status: "DONE"},

		// Ongoing work during incident (slowed pace)
		{Ref: "MO-23", Offset: 584, Actor: pLuna, Kind: evStatus, Status: "DOING"},
		{Ref: "MO-23", Offset: 612, Actor: pLuna, Kind: evComment, Text: "Responsive pass complete. All customer pages tested on 5 device sizes. Key fixes: cart sidebar now slides in as a bottom sheet on mobile, menu grid switches from 3 to 2 to 1 columns, and the dietary filter chips scroll horizontally on small screens."},
		{Ref: "MO-23", Offset: 614, Actor: pLuna, Kind: evStatus, Status: "DONE"},

		{Ref: "MO-24", Offset: 592, Actor: pWhiskers, Kind: evStatus, Status: "PLANNED"},
		{Ref: "MO-24", Offset: 594, Actor: pWhiskers, Kind: evStatus, Status: "DOING"},
		{Ref: "MO-24", Offset: 604, Actor: pWhiskers, Kind: evComment, Text: "Admin dashboard MVP done! User search by email/name, full order history with timeline view, manual credit adjustment with audit log, and account suspension with reason tracking. Only accessible to admin role users."},
		{Ref: "MO-24", Offset: 605, Actor: pWhiskers, Kind: evStatus, Status: "DONE"},

		{Ref: "MO-25", Offset: 602, Actor: pClaude, Kind: evStatus, Status: "PLANNED"},
		{Ref: "MO-25", Offset: 604, Actor: pClaude, Kind: evStatus, Status: "DOING"},
		{Ref: "MO-25", Offset: 612, Actor: pClaude, Kind: evComment, Text: "Soft delete implemented for all major entities. `deleted_at` column with filtered indexes. Hard delete cron runs weekly, purges records older than 90 days. GDPR export endpoint returns all user data as a JSON download."},
		{Ref: "MO-25", Offset: 613, Actor: pClaude, Kind: evStatus, Status: "DONE"},

		{Ref: "MO-26", Offset: 622, Actor: pWhiskers, Kind: evStatus, Status: "PLANNED"},
		{Ref: "MO-26", Offset: 624, Actor: pWhiskers, Kind: evStatus, Status: "DOING"},
		{Ref: "MO-26", Offset: 632, Actor: pWhiskers, Kind: evComment, Text: "Contact form done! Categories: Order Issue, Payment Problem, Account Help, Restaurant Feedback, Other. If the user selects Order Issue, it shows a dropdown of their recent orders to attach context automatically. Submissions go to the support inbox."},
		{Ref: "MO-26", Offset: 633, Actor: pWhiskers, Kind: evStatus, Status: "DONE"},

		{Ref: "MO-27", Offset: 632, Actor: pJake, Kind: evStatus, Status: "PLANNED"},
		{Ref: "MO-27", Offset: 636, Actor: pJake, Kind: evStatus, Status: "DOING"},
		{Ref: "MO-27", Offset: 656, Actor: pJake, Kind: evComment, Text: "Receipt printing endpoint done. Generates thermal-printer-compatible HTML (58mm width, monospace font). QR code contains a signed URL for driver pickup verification -- driver scans it when collecting the order."},
		{Ref: "MO-27", Offset: 658, Actor: pJake, Kind: evStatus, Status: "DONE"},

		{Ref: "MO-28", Offset: 642, Actor: pDevin, Kind: evStatus, Status: "PLANNED"},
		{Ref: "MO-28", Offset: 644, Actor: pDevin, Kind: evStatus, Status: "DOING"},
		{Ref: "MO-28", Offset: 650, Actor: pDevin, Kind: evStatus, Status: "DONE"},

		{Ref: "MO-29", Offset: 652, Actor: pLuna, Kind: evStatus, Status: "PLANNED"},
		{Ref: "MO-29", Offset: 660, Actor: pLuna, Kind: evStatus, Status: "DOING"},
		{Ref: "MO-29", Offset: 682, Actor: pLuna, Kind: evComment, Text: "Notification preferences page done. Toggle switches for: order updates (email/push), promotional offers (email/push/SMS), feeding reminders (push), weekly digest (email). All stored per-user with immediate effect."},
		{Ref: "MO-29", Offset: 684, Actor: pLuna, Kind: evStatus, Status: "DONE"},

		{Ref: "MO-30", Offset: 662, Actor: pClaude, Kind: evStatus, Status: "PLANNED"},
		{Ref: "MO-30", Offset: 664, Actor: pClaude, Kind: evStatus, Status: "DOING"},
		{Ref: "MO-30", Offset: 672, Actor: pClaude, Kind: evComment, Text: "Menu scheduling implemented. Restaurants define time slots for each menu category. Breakfast items auto-hide after 11 AM, lunch from 11-4, dinner after 4. Customers see \"Available at [time]\" for items outside their window."},
		{Ref: "MO-30", Offset: 673, Actor: pClaude, Kind: evStatus, Status: "DONE"},

		{Ref: "MO-31", Offset: 672, Actor: pDevin, Kind: evStatus, Status: "PLANNED"},
		{Ref: "MO-31", Offset: 674, Actor: pDevin, Kind: evStatus, Status: "DOING"},
		{Ref: "MO-31", Offset: 682, Actor: pDevin, Kind: evStatus, Status: "DONE"},

		{Ref: "MO-32", Offset: 682, Actor: pWhiskers, Kind: evStatus, Status: "PLANNED"},
		{Ref: "MO-32", Offset: 684, Actor: pWhiskers, Kind: evStatus, Status: "DOING"},
		{Ref: "MO-32", Offset: 694, Actor: pWhiskers, Kind: evComment, Text: "Payment service unit tests complete! 52 tests with mocked Stripe client. Covers charge creation, partial refunds, webhook signature validation, subscription lifecycle, and idempotency key handling. 94% line coverage. This would have caught the double-charge bug."},
		{Ref: "MO-32", Offset: 695, Actor: pWhiskers, Kind: evStatus, Status: "DONE"},

		{Ref: "MO-33", Offset: 692, Actor: pClaude, Kind: evStatus, Status: "PLANNED"},
		{Ref: "MO-33", Offset: 694, Actor: pClaude, Kind: evStatus, Status: "DOING"},

		// CAT-4: recommendations engine (Sarah, continues)
		{Ref: "CAT-4", Offset: 580, Actor: pSarah, Kind: evStatus, Status: "DOING"},
	}
}

// ──────────────────────────────────────────────────────────────────────
// Phase 4: Feature Descope (offset 696-864h, W21 ~50pts)
// ──────────────────────────────────────────────────────────────────────

func phase4Descope() []seedIssue {
	return []seedIssue{
		{Ref: "MO-34", Title: "Split-order delivery coordination", Desc: "When a cart has items from multiple restaurants, coordinate separate deliveries. Show combined tracking view.", Labels: []string{"feature", "backend"}, Parent: "E2", Estimate: 3, Actor: pClaude, Offset: 700},
		{Ref: "MO-35", Title: "Favorite restaurants and quick-order shortcuts", Desc: "Save favorite restaurants. Quick-order buttons for frequently ordered items on the home screen.", Labels: []string{"feature", "frontend"}, Parent: "E2", Estimate: 2, Actor: pWhiskers, Offset: 710},
		{Ref: "MO-36", Title: "Push notification for order status changes", Desc: "Mobile push notifications at each order state transition: confirmed, preparing, picked up, nearby, delivered.", Labels: []string{"feature", "mobile"}, Parent: "E2", Estimate: 2, Actor: pDevin, Offset: 720},
		{Ref: "MO-37", Title: "Build restaurant analytics dashboard", Desc: "Charts for restaurant owners: daily orders, revenue, popular items, average prep time, customer ratings over time.", Labels: []string{"feature", "frontend"}, Parent: "E2", Estimate: 3, Actor: pLuna, Offset: 740},
		{Ref: "MO-38", Title: "Add CORS and security headers middleware", Desc: "Strict CORS policy, X-Content-Type-Options, X-Frame-Options, Content-Security-Policy, and Strict-Transport-Security headers.", Labels: []string{"improvement", "backend"}, Estimate: 1, Actor: pDevin, Offset: 760},
		{Ref: "MO-39", Title: "Post-incident error handling audit", Desc: "Review all API endpoints for proper error handling. Ensure no panic-recoverable errors leak stack traces to clients.", Labels: []string{"improvement", "backend"}, Estimate: 3, Actor: pSarah, Offset: 770},
		{Ref: "MO-40", Title: "Add retry logic to all external API calls", Desc: "Configurable retry with exponential backoff for: Stripe, Google Maps, TaxJar, SendGrid. Max 3 retries, circuit breaker integration.", Labels: []string{"improvement", "backend"}, Estimate: 3, Actor: pClaude, Offset: 780},
		{Ref: "MO-41", Title: "Implement request timeout configuration", Desc: "Per-service configurable timeouts. Default 5s for API calls, 30s for payment operations, 10s for external services.", Labels: []string{"improvement", "backend"}, Estimate: 2, Actor: pDevin, Offset: 790},
		{Ref: "MO-42", Title: "Write runbook for payment service recovery", Desc: "Step-by-step runbook for: payment service restart, Stripe webhook backfill, reconciliation check, and customer notification.", Labels: []string{"improvement"}, Estimate: 2, Actor: pJake, Offset: 800},
		{Ref: "MO-43", Title: "Add smoke tests for critical API endpoints", Desc: "Lightweight smoke test suite that runs every 5 minutes in production. Tests: auth, menu fetch, order creation (dry run), payment health.", Labels: []string{"improvement", "infra"}, Estimate: 3, Actor: pWhiskers, Offset: 810},
	}
}

func phase4Events() []seedEvent {
	return []seedEvent{
		// Descope decision
		{Ref: "E6", Offset: 700, Actor: pSarah, Kind: evComment, Text: "After the payment incident and the timeline impact, we're making a hard call: **Cat Mood Detection is descoped from v1.** We need to focus all engineering effort on launch readiness.\n\nThe research (MD-1) was promising and we'll pick this back up post-launch. Moving all remaining mood detection tickets back to BACKLOG."},
		{Ref: "E6", Offset: 701, Actor: pSarah, Kind: evStatus, Status: "BACKLOG"},
		{Ref: "MD-2", Offset: 701, Actor: pSarah, Kind: evStatus, Status: "BACKLOG"},
		{Ref: "MD-3", Offset: 701, Actor: pSarah, Kind: evStatus, Status: "BACKLOG"},
		{Ref: "MD-4", Offset: 701, Actor: pSarah, Kind: evStatus, Status: "BACKLOG"},
		{Ref: "MD-5", Offset: 701, Actor: pSarah, Kind: evStatus, Status: "BACKLOG"},
		{Ref: "MD-6", Offset: 701, Actor: pSarah, Kind: evStatus, Status: "BACKLOG"},

		{Ref: "E6", Offset: 702, Actor: pJake, Kind: evComment, Text: "Right call. The mood detection feature is genuinely cool but it's a nice-to-have for launch. Our paying customers need reliable ordering and payments first. We'll come back to it in Q2."},
		{Ref: "E6", Offset: 703, Actor: pLuna, Kind: evComment, Text: "Understood. I had the mood-based UI designs mostly done but they'll keep. Redirecting my time to polish the core ordering experience and the cat profile cards."},

		// MO-33 completes (from phase 3)
		{Ref: "MO-33", Offset: 702, Actor: pClaude, Kind: evComment, Text: "Driver rating system implemented. Post-delivery 1-5 star rating. Weighted running average (recent ratings count more). Drivers below 3.5 average get flagged for review. Rating shown to customers during driver assignment."},
		{Ref: "MO-33", Offset: 703, Actor: pClaude, Kind: evStatus, Status: "DONE"},

		// MO-34: split-order delivery (agent)
		{Ref: "MO-34", Offset: 702, Actor: pClaude, Kind: evStatus, Status: "PLANNED"},
		{Ref: "MO-34", Offset: 704, Actor: pClaude, Kind: evStatus, Status: "DOING"},
		{Ref: "MO-34", Offset: 712, Actor: pClaude, Kind: evComment, Text: "Split-order delivery coordination implemented. Each sub-order gets its own driver. Customer sees a combined tracking view with multiple pins on the map. ETAs are independent per restaurant."},
		{Ref: "MO-34", Offset: 713, Actor: pClaude, Kind: evStatus, Status: "DONE"},

		// CAT-4: recommendations engine (Sarah, completes)
		{Ref: "CAT-4", Offset: 710, Actor: pSarah, Kind: evComment, Text: "Recommendation engine MVP done. Uses a simple but effective approach:\n1. Filter by cat's dietary restrictions (hard filters)\n2. Boost items matching breed-typical preferences\n3. Weight by order history (collaborative filtering light)\n4. Diversify results to avoid echo chamber\n\nPrecision isn't perfect yet but the suggestions are relevant. Will improve with more data post-launch."},
		{Ref: "CAT-4", Offset: 712, Actor: pSarah, Kind: evStatus, Status: "DONE"},

		// MO-35: favorites (Whiskers)
		{Ref: "MO-35", Offset: 712, Actor: pWhiskers, Kind: evStatus, Status: "PLANNED"},
		{Ref: "MO-35", Offset: 714, Actor: pWhiskers, Kind: evStatus, Status: "DOING"},
		{Ref: "MO-35", Offset: 722, Actor: pWhiskers, Kind: evStatus, Status: "DONE"},

		// MO-36: push notifications (agent)
		{Ref: "MO-36", Offset: 722, Actor: pDevin, Kind: evStatus, Status: "PLANNED"},
		{Ref: "MO-36", Offset: 724, Actor: pDevin, Kind: evStatus, Status: "DOING"},
		{Ref: "MO-36", Offset: 730, Actor: pDevin, Kind: evComment, Text: "Push notifications for order status changes working on both iOS and Android. Used Firebase Cloud Messaging. Notifications include the cat avatar as the icon if the user has a cat profile."},
		{Ref: "MO-36", Offset: 731, Actor: pDevin, Kind: evStatus, Status: "DONE"},

		// MO-37: restaurant analytics (Luna)
		{Ref: "MO-37", Offset: 742, Actor: pLuna, Kind: evStatus, Status: "PLANNED"},
		{Ref: "MO-37", Offset: 748, Actor: pLuna, Kind: evStatus, Status: "DOING"},
		{Ref: "MO-37", Offset: 770, Actor: pLuna, Kind: evComment, Text: "Restaurant analytics dashboard live. Charts: daily order volume (bar), revenue trend (line), top 10 items (horizontal bar), average prep time (gauge), rating distribution (pie). Date range selector with last 7d/30d/90d presets."},
		{Ref: "MO-37", Offset: 772, Actor: pLuna, Kind: evStatus, Status: "DONE"},

		// MO-38: CORS headers (agent, quick)
		{Ref: "MO-38", Offset: 762, Actor: pDevin, Kind: evStatus, Status: "PLANNED"},
		{Ref: "MO-38", Offset: 764, Actor: pDevin, Kind: evStatus, Status: "DOING"},
		{Ref: "MO-38", Offset: 768, Actor: pDevin, Kind: evStatus, Status: "DONE"},

		// MO-39: error handling audit (Sarah)
		{Ref: "MO-39", Offset: 772, Actor: pSarah, Kind: evStatus, Status: "PLANNED"},
		{Ref: "MO-39", Offset: 776, Actor: pSarah, Kind: evStatus, Status: "DOING"},
		{Ref: "MO-39", Offset: 798, Actor: pSarah, Kind: evComment, Text: "Error handling audit complete. Found 7 endpoints where panics could leak stack traces -- wrapped with recover middleware. Also found 3 places where database errors were returned directly to clients -- now mapped to generic 500 responses with request IDs for internal debugging."},
		{Ref: "MO-39", Offset: 800, Actor: pSarah, Kind: evStatus, Status: "DONE"},

		// MO-40: retry logic (agent)
		{Ref: "MO-40", Offset: 782, Actor: pClaude, Kind: evStatus, Status: "PLANNED"},
		{Ref: "MO-40", Offset: 784, Actor: pClaude, Kind: evStatus, Status: "DOING"},
		{Ref: "MO-40", Offset: 792, Actor: pClaude, Kind: evComment, Text: "Retry logic added to all external calls. Using exponential backoff: 100ms, 200ms, 400ms base delays with +/-25% jitter. Retries only on 5xx and timeout errors. Circuit breaker integration: if circuit is open, skip retries and fail fast."},
		{Ref: "MO-40", Offset: 793, Actor: pClaude, Kind: evStatus, Status: "DONE"},

		// MO-41: request timeouts (agent)
		{Ref: "MO-41", Offset: 792, Actor: pDevin, Kind: evStatus, Status: "PLANNED"},
		{Ref: "MO-41", Offset: 794, Actor: pDevin, Kind: evStatus, Status: "DOING"},
		{Ref: "MO-41", Offset: 800, Actor: pDevin, Kind: evStatus, Status: "DONE"},

		// MO-42: payment runbook (Jake)
		{Ref: "MO-42", Offset: 802, Actor: pJake, Kind: evStatus, Status: "PLANNED"},
		{Ref: "MO-42", Offset: 808, Actor: pJake, Kind: evStatus, Status: "DOING"},
		{Ref: "MO-42", Offset: 822, Actor: pJake, Kind: evStatus, Status: "DONE"},

		// MO-43: smoke tests (Whiskers)
		{Ref: "MO-43", Offset: 812, Actor: pWhiskers, Kind: evStatus, Status: "PLANNED"},
		{Ref: "MO-43", Offset: 814, Actor: pWhiskers, Kind: evStatus, Status: "DOING"},
		{Ref: "MO-43", Offset: 824, Actor: pWhiskers, Kind: evComment, Text: "Smoke tests running every 5 min via a dedicated ECS task. Tests: GET /healthz on all services, POST /auth/login with test creds, GET /restaurants (verify non-empty), GET /menu/{id} (verify schema), and Stripe API connectivity check. Alerts to PagerDuty on failure."},
		{Ref: "MO-43", Offset: 825, Actor: pWhiskers, Kind: evStatus, Status: "DONE"},

		// Epic completions
		{Ref: "E4", Offset: 830, Actor: pSarah, Kind: evComment, Text: "Payments epic is done. Stripe integration, webhooks, refunds, subscriptions, invoicing, and tax calculation all shipped. The incident taught us hard lessons but the system is much more robust now."},
		{Ref: "E4", Offset: 830, Actor: pSarah, Kind: evStatus, Status: "DONE"},

		{Ref: "E5", Offset: 840, Actor: pLuna, Kind: evComment, Text: "Cat profiles epic complete. Profiles, dietary tracking, allergies, recommendations, feeding schedules, birthdays, sharing -- all done. The cat experience is going to be delightful."},
		{Ref: "E5", Offset: 840, Actor: pLuna, Kind: evStatus, Status: "DONE"},

		{Ref: "E2", Offset: 850, Actor: pSarah, Kind: evComment, Text: "Menu & ordering is done! All features shipped: browsing, dietary filters, cart, checkout, order tracking, restaurant dashboard, multi-restaurant orders. Ready for launch."},
		{Ref: "E2", Offset: 850, Actor: pSarah, Kind: evStatus, Status: "DONE"},

		{Ref: "E3", Offset: 855, Actor: pJake, Kind: evStatus, Status: "DONE"},
		{Ref: "E1", Offset: 860, Actor: pSarah, Kind: evComment, Text: "Core platform is feature-complete. Auth, API gateway, DB, logging, monitoring -- all solid. Moving on to launch prep."},
		{Ref: "E1", Offset: 860, Actor: pSarah, Kind: evStatus, Status: "DONE"},
	}
}

// ──────────────────────────────────────────────────────────────────────
// Phase 5: Refactor / Tech Debt (offset 864-1056h, W22 ~60pts)
// ──────────────────────────────────────────────────────────────────────

func phase5Refactor() []seedIssue {
	return []seedIssue{
		{Ref: "TD-1", Title: "Add end-to-end payment integration tests", Desc: "Full integration test suite for the payment flow using Stripe's test mode. Cover: successful payment, failed payment, timeout + retry, refund, subscription billing.", Labels: []string{"improvement", "backend"}, Parent: "E7", Estimate: 5, Actor: pWhiskers, Offset: 888},
		{Ref: "TD-2", Title: "Implement circuit breaker for external APIs", Desc: "Circuit breaker pattern for Stripe, Google Maps, TaxJar, and SendGrid. Prevent cascade failures when external services are down.", Labels: []string{"improvement", "backend"}, Parent: "E7", Estimate: 3, Actor: pSarah, Offset: 892},
		{Ref: "TD-3", Title: "Add idempotency keys to all write endpoints", Desc: "Post-incident action item: every state-changing API endpoint should accept an idempotency key. Dedup window: 24 hours.", Labels: []string{"improvement", "backend"}, Parent: "E7", Estimate: 3, Actor: pClaude, Offset: 896},
		{Ref: "TD-4", Title: "Database connection pooling and query optimization", Desc: "Review and optimize slow queries identified by OpenTelemetry. Implement pgBouncer for connection pooling. Target: p99 query latency < 50ms.", Labels: []string{"improvement", "backend"}, Parent: "E7", Estimate: 3, Actor: pDevin, Offset: 900},
		{Ref: "TD-5", Title: "Implement graceful shutdown for all services", Desc: "Handle SIGTERM gracefully: drain in-flight requests, close DB connections, flush logs. 30s timeout before force kill.", Labels: []string{"improvement", "backend"}, Parent: "E7", Estimate: 2, Actor: pClaude, Offset: 910},
		{Ref: "TD-6", Title: "Add structured error tracking with Sentry", Desc: "Sentry integration for error reporting with source maps (frontend) and stack traces (backend). Group by root cause. Alert on new error types.", Labels: []string{"improvement", "infra"}, Parent: "E7", Estimate: 3, Actor: pDevin, Offset: 920},
		{Ref: "TD-7", Title: "Review and harden rate limiting configuration", Desc: "Post-launch rate limits: tighten per-IP limits, add per-user limits, special limits for payment endpoints. DDoS mitigation review.", Labels: []string{"improvement", "infra"}, Parent: "E7", Estimate: 2, Actor: pDevin, Offset: 930},
		{Ref: "TD-8", Title: "Add database backup and restore procedure", Desc: "Automated daily backups to S3 with point-in-time recovery. Document and test the restore procedure. Retention: 30 days.", Labels: []string{"improvement", "infra"}, Parent: "E7", Estimate: 2, Actor: pJake, Offset: 940},
		{Ref: "TD-9", Title: "Implement request replay for failed webhooks", Desc: "Dead letter queue for failed webhook deliveries. Admin UI to inspect and replay. Max 5 retry attempts with exponential backoff.", Labels: []string{"improvement", "backend"}, Parent: "E7", Estimate: 3, Actor: pClaude, Offset: 950},
		{Ref: "TD-10", Title: "Load testing with realistic traffic patterns", Desc: "k6 load test scripts simulating: browsing (60%), ordering (25%), payment (10%), driver updates (5%). Target: 500 concurrent users with p99 < 200ms.", Labels: []string{"improvement", "infra"}, Parent: "E7", Estimate: 5, Actor: pJake, Offset: 960},
		{Ref: "TD-11", Title: "API documentation with OpenAPI/Swagger", Desc: "Generate OpenAPI 3.0 spec for all public endpoints. Host Swagger UI at /docs. Include request/response examples and authentication details.", Labels: []string{"improvement", "backend"}, Parent: "E7", Estimate: 2, Actor: pDevin, Offset: 970},
		{Ref: "TD-12", Title: "Implement feature flags with LaunchDarkly", Desc: "Feature flag system for gradual rollouts. Priority flags: premium_features, multi_restaurant_cart, recommendation_engine.", Labels: []string{"improvement", "backend"}, Parent: "E7", Estimate: 3, Actor: pClaude, Offset: 980},
		{Ref: "TD-13", Title: "Automated dependency vulnerability scanning", Desc: "Integrate Snyk or Dependabot for Go and npm dependency scanning. Auto-create PRs for security patches. Block merges on critical CVEs.", Labels: []string{"improvement", "infra"}, Parent: "E7", Estimate: 2, Actor: pWhiskers, Offset: 990},
		{Ref: "TD-14", Title: "Implement structured request/response logging", Desc: "Log sanitized request/response bodies for debugging. Redact PII fields (email, phone, card). Configurable per-endpoint verbosity.", Labels: []string{"improvement", "backend"}, Parent: "E7", Estimate: 2, Actor: pDevin, Offset: 1000},
		{Ref: "TD-15", Title: "Add database migration rollback scripts", Desc: "Write down-migration for every up-migration. Test rollback procedure. Document in runbook.", Labels: []string{"improvement", "backend"}, Parent: "E7", Estimate: 2, Actor: pClaude, Offset: 1010},
		{Ref: "TD-16", Title: "Implement service mesh health monitoring", Desc: "Cross-service health dashboard showing dependency graph, latency between services, and error propagation paths.", Labels: []string{"improvement", "infra"}, Parent: "E7", Estimate: 3, Actor: pWhiskers, Offset: 1020},
		{Ref: "TD-17", Title: "Performance profiling and memory leak audit", Desc: "Run pprof on all Go services under load. Check for goroutine leaks, memory growth, and CPU hotspots. Fix any issues found.", Labels: []string{"improvement", "backend"}, Parent: "E7", Estimate: 3, Actor: pSarah, Offset: 1030},
		{Ref: "TD-18", Title: "Add chaos engineering test suite", Desc: "Simulate failures: kill random service instances, inject network latency, corrupt responses. Verify graceful degradation.", Labels: []string{"improvement", "infra"}, Parent: "E7", Estimate: 3, Actor: pDevin, Offset: 1040},
	}
}

func phase5Events() []seedEvent {
	return []seedEvent{
		{Ref: "E7", Offset: 890, Actor: pSarah, Kind: evStatus, Status: "PLANNED"},
		{Ref: "E7", Offset: 890, Actor: pSarah, Kind: evComment, Text: "Starting the tech debt sprint. The payment incident exposed gaps in our reliability story. Every item here is an action item from the postmortem or a risk we've been carrying.\n\nPriority order:\n1. Payment integration tests (never again)\n2. Circuit breakers (prevent cascade failures)\n3. Idempotency (prevent duplicate operations)\n4. Everything else by risk/effort"},
		{Ref: "E7", Offset: 892, Actor: pSarah, Kind: evStatus, Status: "DOING"},

		// TD-1: payment integration tests (Whiskers)
		{Ref: "TD-1", Offset: 890, Actor: pWhiskers, Kind: evStatus, Status: "PLANNED"},
		{Ref: "TD-1", Offset: 892, Actor: pWhiskers, Kind: evStatus, Status: "DOING"},
		{Ref: "TD-1", Offset: 900, Actor: pWhiskers, Kind: evComment, Text: "Payment test suite complete! 47 test cases covering:\n- Happy path: card payment, saved card, Apple Pay\n- Failures: declined card, insufficient funds, expired card\n- Edge cases: timeout + retry (the incident scenario), concurrent payments, refund after partial delivery\n- Subscription: create, renew, cancel, payment failure\n\nAll tests run against Stripe test mode in CI. Takes about 90 seconds."},
		{Ref: "TD-1", Offset: 901, Actor: pWhiskers, Kind: evStatus, Status: "DONE"},

		// TD-2: circuit breakers (Sarah)
		{Ref: "TD-2", Offset: 894, Actor: pSarah, Kind: evStatus, Status: "PLANNED"},
		{Ref: "TD-2", Offset: 900, Actor: pSarah, Kind: evStatus, Status: "DOING"},
		{Ref: "TD-2", Offset: 920, Actor: pSarah, Kind: evComment, Text: "Circuit breakers implemented using `sony/gobreaker`. Configuration per service:\n- Stripe: 5 failures in 10s -> open for 30s\n- Google Maps: 3 failures in 10s -> open for 60s\n- TaxJar: 3 failures in 10s -> open for 120s (cache fallback)\n- SendGrid: 5 failures in 30s -> open for 60s (queue fallback)\n\nWhen circuits open, we return cached data where possible or a graceful degradation response."},
		{Ref: "TD-2", Offset: 922, Actor: pSarah, Kind: evStatus, Status: "DONE"},

		// TD-3: idempotency keys (agent)
		{Ref: "TD-3", Offset: 898, Actor: pClaude, Kind: evStatus, Status: "PLANNED"},
		{Ref: "TD-3", Offset: 900, Actor: pClaude, Kind: evStatus, Status: "DOING"},
		{Ref: "TD-3", Offset: 908, Actor: pClaude, Kind: evComment, Text: "All write endpoints now accept `X-Idempotency-Key` header. Implementation: keys stored in Redis with 24h TTL. First request executes and caches response. Duplicate request returns cached response with 200. Conflict (different payload, same key) returns 409. Endpoints covered: order creation, payment, refund, cart updates, profile changes."},
		{Ref: "TD-3", Offset: 909, Actor: pClaude, Kind: evStatus, Status: "DONE"},

		// TD-4: DB optimization (agent)
		{Ref: "TD-4", Offset: 902, Actor: pDevin, Kind: evStatus, Status: "PLANNED"},
		{Ref: "TD-4", Offset: 904, Actor: pDevin, Kind: evStatus, Status: "DOING"},
		{Ref: "TD-4", Offset: 914, Actor: pDevin, Kind: evComment, Text: "Query optimization done:\n- Added composite index on `orders(user_id, created_at)` -- order history query went from 340ms to 12ms\n- Added index on `menu_items(restaurant_id, dietary_tags)` -- menu filtering 5x faster\n- Switched to pgBouncer: connection pool size 20, max 50 in burst\n- p99 query latency is now 28ms (was 180ms)"},
		{Ref: "TD-4", Offset: 915, Actor: pDevin, Kind: evStatus, Status: "DONE"},

		// TD-5: graceful shutdown (agent)
		{Ref: "TD-5", Offset: 912, Actor: pClaude, Kind: evStatus, Status: "PLANNED"},
		{Ref: "TD-5", Offset: 914, Actor: pClaude, Kind: evStatus, Status: "DOING"},
		{Ref: "TD-5", Offset: 920, Actor: pClaude, Kind: evStatus, Status: "DONE"},

		// TD-6: Sentry (agent)
		{Ref: "TD-6", Offset: 922, Actor: pDevin, Kind: evStatus, Status: "PLANNED"},
		{Ref: "TD-6", Offset: 924, Actor: pDevin, Kind: evStatus, Status: "DOING"},
		{Ref: "TD-6", Offset: 934, Actor: pDevin, Kind: evComment, Text: "Sentry is live for both frontend and backend. Source maps uploaded in CI for readable stack traces. Error grouping configured. Slack integration posts new issues to #errors. Already caught a null pointer in the recommendation engine that would have been hard to debug in prod."},
		{Ref: "TD-6", Offset: 935, Actor: pDevin, Kind: evStatus, Status: "DONE"},

		// TD-7: rate limiting (agent)
		{Ref: "TD-7", Offset: 932, Actor: pDevin, Kind: evStatus, Status: "PLANNED"},
		{Ref: "TD-7", Offset: 936, Actor: pDevin, Kind: evStatus, Status: "DOING"},
		{Ref: "TD-7", Offset: 944, Actor: pDevin, Kind: evComment, Text: "Rate limits tightened:\n- General API: 100 req/min per user\n- Payment endpoints: 10 req/min per user\n- Auth endpoints: 5 req/min per IP (brute force protection)\n- Search: 30 req/min per user\n- Webhook ingress: IP allowlist for Stripe/Google"},
		{Ref: "TD-7", Offset: 945, Actor: pDevin, Kind: evStatus, Status: "DONE"},

		// TD-8: backups (Jake)
		{Ref: "TD-8", Offset: 942, Actor: pJake, Kind: evStatus, Status: "PLANNED"},
		{Ref: "TD-8", Offset: 948, Actor: pJake, Kind: evStatus, Status: "DOING"},
		{Ref: "TD-8", Offset: 966, Actor: pJake, Kind: evComment, Text: "Backups automated. Daily snapshots at 03:00 UTC to S3 with AES-256 encryption. Point-in-time recovery up to 5 minutes via WAL archiving. Tested full restore -- took 4 minutes for our current data size. Runbook documented."},
		{Ref: "TD-8", Offset: 968, Actor: pJake, Kind: evStatus, Status: "DONE"},

		// TD-9: webhook replay (agent)
		{Ref: "TD-9", Offset: 952, Actor: pClaude, Kind: evStatus, Status: "PLANNED"},
		{Ref: "TD-9", Offset: 954, Actor: pClaude, Kind: evStatus, Status: "DOING"},
		{Ref: "TD-9", Offset: 962, Actor: pClaude, Kind: evComment, Text: "Webhook dead letter queue implemented using SQS. Failed deliveries stored with full payload and headers. Admin endpoint allows listing failed webhooks with filters, inspecting payloads, replaying individual events, and bulk replay by time range. Max 5 retries with exponential backoff: 1m, 5m, 30m, 2h, 12h."},
		{Ref: "TD-9", Offset: 963, Actor: pClaude, Kind: evStatus, Status: "DONE"},

		// TD-10: load testing (Jake)
		{Ref: "TD-10", Offset: 962, Actor: pJake, Kind: evStatus, Status: "PLANNED"},
		{Ref: "TD-10", Offset: 968, Actor: pJake, Kind: evStatus, Status: "DOING"},
		{Ref: "TD-10", Offset: 992, Actor: pJake, Kind: evComment, Text: "Load test results (k6, 500 concurrent users, 10 min run):\n\n| Endpoint | p50 | p95 | p99 | Error Rate |\n|---|---|---|---|---|\n| GET /restaurants | 45ms | 120ms | 180ms | 0.0% |\n| GET /menu/:id | 38ms | 95ms | 145ms | 0.0% |\n| POST /orders | 120ms | 280ms | 420ms | 0.1% |\n| POST /payments | 850ms | 1.8s | 2.4s | 0.3% |\n| WS /tracking | 12ms | 35ms | 60ms | 0.0% |\n\nPayment latency is high because of Stripe round-trip. Everything else is well within target. Database handles the load fine with pgBouncer."},
		{Ref: "TD-10", Offset: 994, Actor: pJake, Kind: evStatus, Status: "DONE"},

		// TD-11: API docs (agent)
		{Ref: "TD-11", Offset: 972, Actor: pDevin, Kind: evStatus, Status: "PLANNED"},
		{Ref: "TD-11", Offset: 974, Actor: pDevin, Kind: evStatus, Status: "DOING"},
		{Ref: "TD-11", Offset: 982, Actor: pDevin, Kind: evStatus, Status: "DONE"},

		// TD-12: feature flags (agent)
		{Ref: "TD-12", Offset: 982, Actor: pClaude, Kind: evStatus, Status: "PLANNED"},
		{Ref: "TD-12", Offset: 984, Actor: pClaude, Kind: evStatus, Status: "DOING"},
		{Ref: "TD-12", Offset: 994, Actor: pClaude, Kind: evComment, Text: "LaunchDarkly integrated. Feature flags in place:\n- `premium_features` - gate Upurr Premium features\n- `multi_restaurant_cart` - gradual rollout of split orders\n- `recommendation_engine` - toggle personalized recommendations\n- `birthday_feature` - cat birthday celebrations\n\nAll flags default to off in production. We'll turn them on progressively during launch week."},
		{Ref: "TD-12", Offset: 995, Actor: pClaude, Kind: evStatus, Status: "DONE"},

		// TD-13: dependency scanning (Whiskers)
		{Ref: "TD-13", Offset: 992, Actor: pWhiskers, Kind: evStatus, Status: "PLANNED"},
		{Ref: "TD-13", Offset: 994, Actor: pWhiskers, Kind: evStatus, Status: "DOING"},
		{Ref: "TD-13", Offset: 1002, Actor: pWhiskers, Kind: evComment, Text: "Snyk integrated into CI pipeline. Scans Go modules and npm packages on every PR. Found 2 medium-severity CVEs in transitive dependencies -- patched by bumping parent packages. Auto-PR creation enabled for critical and high severity findings."},
		{Ref: "TD-13", Offset: 1003, Actor: pWhiskers, Kind: evStatus, Status: "DONE"},

		// TD-14: request/response logging (agent)
		{Ref: "TD-14", Offset: 1002, Actor: pDevin, Kind: evStatus, Status: "PLANNED"},
		{Ref: "TD-14", Offset: 1004, Actor: pDevin, Kind: evStatus, Status: "DOING"},
		{Ref: "TD-14", Offset: 1012, Actor: pDevin, Kind: evStatus, Status: "DONE"},

		// TD-15: migration rollback (agent)
		{Ref: "TD-15", Offset: 1012, Actor: pClaude, Kind: evStatus, Status: "PLANNED"},
		{Ref: "TD-15", Offset: 1014, Actor: pClaude, Kind: evStatus, Status: "DOING"},
		{Ref: "TD-15", Offset: 1020, Actor: pClaude, Kind: evStatus, Status: "DONE"},

		// TD-16: service mesh health (Whiskers)
		{Ref: "TD-16", Offset: 1022, Actor: pWhiskers, Kind: evStatus, Status: "PLANNED"},
		{Ref: "TD-16", Offset: 1024, Actor: pWhiskers, Kind: evStatus, Status: "DOING"},
		{Ref: "TD-16", Offset: 1032, Actor: pWhiskers, Kind: evComment, Text: "Service mesh health monitoring dashboard live! Shows dependency graph with latency on each edge, error propagation highlighting, and alerts when any service-to-service latency exceeds thresholds. Already helped identify a slow Redis connection that was affecting menu load times."},
		{Ref: "TD-16", Offset: 1033, Actor: pWhiskers, Kind: evStatus, Status: "DONE"},

		// TD-17: performance profiling (Sarah)
		{Ref: "TD-17", Offset: 1032, Actor: pSarah, Kind: evStatus, Status: "PLANNED"},
		{Ref: "TD-17", Offset: 1036, Actor: pSarah, Kind: evStatus, Status: "DOING"},
		{Ref: "TD-17", Offset: 1050, Actor: pSarah, Kind: evComment, Text: "Performance profiling done. Found and fixed:\n- Goroutine leak in WebSocket handler (connections not being cleaned up on timeout) -- leaked ~50 goroutines/hour\n- Unnecessary JSON marshal/unmarshal cycle in the caching layer -- 15% CPU reduction\n- Menu listing endpoint was loading full item descriptions for list view -- trimmed to summaries, 40% memory reduction"},
		{Ref: "TD-17", Offset: 1052, Actor: pSarah, Kind: evStatus, Status: "DONE"},

		// TD-18: chaos engineering (agent)
		{Ref: "TD-18", Offset: 1042, Actor: pDevin, Kind: evStatus, Status: "PLANNED"},
		{Ref: "TD-18", Offset: 1044, Actor: pDevin, Kind: evStatus, Status: "DOING"},
		{Ref: "TD-18", Offset: 1054, Actor: pDevin, Kind: evComment, Text: "Chaos test suite running. Tested: random service kill (auto-recovers in 30s), network partition (circuit breakers activate correctly), Stripe API 500 (payments queue for retry), database failover (RDS Multi-AZ switches in ~60s with 2 dropped requests). All degradation modes are graceful."},
		{Ref: "TD-18", Offset: 1055, Actor: pDevin, Kind: evStatus, Status: "DONE"},

		// E7 -> DONE
		{Ref: "E7", Offset: 1055, Actor: pSarah, Kind: evComment, Text: "Tech debt sprint complete. All 18 items shipped. The system is significantly more robust than before the incident. Load testing confirms we can handle 500 concurrent users comfortably."},
		{Ref: "E7", Offset: 1055, Actor: pSarah, Kind: evStatus, Status: "DONE"},
	}
}

// ──────────────────────────────────────────────────────────────────────
// Phase 6: Launch Crunch (offset 1056-1368h, W23 ~70pts, W24 ~90pts)
//
// prev_7d window (1104-1272): target ~70 pts of DONE transitions
// last_7d window (1272-1440): target ~95 pts of DONE transitions
// ──────────────────────────────────────────────────────────────────────

func phase6Launch() []seedIssue {
	return []seedIssue{
		// W23 fillers (offset 1056-1176)
		{Ref: "LN-X1", Title: "Pre-warm CDN caches for all restaurant assets", Desc: "Script to crawl all restaurant and menu photo URLs, priming CloudFront edge caches before launch.", Labels: []string{"improvement", "infra"}, Estimate: 3, Actor: pDevin, Offset: 1040},
		{Ref: "LN-X2", Title: "Implement graceful degradation for Stripe outages", Desc: "When Stripe is down: queue payments for retry, show 'payment processing' to users. Resume automatically when Stripe recovers.", Labels: []string{"improvement", "backend"}, Estimate: 3, Actor: pClaude, Offset: 1048},

		// Pre-launch bridge (offset 1056-1104, early W23)
		{Ref: "LN-1", Title: "Verify all environment variables in production", Desc: "Audit that every env var used in staging exists in production with correct values. Validate Stripe keys, API secrets, database URLs.", Labels: []string{"improvement", "infra"}, Estimate: 2, Actor: pDevin, Offset: 1056},
		{Ref: "LN-2", Title: "Write deployment runbook for launch day", Desc: "Step-by-step deployment guide: order of service deployments, smoke test checkpoints, rollback triggers, and communication plan.", Labels: []string{"improvement"}, Estimate: 3, Actor: pSarah, Offset: 1060},
		{Ref: "LN-3", Title: "Pre-launch security: rotate all API keys", Desc: "Rotate Stripe keys, SendGrid keys, Google Maps key, and internal service tokens. Update in AWS Secrets Manager. Verify all services reconnect.", Labels: []string{"improvement", "infra"}, Estimate: 2, Actor: pDevin, Offset: 1068},

		// Main launch work (offset 1104-1368)
		{Ref: "LN-4", Title: "Production environment setup", Desc: "Mirror staging to production with production-grade settings. Multi-AZ RDS, autoscaling ECS, CloudFront distribution.", Labels: []string{"feature", "infra"}, Estimate: 5, Actor: pJake, Offset: 1104},
		{Ref: "LN-5", Title: "DNS and SSL certificate setup", Desc: "Configure upurreats.com DNS. SSL certificates via ACM. Redirect www to apex. Set up api.upurreats.com subdomain.", Labels: []string{"feature", "infra"}, Estimate: 2, Actor: pDevin, Offset: 1108},
		{Ref: "LN-6", Title: "Implement CDN caching strategy", Desc: "CloudFront caching for static assets (1 year), menu photos (1 day), API responses (no cache). Cache invalidation on menu updates.", Labels: []string{"improvement", "infra"}, Estimate: 3, Actor: pDevin, Offset: 1112},
		{Ref: "LN-7", Title: "Security audit: OWASP top 10 review", Desc: "Review all endpoints against OWASP top 10. Focus on: injection, broken auth, sensitive data exposure, CSRF, and rate limiting.", Labels: []string{"improvement", "backend"}, Estimate: 5, Actor: pSarah, Offset: 1116},
		{Ref: "LN-8", Title: "Mobile app store submission (iOS)", Desc: "Prepare and submit iOS app to App Store. Screenshots, description, privacy policy, app review guidelines compliance.", Labels: []string{"feature", "mobile"}, Estimate: 2, Actor: pJake, Offset: 1120},
		{Ref: "LN-9", Title: "Mobile app store submission (Android)", Desc: "Prepare and submit Android app to Google Play. Store listing, screenshots, content rating questionnaire.", Labels: []string{"feature", "mobile"}, Estimate: 2, Actor: pJake, Offset: 1120},
		{Ref: "LN-10", Title: "Landing page for upurreats.com", Desc: "Marketing landing page with hero section, feature highlights, app store links, and email waitlist signup.", Labels: []string{"feature", "frontend", "design"}, Estimate: 5, Actor: pLuna, Offset: 1124},
		{Ref: "LN-11", Title: "Set up on-call rotation and runbooks", Desc: "PagerDuty on-call rotation for launch week. Runbooks for: service down, high error rate, payment issues, database issues.", Labels: []string{"improvement", "infra"}, Estimate: 3, Actor: pJake, Offset: 1128},
		{Ref: "LN-12", Title: "Seed production database with launch restaurants", Desc: "Onboard 15 partner restaurants for launch. Import menus, photos, operating hours. Verify all data is correct.", Labels: []string{"feature", "backend"}, Estimate: 3, Actor: pSarah, Offset: 1132},
		{Ref: "LN-13", Title: "Final QA pass on critical user flows", Desc: "End-to-end testing of: registration, cat profile creation, browsing, ordering, payment, delivery tracking, and refund. Both web and mobile.", Labels: []string{"improvement"}, Estimate: 5, Actor: pWhiskers, Offset: 1140},
		{Ref: "LN-14", Title: "Performance budget and Lighthouse audit", Desc: "Run Lighthouse on all customer-facing pages. Target: Performance > 90, Accessibility > 95. Fix any critical issues.", Labels: []string{"improvement", "frontend"}, Estimate: 3, Actor: pLuna, Offset: 1148},
		{Ref: "LN-15", Title: "Configure production logging and alerting", Desc: "Production-specific log levels, alert thresholds, and escalation policies. Separate from staging. 90-day retention.", Labels: []string{"improvement", "infra"}, Estimate: 2, Actor: pDevin, Offset: 1156},
		{Ref: "LN-16", Title: "Write launch day checklist and rollback plan", Desc: "Step-by-step checklist for launch day. Include go/no-go criteria, feature flag activation order, and rollback procedures for each component.", Labels: []string{"improvement"}, Estimate: 3, Actor: pSarah, Offset: 1160},
		{Ref: "LN-17", Title: "Enable Upurr Premium subscription in production", Desc: "Configure Stripe production keys for subscription billing. Verify webhook endpoints. Test with a real card in production.", Labels: []string{"feature", "backend"}, Estimate: 2, Actor: pJake, Offset: 1168},
		{Ref: "LN-18", Title: "Accessibility audit and fixes", Desc: "Screen reader testing, keyboard navigation, color contrast, ARIA labels. Fix issues found in QA.", Labels: []string{"improvement", "frontend"}, Estimate: 3, Actor: pLuna, Offset: 1176},
		{Ref: "LN-19", Title: "Implement real-time order count widget for launch", Desc: "Live counter on the landing page showing total orders placed since launch. WebSocket-powered, updates in real-time.", Labels: []string{"feature", "frontend"}, Estimate: 3, Actor: pClaude, Offset: 1136},
		{Ref: "LN-20", Title: "Build restaurant onboarding verification checklist", Desc: "Automated checklist for verifying new restaurants: menu completeness, photo quality, operating hours, delivery zone, payment setup.", Labels: []string{"feature", "backend"}, Estimate: 3, Actor: pClaude, Offset: 1144},

		// Bugs found during QA
		{Ref: "BUG-5", Title: "Order confirmation email has wrong delivery address", Desc: "The confirmation email shows the restaurant address instead of the delivery address. Template variable mixup.", Labels: []string{"bug", "backend"}, Estimate: 1, Actor: pClaude, Offset: 1152},
		{Ref: "BUG-6", Title: "Cat avatar upload fails for HEIC images from iPhone", Desc: "iPhone photos in HEIC format aren't handled by the image processing pipeline. Need to add HEIC to JPEG conversion.", Labels: []string{"bug", "backend"}, Estimate: 2, Actor: pDevin, Offset: 1160},
		{Ref: "BUG-7", Title: "Promo code field accepts expired codes without error", Desc: "Expired promo codes are silently accepted at checkout but the discount isn't applied. Should show clear error message.", Labels: []string{"bug", "frontend"}, Estimate: 2, Actor: pLuna, Offset: 1172},
		{Ref: "BUG-8", Title: "Driver app crashes when GPS is disabled", Desc: "The driver app crashes with an unhandled exception when location services are turned off. Should show a prompt to enable GPS.", Labels: []string{"bug", "mobile"}, Estimate: 1, Actor: pWhiskers, Offset: 1180},
		{Ref: "BUG-9", Title: "Search results don't respect restaurant operating hours", Desc: "Closed restaurants appear in search results with no indication they're closed. Should be dimmed or filtered.", Labels: []string{"bug", "backend"}, Estimate: 2, Actor: pClaude, Offset: 1192},
		{Ref: "BUG-10", Title: "Keyboard navigation breaks in checkout flow", Desc: "Tab order skips the tip selector and jumps to the Stripe card input. Focus trap issue in the checkout modal.", Labels: []string{"bug", "frontend"}, Estimate: 1, Actor: pLuna, Offset: 1200},

		// Launch polish (offset 1200-1368, feeds last_7d window)
		{Ref: "LN-21", Title: "Configure auto-scaling policies for launch traffic", Desc: "ECS auto-scaling: scale up at 60% CPU, scale down at 30%. Min 2 tasks, max 12. CloudWatch alarms for scaling events.", Labels: []string{"improvement", "infra"}, Estimate: 3, Actor: pDevin, Offset: 1208},
		{Ref: "LN-22", Title: "Implement feature flag gradual rollout plan", Desc: "Define rollout order and percentage ramps for all feature flags. Start at 5% for each flag, ramp to 100% over 3 days.", Labels: []string{"improvement", "backend"}, Estimate: 2, Actor: pClaude, Offset: 1212},
		{Ref: "LN-23", Title: "Build internal metrics dashboard for launch day", Desc: "Grafana dashboard with: orders/min, payment success rate, p99 latency, error rate, active users. Auto-refresh every 10s.", Labels: []string{"feature", "infra"}, Estimate: 3, Actor: pDevin, Offset: 1216},
		{Ref: "LN-24", Title: "Pre-warm CDN caches for all restaurant menus", Desc: "Script to crawl all restaurant and menu endpoints, priming CloudFront edge caches. Run 1 hour before launch.", Labels: []string{"improvement", "infra"}, Estimate: 2, Actor: pClaude, Offset: 1224},
		{Ref: "LN-25", Title: "Write customer-facing launch announcement email", Desc: "Email to waitlist subscribers announcing launch. Include welcome promo code, app store links, and top restaurant highlights.", Labels: []string{"feature"}, Estimate: 2, Actor: pSarah, Offset: 1228},
		{Ref: "LN-26", Title: "Implement graceful degradation for external services", Desc: "When Stripe/Maps/TaxJar are down: show cached data, queue payments for retry, use flat-rate tax estimate. No hard failures.", Labels: []string{"improvement", "backend"}, Estimate: 3, Actor: pSarah, Offset: 1236},
		{Ref: "LN-27", Title: "Implement order throttling for overloaded restaurants", Desc: "When a restaurant has 10+ pending orders, pause accepting new orders and show estimated wait time. Auto-resume when queue drops.", Labels: []string{"feature", "backend"}, Estimate: 3, Actor: pClaude, Offset: 1244},
		{Ref: "LN-28", Title: "Final driver app QA pass", Desc: "Test driver app on 5 device types: order acceptance, navigation, delivery confirmation, photo upload, earnings view.", Labels: []string{"improvement", "mobile"}, Estimate: 3, Actor: pWhiskers, Offset: 1252},
		{Ref: "LN-29", Title: "Database migration dry-run on production", Desc: "Run all pending migrations against a production DB snapshot. Verify data integrity, measure execution time, check for locks.", Labels: []string{"improvement", "backend"}, Estimate: 2, Actor: pDevin, Offset: 1260},
		{Ref: "LN-30", Title: "Set up status page at status.upurreats.com", Desc: "Public status page showing system health. Components: API, Web App, Mobile, Payments, Delivery Tracking. Automated incident detection.", Labels: []string{"feature", "infra"}, Estimate: 3, Actor: pDevin, Offset: 1268},
		{Ref: "LN-31", Title: "Configure WAF rules for bot protection", Desc: "AWS WAF rules: rate limiting, known bot signatures, SQL injection patterns, geographic restrictions (US-only for MVP).", Labels: []string{"improvement", "infra"}, Estimate: 2, Actor: pClaude, Offset: 1276},
		{Ref: "LN-32", Title: "Final end-to-end payment test in production", Desc: "Place real test orders through the full production stack. Verify: charge, receipt email, refund, subscription create/cancel.", Labels: []string{"improvement", "backend"}, Estimate: 3, Actor: pJake, Offset: 1280},
		{Ref: "LN-33", Title: "Add real-time driver availability map for ops", Desc: "Internal map showing all drivers, their status (available/busy/offline), current deliveries, and zones with no coverage.", Labels: []string{"feature", "frontend"}, Estimate: 3, Actor: pWhiskers, Offset: 1284},
		{Ref: "LN-34", Title: "Implement per-restaurant request rate monitoring", Desc: "Track request volume per restaurant to detect if a popular restaurant is getting hammered. Auto-scale their menu cache.", Labels: []string{"improvement", "backend"}, Estimate: 3, Actor: pClaude, Offset: 1292},
		{Ref: "LN-35", Title: "Implement automated order anomaly detection", Desc: "Flag orders that look unusual: very high value, rapid repeat orders from same user, orders to invalid addresses. Queue for manual review.", Labels: []string{"feature", "backend"}, Estimate: 3, Actor: pSarah, Offset: 1300},
		{Ref: "LN-36", Title: "Build launch day communication templates", Desc: "Pre-written templates for: launch announcement, incident communication, status updates, and post-launch thank you. Ready in Slack and email.", Labels: []string{"improvement"}, Estimate: 2, Actor: pSarah, Offset: 1304},
		{Ref: "LN-37", Title: "Implement CDN failover for static assets", Desc: "S3 origin failover if CloudFront returns 5xx. Secondary origin in different region. Tested with simulated CloudFront outage.", Labels: []string{"improvement", "infra"}, Estimate: 3, Actor: pDevin, Offset: 1308},
		{Ref: "LN-38", Title: "Add database read replicas for reporting queries", Desc: "RDS read replica for analytics and reporting queries. Route admin dashboard and analytics endpoints to replica. Reduces load on primary.", Labels: []string{"improvement", "infra"}, Estimate: 3, Actor: pDevin, Offset: 1316},
		{Ref: "LN-39", Title: "Implement customer referral tracking", Desc: "Track referral codes through the signup flow. Credit both referrer and referee. Dashboard showing referral chain and conversion rates.", Labels: []string{"feature", "backend"}, Estimate: 3, Actor: pClaude, Offset: 1324},
		{Ref: "LN-40", Title: "Build automated daily backup verification", Desc: "Daily job that restores latest backup to a test instance, runs integrity checks, and reports results. Slack alert on failure.", Labels: []string{"improvement", "infra"}, Estimate: 3, Actor: pJake, Offset: 1332},
		{Ref: "LN-41", Title: "Add request logging and audit trail for admin actions", Desc: "Log all admin actions (refunds, credits, account changes) with timestamp, actor, and before/after state. Queryable via admin UI.", Labels: []string{"improvement", "backend"}, Estimate: 3, Actor: pClaude, Offset: 1340},
		{Ref: "LN-42", Title: "Final load test at 2x expected launch traffic", Desc: "Run load test at 1000 concurrent users (2x target). Verify auto-scaling, database performance, and payment flow under heavy load.", Labels: []string{"improvement", "infra"}, Estimate: 3, Actor: pJake, Offset: 1350},
		{Ref: "LN-43", Title: "Implement real-time revenue tracking dashboard", Desc: "Live dashboard showing total revenue, orders per hour, average order value, and payment method breakdown. For the launch war room.", Labels: []string{"feature", "frontend"}, Estimate: 3, Actor: pWhiskers, Offset: 1356},
	}
}

func phase6Events() []seedEvent {
	return []seedEvent{
		// W23 fillers
		{Ref: "LN-X1", Offset: 1042, Actor: pDevin, Kind: evStatus, Status: "DOING"},
		{Ref: "LN-X1", Offset: 1048, Actor: pDevin, Kind: evStatus, Status: "DONE"},
		{Ref: "LN-X2", Offset: 1050, Actor: pClaude, Kind: evStatus, Status: "DOING"},
		{Ref: "LN-X2", Offset: 1056, Actor: pClaude, Kind: evStatus, Status: "DONE"},

		// Pre-launch bridge (offset 1056-1104)
		{Ref: "LN-1", Offset: 1058, Actor: pDevin, Kind: evStatus, Status: "DOING"},
		{Ref: "LN-1", Offset: 1066, Actor: pDevin, Kind: evComment, Text: "Env var audit complete. Found 2 missing vars in production: TAXJAR_API_KEY and SENDGRID_WEBHOOK_SECRET. Added via Secrets Manager. All 34 env vars verified across 4 services."},
		{Ref: "LN-1", Offset: 1067, Actor: pDevin, Kind: evStatus, Status: "DONE"},

		{Ref: "LN-2", Offset: 1062, Actor: pSarah, Kind: evStatus, Status: "DOING"},
		{Ref: "LN-2", Offset: 1084, Actor: pSarah, Kind: evComment, Text: "Deployment runbook documented. 4 phases: database migration, backend services, frontend, feature flag activation. Each phase has a smoke test gate before proceeding. Rollback = revert Docker tag + feature flag kill switch."},
		{Ref: "LN-2", Offset: 1086, Actor: pSarah, Kind: evStatus, Status: "DONE"},

		{Ref: "LN-3", Offset: 1070, Actor: pDevin, Kind: evStatus, Status: "DOING"},
		{Ref: "LN-3", Offset: 1078, Actor: pDevin, Kind: evComment, Text: "All API keys rotated. Old keys will be revoked 48h after launch as a safety net. Verified all services reconnect with new credentials. No downtime during rotation."},
		{Ref: "LN-3", Offset: 1079, Actor: pDevin, Kind: evStatus, Status: "DONE"},

		// Main launch work (prev_7d window starts at 1104)
		{Ref: "LN-4", Offset: 1106, Actor: pJake, Kind: evStatus, Status: "DOING"},
		{Ref: "LN-4", Offset: 1126, Actor: pJake, Kind: evComment, Text: "Prod infra is up. Multi-AZ RDS, ECS with autoscaling (2-8 tasks), CloudFront, WAF. All Terraform. Separate AWS account from staging."},
		{Ref: "LN-4", Offset: 1128, Actor: pJake, Kind: evStatus, Status: "DONE"},

		{Ref: "LN-5", Offset: 1110, Actor: pDevin, Kind: evStatus, Status: "DOING"},
		{Ref: "LN-5", Offset: 1116, Actor: pDevin, Kind: evStatus, Status: "DONE"},

		{Ref: "LN-6", Offset: 1114, Actor: pDevin, Kind: evStatus, Status: "DOING"},
		{Ref: "LN-6", Offset: 1122, Actor: pDevin, Kind: evStatus, Status: "DONE"},

		{Ref: "LN-7", Offset: 1118, Actor: pSarah, Kind: evStatus, Status: "DOING"},
		{Ref: "LN-7", Offset: 1286, Actor: pSarah, Kind: evComment, Text: "OWASP audit findings:\n- SQL injection: parameterized queries everywhere (pass)\n- XSS: React escaping + CSP headers (pass)\n- CSRF: missing on 2 endpoints (fixed)\n- Broken auth: JWT + refresh tokens solid (pass)\n- Rate limiting: added stricter limits on auth endpoints\n- Sensitive data: PII encrypted at rest (pass)\n\nAll issues fixed. No critical findings."},
		{Ref: "LN-7", Offset: 1148, Actor: pSarah, Kind: evStatus, Status: "DONE"},

		{Ref: "LN-8", Offset: 1122, Actor: pJake, Kind: evStatus, Status: "DOING"},
		{Ref: "LN-8", Offset: 1280, Actor: pJake, Kind: evComment, Text: "iOS app submitted. Review typically takes 24-48h. Fingers crossed for first-time approval."},
		{Ref: "LN-8", Offset: 1142, Actor: pJake, Kind: evStatus, Status: "DONE"},

		{Ref: "LN-9", Offset: 1122, Actor: pJake, Kind: evStatus, Status: "DOING"},
		{Ref: "LN-9", Offset: 1134, Actor: pJake, Kind: evStatus, Status: "DONE"},

		{Ref: "LN-10", Offset: 1126, Actor: pLuna, Kind: evStatus, Status: "DOING"},
		{Ref: "LN-10", Offset: 1290, Actor: pLuna, Kind: evComment, Text: "Landing page is live at upurreats.com. Hero with a cute cat eating from a fancy bowl, feature carousel, app store badges, and email signup. Already got 200+ signups from the beta waitlist email."},
		{Ref: "LN-10", Offset: 1292, Actor: pLuna, Kind: evStatus, Status: "DONE"},

		{Ref: "LN-11", Offset: 1130, Actor: pJake, Kind: evStatus, Status: "DOING"},
		{Ref: "LN-11", Offset: 1288, Actor: pJake, Kind: evComment, Text: "On-call rotation set up in PagerDuty. Launch week schedule:\n- Mon/Tue: Jake (primary), Sarah (secondary)\n- Wed/Thu: Sarah (primary), Jake (secondary)\n- Fri/Sat/Sun: Jake (primary), Claude (secondary)\n\nRunbooks documented for all critical scenarios. Slack channel #launch-war-room created."},
		{Ref: "LN-11", Offset: 1290, Actor: pJake, Kind: evStatus, Status: "DONE"},

		{Ref: "LN-19", Offset: 1138, Actor: pClaude, Kind: evStatus, Status: "DOING"},
		{Ref: "LN-19", Offset: 1148, Actor: pClaude, Kind: evStatus, Status: "DONE"},

		{Ref: "LN-12", Offset: 1134, Actor: pSarah, Kind: evStatus, Status: "DOING"},
		{Ref: "LN-12", Offset: 1298, Actor: pSarah, Kind: evComment, Text: "15 launch restaurants onboarded and verified. Mix of: 5 premium cat food kitchens, 4 pet-friendly delis, 3 raw food specialists, 2 organic cat cafes, 1 cat bakery. All menus imported with photos. Test orders placed and confirmed with each restaurant."},
		{Ref: "LN-12", Offset: 1300, Actor: pSarah, Kind: evStatus, Status: "DONE"},

		{Ref: "LN-13", Offset: 1282, Actor: pWhiskers, Kind: evStatus, Status: "DOING"},
		{Ref: "LN-13", Offset: 1302, Actor: pWhiskers, Kind: evComment, Text: "Final QA complete! Tested all critical flows on Chrome, Safari, Firefox (desktop), iPhone 14, iPhone SE (iOS Safari), Pixel 7, Samsung S23 (Android Chrome). Found 6 bugs (filed separately). All P1/P2 -- no blockers."},
		{Ref: "LN-13", Offset: 1303, Actor: pWhiskers, Kind: evStatus, Status: "DONE"},

		{Ref: "LN-20", Offset: 1286, Actor: pClaude, Kind: evStatus, Status: "DOING"},
		{Ref: "LN-20", Offset: 1298, Actor: pClaude, Kind: evStatus, Status: "DONE"},

		{Ref: "LN-14", Offset: 1290, Actor: pLuna, Kind: evStatus, Status: "DOING"},
		{Ref: "LN-14", Offset: 1312, Actor: pLuna, Kind: evComment, Text: "Lighthouse scores:\n- Performance: 94\n- Accessibility: 97\n- Best Practices: 100\n- SEO: 98\n\nFixed: lazy loading for below-fold images, font-display: swap, and one missing alt text."},
		{Ref: "LN-14", Offset: 1314, Actor: pLuna, Kind: evStatus, Status: "DONE"},

		// BUG-5: email address bug (agent, fast)
		{Ref: "BUG-5", Offset: 1294, Actor: pClaude, Kind: evStatus, Status: "DOING"},
		{Ref: "BUG-5", Offset: 1296, Actor: pClaude, Kind: evComment, Text: "Template variable was `{{.Restaurant.Address}}` instead of `{{.Delivery.Address}}`. Fixed and added a test that verifies template rendering against a known order."},
		{Ref: "BUG-5", Offset: 1297, Actor: pClaude, Kind: evStatus, Status: "DONE"},

		{Ref: "LN-15", Offset: 1298, Actor: pDevin, Kind: evStatus, Status: "DOING"},
		{Ref: "LN-15", Offset: 1306, Actor: pDevin, Kind: evStatus, Status: "DONE"},

		{Ref: "LN-16", Offset: 1302, Actor: pSarah, Kind: evStatus, Status: "DOING"},
		{Ref: "LN-16", Offset: 1320, Actor: pSarah, Kind: evComment, Text: "Launch checklist and rollback plan documented. Go/no-go criteria:\n- All P0/P1 bugs fixed\n- Load test passing at 500 users\n- Payment flow verified in production\n- All monitoring/alerting active\n- App store approvals received\n- Restaurants confirmed and ready\n\nRollback: feature flags kill switch for each major feature. Full rollback = DNS failover to maintenance page."},
		{Ref: "LN-16", Offset: 1322, Actor: pSarah, Kind: evStatus, Status: "DONE"},

		// BUG-6: HEIC images (agent)
		{Ref: "BUG-6", Offset: 1304, Actor: pDevin, Kind: evStatus, Status: "DOING"},
		{Ref: "BUG-6", Offset: 1310, Actor: pDevin, Kind: evComment, Text: "Added HEIC support using libheif. Convert to JPEG on upload before processing. Also added AVIF support while at it."},
		{Ref: "BUG-6", Offset: 1311, Actor: pDevin, Kind: evStatus, Status: "DONE"},

		{Ref: "LN-17", Offset: 1310, Actor: pJake, Kind: evStatus, Status: "DOING"},
		{Ref: "LN-17", Offset: 1322, Actor: pJake, Kind: evComment, Text: "Premium subscription live in prod. Tested with my own card. $9.99 charged, trial started, webhook received, all good. Flipped the LaunchDarkly flag for internal testers."},
		{Ref: "LN-17", Offset: 1324, Actor: pJake, Kind: evStatus, Status: "DONE"},

		// BUG-7: promo codes
		{Ref: "BUG-7", Offset: 1316, Actor: pLuna, Kind: evStatus, Status: "DOING"},
		{Ref: "BUG-7", Offset: 1322, Actor: pLuna, Kind: evComment, Text: "Promo code validation now checks expiry date before applying. Shows clear error: \"This promo code expired on [date]\". Also added visual feedback -- the input turns red."},
		{Ref: "BUG-7", Offset: 1324, Actor: pLuna, Kind: evStatus, Status: "DONE"},

		{Ref: "LN-18", Offset: 1318, Actor: pLuna, Kind: evStatus, Status: "DOING"},
		{Ref: "LN-18", Offset: 1270, Actor: pLuna, Kind: evComment, Text: "Accessibility audit done. Fixed: 12 missing ARIA labels, 3 color contrast issues, keyboard trap in the checkout modal, and screen reader ordering for the menu filter chips. WCAG 2.1 AA compliant."},
		{Ref: "LN-18", Offset: 1273, Actor: pLuna, Kind: evStatus, Status: "DONE"},

		// BUG-8: driver GPS crash (Whiskers, fast)
		{Ref: "BUG-8", Offset: 1324, Actor: pWhiskers, Kind: evStatus, Status: "DOING"},
		{Ref: "BUG-8", Offset: 1330, Actor: pWhiskers, Kind: evComment, Text: "Added GPS check on app launch. If disabled, shows a modal explaining why location is needed and a button to open Settings. App no longer crashes -- handled gracefully!"},
		{Ref: "BUG-8", Offset: 1331, Actor: pWhiskers, Kind: evStatus, Status: "DONE"},

		// BUG-9: search hours (agent)
		{Ref: "BUG-9", Offset: 1334, Actor: pClaude, Kind: evStatus, Status: "DOING"},
		{Ref: "BUG-9", Offset: 1342, Actor: pClaude, Kind: evComment, Text: "Search now filters by operating hours. Closed restaurants appear at the bottom, greyed out, with \"Opens at [time]\" label. Open restaurants get a green dot indicator."},
		{Ref: "BUG-9", Offset: 1343, Actor: pClaude, Kind: evStatus, Status: "DONE"},

		// BUG-10: keyboard nav (Luna)
		{Ref: "BUG-10", Offset: 1344, Actor: pLuna, Kind: evStatus, Status: "DOING"},
		{Ref: "BUG-10", Offset: 1350, Actor: pLuna, Kind: evComment, Text: "Fixed tab order in checkout. The tip selector was missing tabindex. Also fixed the focus trap -- Escape now closes the modal properly."},
		{Ref: "BUG-10", Offset: 1352, Actor: pLuna, Kind: evStatus, Status: "DONE"},

		// Launch polish (offset 1200+, feeds last_7d window starting at 1272)
		{Ref: "LN-21", Offset: 1350, Actor: pDevin, Kind: evStatus, Status: "DOING"},
		{Ref: "LN-21", Offset: 1360, Actor: pDevin, Kind: evComment, Text: "Auto-scaling configured. Scale-up threshold: 60% avg CPU over 3 min. Scale-down: 30% over 10 min. Tested by simulating load spike -- scales from 2 to 6 tasks in 4 minutes."},
		{Ref: "LN-21", Offset: 1361, Actor: pDevin, Kind: evStatus, Status: "DONE"},

		{Ref: "LN-22", Offset: 1354, Actor: pClaude, Kind: evStatus, Status: "DOING"},
		{Ref: "LN-22", Offset: 1362, Actor: pClaude, Kind: evStatus, Status: "DONE"},

		{Ref: "LN-23", Offset: 1358, Actor: pDevin, Kind: evStatus, Status: "DOING"},
		{Ref: "LN-23", Offset: 1372, Actor: pDevin, Kind: evComment, Text: "Launch day Grafana dashboard ready. 6 panels: orders/minute (real-time), payment success rate, API p99 latency, error rate by service, active WebSocket connections, and ECS task count. Shared link with the team."},
		{Ref: "LN-23", Offset: 1373, Actor: pDevin, Kind: evStatus, Status: "DONE"},

		{Ref: "LN-24", Offset: 1366, Actor: pClaude, Kind: evStatus, Status: "DOING"},
		{Ref: "LN-24", Offset: 1374, Actor: pClaude, Kind: evStatus, Status: "DONE"},

		{Ref: "LN-25", Offset: 1370, Actor: pSarah, Kind: evStatus, Status: "DOING"},
		{Ref: "LN-25", Offset: 1268, Actor: pSarah, Kind: evComment, Text: "Launch email sent to 847 waitlist subscribers. Includes 20% off first order promo code FIRSTPURR. Open rate: 62% in the first hour -- great engagement."},
		{Ref: "LN-25", Offset: 1272, Actor: pSarah, Kind: evStatus, Status: "DONE"},

		{Ref: "LN-26", Offset: 1378, Actor: pSarah, Kind: evStatus, Status: "DOING"},
		{Ref: "LN-26", Offset: 1272, Actor: pSarah, Kind: evComment, Text: "Graceful degradation implemented. When external services fail:\n- Stripe down: queue payment, show \"payment processing\" to user, charge when recovered\n- Google Maps down: use cached ETAs, hide live tracking map\n- TaxJar down: use cached rate for zip code, flag for reconciliation\n\nEach degraded state is clearly logged and alerts fire."},
		{Ref: "LN-26", Offset: 1274, Actor: pSarah, Kind: evStatus, Status: "DONE"},

		{Ref: "LN-27", Offset: 1246, Actor: pClaude, Kind: evStatus, Status: "DOING"},
		{Ref: "LN-27", Offset: 1270, Actor: pClaude, Kind: evComment, Text: "Order throttling working. When restaurant queue hits 10, new orders see \"This restaurant is busy -- estimated wait: X min\". Customers can either wait or choose another restaurant. Auto-resumes at 5 pending orders."},
		{Ref: "LN-27", Offset: 1272, Actor: pClaude, Kind: evStatus, Status: "DONE"},

		{Ref: "LN-28", Offset: 1254, Actor: pWhiskers, Kind: evStatus, Status: "DOING"},
		{Ref: "LN-28", Offset: 1272, Actor: pWhiskers, Kind: evComment, Text: "Driver app QA done on: iPhone 14, iPhone SE, Pixel 7, Samsung S23, Samsung A14. All flows working. One minor UI issue on SE (small screen cuts off earnings chart) -- filed as low priority."},
		{Ref: "LN-28", Offset: 1274, Actor: pWhiskers, Kind: evStatus, Status: "DONE"},

		{Ref: "LN-29", Offset: 1262, Actor: pDevin, Kind: evStatus, Status: "DOING"},
		{Ref: "LN-29", Offset: 1274, Actor: pDevin, Kind: evComment, Text: "Migration dry-run complete. 3 pending migrations executed in 12 seconds on a production-size snapshot. No locking issues detected. Data integrity check passed -- row counts match, no orphaned foreign keys."},
		{Ref: "LN-29", Offset: 1276, Actor: pDevin, Kind: evStatus, Status: "DONE"},

		{Ref: "LN-30", Offset: 1270, Actor: pDevin, Kind: evStatus, Status: "DOING"},
		{Ref: "LN-30", Offset: 1280, Actor: pDevin, Kind: evComment, Text: "Status page live at status.upurreats.com. Components monitored: API (health check), Web App (synthetic), Payments (Stripe webhook latency), Delivery Tracking (WebSocket health), Mobile API (health check). Auto-incident on 3 consecutive failures."},
		{Ref: "LN-30", Offset: 1282, Actor: pDevin, Kind: evStatus, Status: "DONE"},

		// last_7d window (1272+)
		{Ref: "LN-31", Offset: 1278, Actor: pClaude, Kind: evStatus, Status: "DOING"},
		{Ref: "LN-31", Offset: 1286, Actor: pClaude, Kind: evStatus, Status: "DONE"},

		{Ref: "LN-32", Offset: 1282, Actor: pJake, Kind: evStatus, Status: "DOING"},
		{Ref: "LN-32", Offset: 1298, Actor: pJake, Kind: evComment, Text: "Full payment cycle tested in prod. Placed 3 orders with real cards (refunded immediately). Subscription create/cancel verified. Webhook delivery confirmed. All green. We're ready."},
		{Ref: "LN-32", Offset: 1300, Actor: pJake, Kind: evStatus, Status: "DONE"},

		{Ref: "LN-33", Offset: 1286, Actor: pWhiskers, Kind: evStatus, Status: "DOING"},
		{Ref: "LN-33", Offset: 1298, Actor: pWhiskers, Kind: evComment, Text: "Real-time driver availability map is live in the ops dashboard! Shows all drivers with color-coded status, current delivery routes, and coverage heat map. The dark spots in the northeast tell us we need to recruit more drivers there."},
		{Ref: "LN-33", Offset: 1299, Actor: pWhiskers, Kind: evStatus, Status: "DONE"},

		{Ref: "LN-34", Offset: 1294, Actor: pClaude, Kind: evStatus, Status: "DOING"},
		{Ref: "LN-34", Offset: 1304, Actor: pClaude, Kind: evComment, Text: "Per-restaurant request monitoring live. Dashboard shows requests/min per restaurant. If any restaurant exceeds 100 req/min, their menu cache TTL bumps from 15min to 1 hour automatically. Reverts when traffic drops."},
		{Ref: "LN-34", Offset: 1305, Actor: pClaude, Kind: evStatus, Status: "DONE"},

		{Ref: "LN-35", Offset: 1302, Actor: pSarah, Kind: evStatus, Status: "DOING"},
		{Ref: "LN-35", Offset: 1318, Actor: pSarah, Kind: evComment, Text: "Anomaly detection running. Rules: order > $200 (review), 3+ orders same user in 10 min (rate limit), delivery address > 50km from restaurant (block). Flagged orders go to a review queue in the admin dashboard."},
		{Ref: "LN-35", Offset: 1320, Actor: pSarah, Kind: evStatus, Status: "DONE"},

		{Ref: "LN-36", Offset: 1306, Actor: pSarah, Kind: evStatus, Status: "DOING"},
		{Ref: "LN-36", Offset: 1316, Actor: pSarah, Kind: evStatus, Status: "DONE"},

		{Ref: "LN-37", Offset: 1310, Actor: pDevin, Kind: evStatus, Status: "DOING"},
		{Ref: "LN-37", Offset: 1322, Actor: pDevin, Kind: evStatus, Status: "DONE"},

		{Ref: "LN-38", Offset: 1318, Actor: pDevin, Kind: evStatus, Status: "DOING"},
		{Ref: "LN-38", Offset: 1330, Actor: pDevin, Kind: evComment, Text: "Read replica live. Analytics and reporting queries routed to replica via connection string. Primary DB CPU dropped from 45% to 28% during load test. Replication lag < 100ms."},
		{Ref: "LN-38", Offset: 1331, Actor: pDevin, Kind: evStatus, Status: "DONE"},

		{Ref: "LN-39", Offset: 1326, Actor: pClaude, Kind: evStatus, Status: "DOING"},
		{Ref: "LN-39", Offset: 1338, Actor: pClaude, Kind: evComment, Text: "Referral tracking live. Each user gets a unique referral code. $10 credit for both parties on first order. Dashboard shows referral funnel: shared, signed up, ordered. Already 23 referrals from the beta waitlist."},
		{Ref: "LN-39", Offset: 1339, Actor: pClaude, Kind: evStatus, Status: "DONE"},

		{Ref: "LN-40", Offset: 1334, Actor: pJake, Kind: evStatus, Status: "DOING"},
		{Ref: "LN-40", Offset: 1350, Actor: pJake, Kind: evStatus, Status: "DONE"},

		{Ref: "LN-41", Offset: 1342, Actor: pClaude, Kind: evStatus, Status: "DOING"},
		{Ref: "LN-41", Offset: 1354, Actor: pClaude, Kind: evComment, Text: "Admin audit trail complete. Every admin action logged with: timestamp, actor email, action type, affected resource, and full before/after diff. Searchable in admin UI with date range filter. 90-day retention."},
		{Ref: "LN-41", Offset: 1355, Actor: pClaude, Kind: evStatus, Status: "DONE"},

		{Ref: "LN-42", Offset: 1352, Actor: pJake, Kind: evStatus, Status: "DOING"},
		{Ref: "LN-42", Offset: 1366, Actor: pJake, Kind: evComment, Text: "2x load test passed. 1000 concurrent users for 15 minutes. Auto-scaling kicked in as expected (2 -> 8 tasks). p99 latency stayed under 300ms for all endpoints except payments (2.8s, Stripe-bound). Zero errors. Database handled it fine with the read replica."},
		{Ref: "LN-42", Offset: 1368, Actor: pJake, Kind: evStatus, Status: "DONE"},

		{Ref: "LN-43", Offset: 1358, Actor: pWhiskers, Kind: evStatus, Status: "DOING"},
		{Ref: "LN-43", Offset: 1368, Actor: pWhiskers, Kind: evComment, Text: "Revenue tracking dashboard ready for launch day! Shows live revenue counter, orders per hour chart, average order value gauge, payment method pie chart, and top restaurants by revenue. Big number display for total revenue -- perfect for the war room TV."},
		{Ref: "LN-43", Offset: 1369, Actor: pWhiskers, Kind: evStatus, Status: "DONE"},
	}
}

// ──────────────────────────────────────────────────────────────────────
// Phase 7: Post-Launch (offset 1368-1440h, W25 ~30pts partial)
// ──────────────────────────────────────────────────────────────────────

func phase7PostLaunch() []seedIssue {
	return []seedIssue{
		{Ref: "PL-1", Title: "Post-launch: slow menu loading in downtown area", Desc: "Users in the downtown area report slow menu loading (5-8 seconds). Likely CDN cache miss for newly onboarded restaurants.", Labels: []string{"bug", "backend"}, Estimate: 2, Actor: pDevin, Offset: 1370},
		{Ref: "PL-2", Title: "Post-launch: push notifications not arriving on Android 14", Desc: "Some Android 14 users not receiving push notifications. Possibly related to new notification permission model.", Labels: []string{"bug", "mobile"}, Estimate: 2, Actor: pWhiskers, Offset: 1374},
		{Ref: "PL-3", Title: "Post-launch: restaurant dashboard shows wrong timezone", Desc: "Restaurant owners on the west coast see order times in UTC instead of local time. Timezone handling bug in the dashboard.", Labels: []string{"bug", "frontend"}, Estimate: 1, Actor: pLuna, Offset: 1380},
		{Ref: "PL-4", Title: "Add order again button to delivery confirmation", Desc: "Feature request from multiple users: add a prominent \"order again\" button on the delivery confirmation screen.", Labels: []string{"improvement", "frontend"}, Estimate: 2, Actor: pClaude, Offset: 1384},
		{Ref: "PL-5", Title: "Investigate high Stripe processing fees", Desc: "Stripe fees are 3.2% instead of expected 2.9%. May need to negotiate volume pricing or optimize payment method mix.", Labels: []string{"improvement", "backend"}, Estimate: 2, Actor: pJake, Offset: 1388},
		{Ref: "PL-6", Title: "Add restaurant response time SLA tracking", Desc: "Track how quickly restaurants confirm orders. Surface slow restaurants in the admin dashboard. Set up alerts for SLA breaches.", Labels: []string{"feature", "backend"}, Estimate: 3, Actor: pClaude, Offset: 1392},
		{Ref: "PL-7", Title: "Post-launch: delivery photo uploads timing out", Desc: "Drivers in areas with poor cellular coverage can't upload delivery photos. Need offline queue with retry.", Labels: []string{"bug", "mobile"}, Estimate: 2, Actor: pDevin, Offset: 1396},
		{Ref: "PL-8", Title: "Customer satisfaction survey integration", Desc: "After delivery, prompt customer to rate their experience (food quality, delivery speed, app experience). NPS tracking.", Labels: []string{"feature", "frontend"}, Estimate: 5, Actor: pLuna, Offset: 1400},
		{Ref: "PL-9", Title: "Add weekly metrics email for restaurant owners", Desc: "Automated weekly email to restaurant owners with: orders received, revenue, average rating, top items, and comparison to previous week.", Labels: []string{"feature", "backend"}, Estimate: 3, Actor: pClaude, Offset: 1404},
		{Ref: "PL-10", Title: "Implement A/B testing framework for recommendations", Desc: "A/B test different recommendation algorithms. Track click-through rate and conversion. Use LaunchDarkly for traffic splitting.", Labels: []string{"feature", "backend"}, Estimate: 5, Actor: pSarah, Offset: 1410},
		{Ref: "PL-11", Title: "Post-launch: order history pagination broken", Desc: "Users with 20+ orders see an infinite loading spinner. The pagination cursor is off by one.", Labels: []string{"bug", "backend"}, Estimate: 1, Actor: pDevin, Offset: 1414},
		{Ref: "PL-12", Title: "Add real-time order volume dashboard for ops", Desc: "Internal dashboard showing live order volume, active drivers, and restaurant load. Used by ops team to monitor launch.", Labels: []string{"feature", "frontend", "infra"}, Estimate: 3, Actor: pWhiskers, Offset: 1418},
		{Ref: "PL-13", Title: "Optimize menu photo CDN costs", Desc: "Menu photos are our biggest S3/CloudFront cost. Implement WebP conversion, aggressive caching, and thumbnail sizes.", Labels: []string{"improvement", "infra"}, Estimate: 2, Actor: pDevin, Offset: 1424},
		{Ref: "PL-14", Title: "Set up customer support ticketing integration", Desc: "Integrate Zendesk for customer support. Auto-create tickets from in-app help button. Include order context in ticket.", Labels: []string{"feature", "backend"}, Estimate: 5, Actor: pJake, Offset: 1428},
		{Ref: "PL-15", Title: "Post-launch: cat birthday confetti crashes Safari on iPad", Desc: "The confetti animation on cat birthdays causes Safari on older iPads to freeze. Too many DOM elements in the particle system.", Labels: []string{"bug", "frontend"}, Estimate: 1, Actor: pWhiskers, Offset: 1432},
	}
}

func phase7Events() []seedEvent {
	return []seedEvent{
		// Launch celebration comment
		{Ref: "LN-4", Offset: 1370, Actor: pSarah, Kind: evComment, Text: "**UPURR EATS IS LIVE!**\n\nAll systems green. First real order came in at 10:04 AM -- a tuna platter for a cat named Sir Whiskertons III. We're in the war room monitoring everything.\n\nGreat work everyone. Whatever happens next, we built something special."},

		// PL-1: slow menu loading (agent)
		{Ref: "PL-1", Offset: 1372, Actor: pDevin, Kind: evStatus, Status: "DOING"},
		{Ref: "PL-1", Offset: 1378, Actor: pDevin, Kind: evComment, Text: "Found the issue. New restaurant menus weren't pre-warmed in CloudFront. Added a cache warming script that runs after any menu update. Also increased the cache TTL for menu data from 1 hour to 6 hours. Load times down to 800ms."},
		{Ref: "PL-1", Offset: 1379, Actor: pDevin, Kind: evStatus, Status: "DONE"},

		// PL-2: Android notifications (Whiskers)
		{Ref: "PL-2", Offset: 1376, Actor: pWhiskers, Kind: evStatus, Status: "DOING"},
		{Ref: "PL-2", Offset: 1384, Actor: pWhiskers, Kind: evComment, Text: "Android 14 requires POST_NOTIFICATIONS permission to be requested at runtime. Our app was only declaring it in the manifest but not requesting it. Added the runtime permission request on first launch. Also added a deep link to notification settings if the user denies. Classic Android fragmentation fun."},
		{Ref: "PL-2", Offset: 1385, Actor: pWhiskers, Kind: evStatus, Status: "DONE"},

		// PL-3: timezone bug (Luna)
		{Ref: "PL-3", Offset: 1382, Actor: pLuna, Kind: evStatus, Status: "DOING"},
		{Ref: "PL-3", Offset: 1388, Actor: pLuna, Kind: evComment, Text: "The restaurant dashboard was using `new Date()` without timezone conversion. Switched to `Intl.DateTimeFormat` with the restaurant's configured timezone. All times now display correctly."},
		{Ref: "PL-3", Offset: 1390, Actor: pLuna, Kind: evStatus, Status: "DONE"},

		// PL-4: order again (agent)
		{Ref: "PL-4", Offset: 1386, Actor: pClaude, Kind: evStatus, Status: "PLANNED"},
		{Ref: "PL-4", Offset: 1388, Actor: pClaude, Kind: evStatus, Status: "DOING"},
		{Ref: "PL-4", Offset: 1396, Actor: pClaude, Kind: evComment, Text: "Added \"Order Again\" button to delivery confirmation and order history. Pre-fills cart with the same items, checks availability. If an item is unavailable, shows a note and suggests alternatives."},
		{Ref: "PL-4", Offset: 1397, Actor: pClaude, Kind: evStatus, Status: "DONE"},

		// PL-5: Stripe fees (Jake)
		{Ref: "PL-5", Offset: 1390, Actor: pJake, Kind: evStatus, Status: "DOING"},
		{Ref: "PL-5", Offset: 1404, Actor: pJake, Kind: evComment, Text: "Investigated the Stripe fees. The 3.2% rate is because most transactions are international cards (lots of cross-border cat food orders, apparently). Standard for international. Reached out to Stripe about volume pricing -- they'll review once we hit $50k/month."},
		{Ref: "PL-5", Offset: 1406, Actor: pJake, Kind: evStatus, Status: "DONE"},

		// PL-6: SLA tracking (agent)
		{Ref: "PL-6", Offset: 1394, Actor: pClaude, Kind: evStatus, Status: "PLANNED"},
		{Ref: "PL-6", Offset: 1398, Actor: pClaude, Kind: evStatus, Status: "DOING"},
		{Ref: "PL-6", Offset: 1408, Actor: pClaude, Kind: evComment, Text: "Restaurant response time SLA tracking implemented. Measures time from order received to restaurant confirmation. Dashboard shows per-restaurant average, p95, and SLA breach count. Alerts fire if a restaurant's p95 exceeds 10 minutes."},
		{Ref: "PL-6", Offset: 1409, Actor: pClaude, Kind: evStatus, Status: "DONE"},

		// PL-7: photo upload timeout (agent)
		{Ref: "PL-7", Offset: 1398, Actor: pDevin, Kind: evStatus, Status: "DOING"},
		{Ref: "PL-7", Offset: 1408, Actor: pDevin, Kind: evComment, Text: "Added offline queue for delivery photo uploads. Photos are saved locally and uploaded when connectivity improves. Queue persists across app restarts. Shows upload status indicator. Max queue size: 50 photos."},
		{Ref: "PL-7", Offset: 1409, Actor: pDevin, Kind: evStatus, Status: "DONE"},

		// PL-8: customer survey (Luna, PLANNED only)
		{Ref: "PL-8", Offset: 1402, Actor: pLuna, Kind: evStatus, Status: "PLANNED"},

		// PL-9: weekly metrics email (agent)
		{Ref: "PL-9", Offset: 1406, Actor: pClaude, Kind: evStatus, Status: "PLANNED"},
		{Ref: "PL-9", Offset: 1410, Actor: pClaude, Kind: evStatus, Status: "DOING"},

		// PL-10: A/B testing (Sarah, PLANNED only)
		{Ref: "PL-10", Offset: 1412, Actor: pSarah, Kind: evStatus, Status: "PLANNED"},

		// PL-11: pagination bug (agent, quick fix)
		{Ref: "PL-11", Offset: 1416, Actor: pDevin, Kind: evStatus, Status: "DOING"},
		{Ref: "PL-11", Offset: 1420, Actor: pDevin, Kind: evComment, Text: "Classic off-by-one. The cursor was using `>=` instead of `>` on the `created_at` column, causing the last item on each page to repeat as the first item of the next page. After 20+ orders, the repeated items caused the frontend to enter an infinite render loop. Fixed with `>` and added a test."},
		{Ref: "PL-11", Offset: 1421, Actor: pDevin, Kind: evStatus, Status: "DONE"},

		// PL-12: ops dashboard (Whiskers)
		{Ref: "PL-12", Offset: 1420, Actor: pWhiskers, Kind: evStatus, Status: "DOING"},
		{Ref: "PL-12", Offset: 1432, Actor: pWhiskers, Kind: evComment, Text: "Real-time ops dashboard is live! Shows live order volume (1-min, 5-min, 1-hour windows), active drivers on map, restaurant queue depths, payment success rate, and system health indicators. Using Grafana with InfluxDB for time-series data. Refreshes every 10 seconds."},
		{Ref: "PL-12", Offset: 1433, Actor: pWhiskers, Kind: evStatus, Status: "DONE"},

		// PL-13: CDN optimization (agent, DOING)
		{Ref: "PL-13", Offset: 1426, Actor: pDevin, Kind: evStatus, Status: "DOING"},

		// PL-14: Zendesk (Jake, PLANNED)
		{Ref: "PL-14", Offset: 1430, Actor: pJake, Kind: evStatus, Status: "PLANNED"},

		// PL-15: confetti bug (Whiskers)
		{Ref: "PL-15", Offset: 1434, Actor: pWhiskers, Kind: evStatus, Status: "DOING"},
		{Ref: "PL-15", Offset: 1438, Actor: pWhiskers, Kind: evComment, Text: "The confetti animation was spawning 500 DOM elements. Reduced to 100 particles with CSS transforms instead of absolute positioning. Also added a `prefers-reduced-motion` check to disable it entirely for users who prefer no animations. Safari is happy now -- my birthday feature lives on!"},
		{Ref: "PL-15", Offset: 1439, Actor: pWhiskers, Kind: evStatus, Status: "DONE"},

		// Weekly launch metrics comment
		{Ref: "LN-12", Offset: 1438, Actor: pSarah, Kind: evComment, Text: "Launch week metrics (first 5 days):\n\n- **Total orders:** 1,247\n- **Unique customers:** 483\n- **Repeat order rate:** 34%\n- **Average order value:** $28.50\n- **Active drivers:** 42\n- **Restaurant average rating:** 4.6/5\n- **App crashes:** 3 (all fixed)\n- **Payment success rate:** 99.7%\n- **Average delivery time:** 38 minutes\n\nTop ordered item: \"Wild Salmon Feast\" from The Purring Chef. The cats of this city have good taste."},
	}
}
