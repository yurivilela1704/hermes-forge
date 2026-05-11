# release-notes-gen — Project Plan

> AI-powered release notes generator. Listens to GitLab/GitHub webhooks on merged MRs/PRs, sends context to an LLM, and saves a formatted `.md` file per release.

---

## Goals

- **Primary**: Generate consolidated daily release notes from merged MRs/PRs
- **Secondary**: Store raw events for daily aggregation and support manual generation via UI
- **Portfolio**: Open source, clean architecture, easy to run locally with Docker

---

## Stack

| Layer       | Choice                              |
|-------------|-------------------------------------|
| Language    | Go                                  |
| Server      | Go stdlib `net/http`                |
| UI          | Plain HTML + JS (served by Go)      |
| AI          | Pluggable: OpenAI / Azure OpenAI / Anthropic |
| Storage     | Local filesystem (Docker volume)    |
| Infra       | Docker + docker-compose             |

---

## Project Structure

```
release-notes-gen/
├── cmd/
│   └── server/
│       └── main.go               # entrypoint, loads config, starts HTTP server
├── internal/
│   ├── webhook/
│   │   ├── gitlab.go             # handles GitLab MR merged events
│   │   ├── github.go             # handles GitHub PR merged events
│   │   └── webhook.go            # shared signature validation
│   ├── llm/
│   │   ├── provider.go           # Provider interface
│   │   ├── openai.go             # OpenAI implementation
│   │   ├── azure.go              # Azure OpenAI implementation
│   │   └── anthropic.go          # Anthropic implementation
│   ├── generator/
│   │   ├── generator.go          # orchestrates: input → prompt → LLM → .md
│   │   └── prompt.go             # builds the prompt from template + MR data
│   └── storage/
│       ├── storage.go            # saves .md files, maintains index.json
│       └── index.go              # list/read saved notes for UI
├── web/
│   ├── index.html                # UI: list notes + manual input form
│   └── static/                   # css, js
├── templates/
│   └── default.tmpl              # release note prompt template (configurable)
├── output/                       # generated .md notes
│   ├── events/                   # raw MR/PR events persisted for aggregation
│   ├── index.json                # metadata index for UI
│   └── traces/                   # step-by-step logs of interactions
│       ├── webhooks/             # raw webhook requests/responses
│       └── generations/          # raw LLM prompts/responses/notes
```

---

## Trigger

A release note can be generated:
1. **Automatically (Daily)**: Triggered via the `/api/generate-daily` endpoint (manually or via a daily cron job).
2. **Manually**: Through the Web UI by providing MR data.

**Webhook handling**:
When a MR (GitLab) or PR (GitHub) is merged, the raw event is saved to `output/events/` for later daily consolidation. No release note is generated immediately upon webhook reception.

---

## Output Format

One `.md` file per merged MR/PR saved to `/output/`.

**File naming:**
```
YYYY-MM-DD_<branch-or-mr-title-slugified>.md
```
Example: `2026-04-16_add-user-authentication.md`

**File content template:**
```markdown
## [type] MR Title
**Date:** YYYY-MM-DD | **Author:** @username | **MR/PR:** #42
**Repository:** org/repo | **Branch:** feature/branch-name

### Summary
<LLM generated — 2 to 4 sentences, clear and readable for devs and managers>

### Changes
- <bullet list of changes, LLM generated from commits + MR description>

### Impact / Breaking Changes
<None | description of breaking changes>

### Notes
<optional: anything else the LLM flags as worth mentioning>
```

---

## LLM Prompt Template (`templates/default.tmpl`)

The prompt is built from the MR data and injected into the template:

```
You are a technical writer generating release notes for a software team.

Given the following merged MR/PR data, generate a release note following the structure below.
Be clear, concise, and readable for both developers and managers.
Do not invent information. If data is missing or vague, say so honestly.

--- MR DATA ---
Title: {{.Title}}
Author: {{.Author}}
Branch: {{.Branch}}
Description: {{.Description}}
Commits:
{{range .Commits}}- {{.Message}} ({{.Author}})
{{end}}
---------------

Output ONLY the release note content in this structure:
### Summary
### Changes
### Impact / Breaking Changes
### Notes
```

---

## Configuration (`.env`)

```env
# LLM Provider: openai | azure | anthropic
LLM_PROVIDER=openai

# OpenAI
OPENAI_API_KEY=sk-...
OPENAI_MODEL=gpt-4o

# Azure OpenAI (if LLM_PROVIDER=azure)
AZURE_OPENAI_ENDPOINT=https://your-resource.openai.azure.com
AZURE_OPENAI_API_KEY=...
AZURE_OPENAI_DEPLOYMENT=gpt-4o

# Anthropic (if LLM_PROVIDER=anthropic)
ANTHROPIC_API_KEY=sk-ant-...
ANTHROPIC_MODEL=claude-sonnet-4-20250514

# Webhooks
GITLAB_WEBHOOK_SECRET=your-secret
GITHUB_WEBHOOK_SECRET=your-secret

# Storage
OUTPUT_DIR=/output

# UI (optional basic protection)
UI_AUTH_TOKEN=         # if set, UI requires ?token=xxx or Bearer header

# Prompt template to use (filename inside /templates)
TEMPLATE=default
```

---

## API Endpoints

| Method | Path                  | Description                          |
|--------|-----------------------|--------------------------------------|
| POST   | `/webhook/gitlab`     | Receives GitLab MR events            |
| POST   | `/webhook/github`     | Receives GitHub PR events            |
| POST   | `/api/generate`       | Manual generation (from UI)          |
| POST   | `/api/generate-daily` | Generate consolidated daily report   |
| GET    | `/api/notes`          | List all generated notes (index)     |
| GET    | `/api/notes/:filename`| Get content of a specific note       |
| GET    | `/`                   | Serves the web UI                    |

---

## Docker Setup

**`Dockerfile`** — multi-stage, minimal final image:

```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY .. .
RUN go build -o release-notes-gen ./cmd/server

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/release-notes-gen .
COPY ../web ./web/
COPY ../templates ./templates/
EXPOSE 8080
CMD ["./release-notes-gen"]
```

**`docker-compose.yml`:**
```yaml
services:
  app:
    build: .
    ports:
      - "8080:8080"
    env_file:
      - .env
    volumes:
      - ./output:/output
```

Run locally:
```bash
cp .env.example .env
# fill in your keys
docker compose up
```

---

## Milestones

### v0.1 — Foundation
- [ ] Go project scaffolding (modules, folder structure)
- [ ] Config loading from `.env`
- [ ] HTTP server with health check endpoint

### v0.2 — Webhook Handlers
- [ ] GitLab MR merged webhook handler + signature validation
- [ ] GitHub PR merged webhook handler + signature validation
- [ ] Parse and normalize MR/PR data into a shared struct

### v0.3 — LLM Integration
- [ ] `Provider` interface
- [ ] OpenAI implementation
- [ ] Azure OpenAI implementation
- [ ] Anthropic implementation
- [ ] Prompt builder from template + MR data

### v0.4 — Storage
- [ ] Save `.md` files to OUTPUT_DIR
- [ ] Maintain `index.json` with metadata per note
- [ ] List and read notes via API

### v0.5 — Web UI
- [ ] Serve static UI from Go
- [ ] List existing release notes
- [ ] Manual input form (paste MR data → generate)
- [ ] Preview + download generated note

### v0.6 — Polish & Open Source Ready
- [ ] README with setup instructions and screenshots
- [ ] `.env.example` with all options documented
- [ ] Error handling for empty/vague MR data
- [ ] Optional UI auth token
- [ ] GitHub Actions CI (build + test)

---

## Open Questions (decide before or during build)

1. **Aggregation (v2?)** — combine multiple MR notes into one versioned release note (e.g. `v1.3.0`). Design storage with this in mind.
2. **Tone config** — should the prompt template support a `TONE` env var (`technical` vs `manager-friendly`)?
3. **What to do with vague MRs** — generate anyway with a warning comment in the `.md`, or skip and log?

---

## Notes for Claude Code

- Keep each package focused and small — this is not a large app
- Prefer stdlib over external dependencies where possible
- The `llm.Provider` interface is the most important abstraction — get it right first
- The prompt template in `/templates/default.tmpl` should be easy to edit without touching code
- All config comes from env vars — no hardcoded values anywhere
