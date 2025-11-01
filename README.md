# Secret & Token Leak Detection System

A robust Go-based system designed to scan public GitHub repositories for leaked tokens and secrets from a local inventory, calculate a confidence score, enrich findings with geolocation metadata, and send immediate alerts via email (SMTP).

## Table of contents

- Overview
- Features
  - Token Inventory
  - Leakage Detection
  - Alerting Mechanism
  - Logging
- Repository structure
- Configuration
  - `.env` (required)
  - `inventory.json` (required)
- API
  - Health Check: `GET /health`
  - Scan Repository: `POST /repo`
- Example requests & responses
- Alert email contents & remediation guidance
- Running in Docker
- Contributing
- License & contact

---

## Overview

This service scans public GitHub repositories for exact matches to known high-value tokens stored in a local inventory. When a match is found, the system:

1. Records contextual information (file path, snippet).
2. Computes a confidence score.
3. Enriches the event with geolocation metadata where available (from the request/commit/author IP or other available metadata).
4. Immediately notifies the token owner by email with remediation steps and logs all actions for auditability.

---

## Features

### Token Inventory (`inventory.json`)
- Local JSON file with known internal/high-value tokens only.
- Supported provider examples: AWS, Azure, GitHub, GCP, and custom token types.
- Each inventory entry includes:
  - token_type (string)
  - token_value (string) — the literal token to match
  - owner_email (string) — where alerts should be sent

Example inventory entry:
```json
{
  "provider": "github",
  "token_type": "github_personal_access_token",
  "token_value": "ghp_exampletokenvalue",
  "owner_email": "dev1@example.com"
}
```

Notes:
- Inventory is intended to reduce false positives by only flagging known internal tokens.
- Keep `inventory.json` secure and rotate values when secrets are rotated.

### Leakage Detection
- Uses the GitHub API to fetch file contents from public repos (no private repo access).
- Exact-match detection against token values in `inventory.json`.
- For each match the system records:
  - Token type
  - Matched value (masked in logs/alerts when necessary)
  - Source repo and file path
  - Code snippet (context)
  - Confidence score (0.0–1.0)
  - Owner email

Confidence scoring (example heuristic):
- 1.0: Exact token string found in code file
- 0.9: Exact token with typical formatting/noise (e.g., surrounded by quotes)
- Lower scores when partial matches or additional noise detected

### Alerting Mechanism (Email / SMTP)
- Sends immediate high-priority emails to the token owner in the inventory.
- SMTP configuration and credentials loaded securely from `.env`.
- Alerts contain:
  - Token type
  - Source link (file path / GitHub file URL)
  - Context snippet
  - Geolocation metadata (when available)
  - Suggested remediation steps
- Email sending logic in `utils/email.go`

Security:
- Never store SMTP credentials in source code.
- Ensure `.env` and `inventory.json` are excluded from public repos and backups when necessary.

### Logging
- Audit-level logs for:
  - Detected leaks (timestamp, token type, repo, file)
  - Emails sent (recipient, status)
  - Errors and API failures
- Purpose: auditing, incident response, and postmortem analysis

---

## Repository structure

SGASSIGNMENT/
- route/
  - func.go — scanning route logic and request processing
  - router.go — HTTP routes (POST /repo)
- scanner/
  - scan.go — core scanning logic that queries the GitHub API and matches tokens
- setup/
  - setup.go — configuration loading and GitHub client initialization
- utils/
  - email.go — SMTP email send logic
- .env — environment variables (SMTP, GitHub token, etc.)
- .gitignore
- Dockerfile — containerized setup
- go.mod
- go.sum
- inventory.json — local token inventory
- main.go — server entrypoint, runs on port 8080

---

## Configuration

Required environment variables (example `.env` keys):
- GITHUB_TOKEN — GitHub API token (scoped to public repo read; keep minimal permissions)
- SMTP_HOST — SMTP server host
- SMTP_PORT — SMTP server port
- SMTP_USER — SMTP auth username
- SMTP_PASS — SMTP auth password
- SMTP_FROM — "from" email address for alerts
- LOG_LEVEL — logging verbosity (optional)

Place these keys in a `.env` file in the project root (do not commit `.env`).

Example `.env` (DO NOT commit real secrets):
```
GITHUB_TOKEN=ghp_xxx
SMTP_HOST=smtp.example.com
SMTP_PORT=587
SMTP_USER=alert@example.com
SMTP_PASS=supersecret
SMTP_FROM=alerts@example.com
LOG_LEVEL=info
```

inventory.json example (partial):
```json
[
  {
    "provider": "github",
    "token_type": "github_personal_access_token",
    "token_value": "ghp_exampletokenvalue",
    "owner_email": "dev1@example.com"
  },
  {
    "provider": "aws",
    "token_type": "aws_access_key_id",
    "token_value": "AKIAEXAMPLEKEY",
    "owner_email": "aws-owner@example.com"
  }
]
```

---

## API

Server runs by default on port 8080.

### 1) Health check
- Endpoint: GET /health
- Response:
```json
{
  "status": "ok"
}
```

### 2) Scan repository for leaks
- Endpoint: POST /repo
- Headers:
  - Content-Type: application/json
- Request body:
```json
{
  "repo_url": "https://github.com/BadadheVed/LeakyRepo"
}
```

Behavior:
- The server scans the given public repository for tokens present in `inventory.json`.
- If no leaks are found, response indicates scan complete.
- If leaks are found, the API returns details and the system sends emails to token owners.

Response — No leak found:
```json
{
  "errors": null,
  "repo": "https://github.com/BadadheVed/Collabify",
  "results": null,
  "status": "scan complete"
}
```

Response — Leak detected (example):
```json
{
  "errors": null,
  "repo": "https://github.com/BadadheVed/LeakyRepo",
  "results": [
    {
            "inventory_id": "github-1",
            "provider": "github",
            "token_type": "github_personal_access_token",
            "matched_value": "ghp_exampletokenvalue",
            "repo_url": "https://github.com/BadadheVed/Scalable-Reminder-System",
            "file_path": "backend/src/index.ts",
            "blob_url": "https://github.com/BadadheVed/Scalable-Reminder-System/blob/main/backend/src/index.ts",
            "snippet": "const sec2 = \"ghp_exampletokenvalue\";",
            "timestamp": "2025-11-01T05:55:03.74687Z"
    }
  ],
  "status": "leak detected"
}
```

Logging example when an alert is sent:
```
[INFO] Sent email to dev1@example.com for leaked key github_personal_access_token.
```

Example cURL command:
```bash
curl -X POST http://localhost:8080/repo \
     -H "Content-Type: application/json" \
     -d '{"repo_url": "https://github.com/BadadheVed/LeakyRepo"}'
```

---

## Alert email contents & suggested remediation

Email includes:
- Summary: provider & token type
- Location: repository + file path + link to the file on GitHub
- Context: code snippet showing the matched token (masked if necessary)

---

## Running in Docker

Build:
```bash
docker build -t leak-detector:latest .
```

Run (example):
```bash
docker run -e GITHUB_TOKEN=<YOUR_GITHUB_PAT> \
           -e SMTP_PORT=587 \
           -e SMTP_USER=<YOUR_EMAIL> \
           -e SMTP_PASSWORD=<GMAIL_APP_PASSWORD> \
           -e INVENTORY_PATH=inventory.json \
           -e SMTP_HOST=smtp.gmail.com \
           -e WORKER_COUNT=<YOUR DESIRED WORKER COUNT> \
           
           leak-detector
```

Ensure `inventory.json` is present inside the container image (or mount it via a volume) and that `.env` contains valid credentials.

---

## Notes & best practices

- Only maintain known internal tokens in `inventory.json`. Do not store third-party or customer tokens.
- Rotate inventory tokens on regular cadence and after any suspected leak.
- Restrict the GitHub token (GITHUB_TOKEN) to the minimum required scopes (public repo read).
- Use secure channels for backups of inventory and .env files.
- Consider rate limits of the GitHub API; implement backoff & retries in `scanner/scan.go`.
- Consider adding multi-factor alerting (Slack, PagerDuty) for critical tokens.


- License: Add your preferred license here (e.g., MIT).
- For questions or issues, open an issue in the repository or contact the repository owner.
