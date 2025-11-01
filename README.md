# 🔐 Secret & Token Leak Detection System

A Go-based system that scans public repositories for leaked tokens and secrets from a predefined **inventory**, calculates a **confidence score**, enriches results with **geolocation metadata**, and sends alerts via **email (SMTP)**.

---

## 🚀 Features

### 🧾 Token Inventory
- Maintains a local inventory (`inventory.json`) of tokens for multiple providers:  
  - AWS, Azure, GitHub, GCP, etc.
- Each entry stores:
  - Token Type  
  - Token Value  
  - Owner Email (used for alerting)

### 🔍 Leakage Detection
- Scans repositories using the **GitHub API** for possible leaks.
- Detects matches between public file content and known tokens in your inventory.
- For each match, collects:
  - **Token Type**
  - **Source URL/Path**
  - **Code Snippet (Context)**
  - **Confidence Score** – based on how closely a leaked string matches a known token.


### 📣 Alerting Mechanism
- Sends concise alerts through:
  - **Email (SMTP)**
- Email content includes:
  - Token Type  
  - Source Link / File Path  
  - Geolocation Info  
  - Suggested Remediation  
- All email credentials and configuration are securely loaded from `.env`.

### 🪵 Logging
- Logs every event such as:
  - Detected leaks  
  - Emails sent  
  - Errors and API failures  
- Helps audit detection and notification flows.

---

## 📁 Repository Structure

├── route/
│ ├── func.go # Handles scanning route logic and request processing
│ └── router.go # Defines HTTP routes (POST /repo)
├── scanner/
│ └── scan.go # Core logic for scanning repositories for leaked secrets
├── setup/
│ └── setup.go # Handles configuration setup and GitHub API client initialization
├── utils/
│ └── email.go # Contains email sending logic using SMTP
├── .env # Environment variables (SMTP, GitHub token, etc.)
├── .gitignore
├── Dockerfile # Docker setup for containerized execution
├── go.mod
├── go.sum
├── inventory.json # Token inventory containing keys and their owner info
└── main.go # Entry point that initializes the server on port 8080


## 🧩 API Endpoints

### 🔹 1. Health Check
Check if the server is running properly.

**Endpoint:**
GET /health

css
Copy code

**Response:**
```json
{
  "status": "ok"
}

🔹 2. Scan Repository for Leaks

This endpoint scans a GitHub repository (or a local file source) for any leaked tokens defined in inventory.json.

Endpoint
POST /repo

Headers
Key	Value
Content-Type	application/json
Request Body
{
  "repo_url": "https://github.com/BadadheVed/Collabify"
}

Response – No Leak Found
{
  "errors": null,
  "repo": "https://github.com/BadadheVed/Collabify",
  "results": null,
  "status": "scan complete"
}

Response – Leak Detected
{
  "errors": null,
  "repo": "https://github.com/BadadheVed/LeakyRepo",
  "results": [
    {
      "provider": "github",
      "token_type": "github_personal_access_token",
      "matched_value": "ghp_exampletokenvalue",
      "file_path": "src/config/dev.env",
      "confidence": 0.95,
      "owner": "dev1@example.com"
    }
  ],
  "status": "leak detected"
}

📧 When a Leak is Detected

An email is automatically sent to the token owner (as listed in inventory.json).

A log entry appears in the server console:

[INFO] Sent email to dev1@example.com for leaked key github_personal_access_token.

Example cURL Command
curl -X POST http://localhost:8080/repo \
     -H "Content-Type: application/json" \
     -d '{"repo_url": "https://github.com/BadadheVed/LeakyRepo"}'
