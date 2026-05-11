# Hermes Forge 

AI-powered release notes generator in Go.  
It receives merged GitLab/GitHub webhook events, generates release note text with an LLM provider, and stores `.md` files locally.

## Project Structure & Learning Go

If you are coming from Laravel/PHP, check out [Go for Laravel Developers](docs/go-for-laravel-devs.md) for a quick transition guide.

The project milestones and evolution can be found in the [docs/milestones](docs/milestones) directory.

## Current Milestones

- v0.1 Foundation: server + config loading
- v0.2 Webhooks: GitLab/GitHub merged event handlers + signature validation
- v0.3 LLM integration: pluggable providers + template prompt generation
- v0.4 Storage: save generated notes + `index.json` + read/list APIs
- v0.5 Daily Reports: consolidated release notes from multiple MRs

## Local Run

1. Copy environment template:

```bash
cp .env.example .env
```

2. Set at least:

- `LLM_PROVIDER` (`mock`, `openai`, `azure`, or `anthropic`)
- matching provider credentials
- webhook secrets:
  - `GITLAB_WEBHOOK_SECRET`
  - `GITHUB_WEBHOOK_SECRET`

3. Run server:

```bash
go run ./cmd/server
```

4. Health check:

```bash
curl -i http://localhost:8080/health
```

## Test Usage: GitLab Webhook

Send a merged merge request payload:

```bash
curl -i -X POST http://localhost:8080/webhook/gitlab \
  -H "Content-Type: application/json" \
  -H "X-Gitlab-Token: your-secret" \
  -d '{
    "object_kind": "merge_request",
    "project": {"path_with_namespace": "acme/hermes"},
    "user": {"username": "yuri"},
    "object_attributes": {
      "iid": 42,
      "title": "Add storage layer",
      "description": "Implements output persistence",
      "source_branch": "feature/storage",
      "state": "merged",
      "action": "merge",
      "url": "https://gitlab.example/acme/hermes/-/merge_requests/42",
      "merged_at": "2026-04-16T10:30:00Z",
      "last_commit": {"message": "feat: add storage"}
    }
  }'
```

Expected:

- HTTP `202 Accepted`
- Raw event saved under `output/events/` for later daily consolidation
- Full interaction trace saved under `output/traces/webhooks/`
- **Note:** Immediate generation on webhook is disabled; use Daily Reports or UI for generation.

## Test Usage: GitHub Webhook

GitHub requires `X-Hub-Signature-256` (HMAC SHA-256 over raw payload).

1. Define payload and secret:

```bash
PAYLOAD='{"action":"closed","repository":{"full_name":"acme/hermes"},"pull_request":{"number":7,"title":"Add storage","body":"Stores generated notes","html_url":"https://github.com/acme/hermes/pull/7","merged":true,"merged_at":"2026-04-16T10:30:00Z","head":{"ref":"feature/storage"},"user":{"login":"yuri"}}}'
SECRET='your-secret'
```

2. Compute signature:

```bash
SIG="sha256=$(printf '%s' "$PAYLOAD" | openssl dgst -sha256 -hmac "$SECRET" -hex | sed 's/^.* //')"
```

3. Send webhook:

```bash
curl -i -X POST http://localhost:8080/webhook/github \
  -H "Content-Type: application/json" \
  -H "X-Hub-Signature-256: $SIG" \
  -d "$PAYLOAD"
```

Expected:

- HTTP `202 Accepted`
- Raw event saved under `output/events/` for later daily consolidation
- Full interaction trace saved under `output/traces/webhooks/`
- **Note:** Immediate generation on webhook is disabled; use Daily Reports or UI for generation.

## GitLab/GitHub Sync API (v0.2+)

Instead of waiting for webhooks, you can actively pull merged MRs/PRs for a date range:

```bash
# Sync MRs for a GitLab project and date range
curl -i "http://localhost:8080/api/sync-gitlab?project=acme/hermes&start_date=2026-04-01&end_date=2026-04-17"

# Sync PRs for a GitHub repository and date range
curl -i "http://localhost:8080/api/sync-github?repo=acme/hermes&start_date=2026-04-01&end_date=2026-04-17"
```

Requires `GITLAB_API_URL`, `GITLAB_API_TOKEN` and/or `GITHUB_API_TOKEN` in `.env`.

## Notes API

- List notes:

```bash
curl -s http://localhost:8080/api/notes
```

- Read a note:

```bash
curl -s http://localhost:8080/api/notes/<filename>.md
```

- Generate today's report (aggregates all merged MRs received today):

```bash
curl -i -X POST http://localhost:8080/api/generate-daily
```

Optionally pass a specific date or use `dry_run=true` to see the prompt without calling AI:

```bash
curl -i -X POST "http://localhost:8080/api/generate-daily?date=2026-04-19&dry_run=true"
```

The same `dry_run=true` parameter can be used with `POST /api/generate`.

## Web UI

- Open `http://localhost:8080/`
- Use the form to manually generate and save notes
- Saved notes appear in the list and can be opened directly

### GitLab & GitHub Sync

- Use the "Sync Sources" section to fetch Merge Requests (GitLab) or Pull Requests (GitHub) directly.
- Enter the project path or repo (e.g., `namespace/project` or `owner/repo`) and a date range.
- Click "Sync and List" to view merged items and their commits.
- Use the "Send to AI" button on any item to generate a release note for it, or "Send ALL to AI" to consolidate everything.

The Web UI is now powered by Tailwind CSS for a modern, responsive experience.

If `UI_AUTH_TOKEN` is set, pass the same token in query string:

- `http://localhost:8080/?token=your-token`

## AI Integration Checklist

Use this when switching from placeholder values to a real provider setup.

For local no-cost testing, you can use:

- `LLM_PROVIDER=mock`

With `mock`, no external API key is required and generation/storage/UI flows can be tested end-to-end.

1. Pick one provider in `.env`:
   - `LLM_PROVIDER=openai` or `azure` or `anthropic`
2. Fill only the matching credentials:
   - OpenAI: `OPENAI_API_KEY`, optional `OPENAI_MODEL`
   - Azure: `AZURE_OPENAI_ENDPOINT`, `AZURE_OPENAI_API_KEY`, `AZURE_OPENAI_DEPLOYMENT`
   - Anthropic: `ANTHROPIC_API_KEY`, optional `ANTHROPIC_MODEL`
3. Keep `TEMPLATE=default` (or point to another file in `templates/`)
4. (Optional) Configure token cost telemetry:
   - `LLM_PRICE_INPUT_PER_MTOK_USD=0.20`
   - `LLM_PRICE_OUTPUT_PER_MTOK_USD=1.25`
5. Start server and check startup logs:
   - expected: `llm provider configured successfully`
6. Run one manual generation from UI or `POST /api/generate`
7. If generation fails (`502`), inspect server logs:
   - manual flow now logs provider/repository/title and exact upstream error
   - webhook flow logs generation failure details too
   - successful generations log token usage and estimated USD cost per request + running total

Tip: keep real keys only in `.env` (already gitignored) and never in `README.md`.

## Development Checks

```bash
go fmt ./...
go test ./...
go build ./cmd/server
```

## Docker

```bash
cp .env.example .env
docker compose up --build
```

The app is available at `http://localhost:8080`.
