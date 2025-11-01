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

