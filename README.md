# 🛡️ ScamShield AI

> **Real-Time AI-Driven Detection of Phishing, Financial Fraud, OTP Scams & Fake News — Right Inside Your Browser.**

[![Go](https://img.shields.io/badge/Backend-Go_1.21+-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![React](https://img.shields.io/badge/Frontend-React_18-61DAFB?logo=react&logoColor=black)](https://react.dev/)
[![Vite](https://img.shields.io/badge/Bundler-Vite_5-646CFF?logo=vite&logoColor=white)](https://vitejs.dev/)
[![Tailwind CSS](https://img.shields.io/badge/UI-Tailwind_CSS_3-06B6D4?logo=tailwindcss&logoColor=white)](https://tailwindcss.com/)
[![Chrome MV3](https://img.shields.io/badge/Platform-Chrome_Extension_MV3-4285F4?logo=googlechrome&logoColor=white)](https://developer.chrome.com/docs/extensions/mv3/)
[![Gemini API](https://img.shields.io/badge/AI-Google_Gemini_3.6_Flash-8E75B2?logo=googlegemini&logoColor=white)](https://ai.google.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-10B981.svg)](LICENSE)

**ScamShield AI** is a production-ready cybersecurity and fact-checking **Chrome Browser Extension (Manifest V3)** powered by a zero-dependency **Go (Golang)** microservice and the **Google Gemini API (`gemini-3.6-flash` with `gemini-3.5-flash` fallback)**. It protects everyday internet users from SMS OTP scams, banking phishing links, lottery fraud, and social media disinformation — with full native output in both **English** and **සිංහල (Sinhala)**.

> 🏆 Built for **JIAT Hack Days Gampaha 2026** · Full Technical Report available in [`doc/index.html`](doc/index.html).

---

## 📑 Table of Contents

1. [Problem Statement & Solution](#-problem-statement--solution)
2. [Key Features](#-key-features)
3. [System Architecture & Data Flow](#-system-architecture--data-flow)
4. [Tech Stack](#-tech-stack)
5. [Project Directory Structure](#-project-directory-structure)
6. [Local Setup, Build & Run Guide](#-local-setup-build--run-guide)
7. [REST API Reference](#-rest-api-reference)
8. [Empirical Test Matrix & Real-World Scenarios](#-empirical-test-matrix--real-world-scenarios)
9. [Team Contributions & Ownership](#-team-contributions--ownership)
10. [AI Disclosure & Security Compliance](#-ai-disclosure--security-compliance)
11. [License & Disclaimer](#-license--disclaimer)

---

## 🎯 Problem Statement & Solution

### ❌ The Problem
Cyber fraudsters increasingly target citizens in Sri Lanka and globally via SMS OTP harvesting, fake bank account suspension alerts, deceptive WhatsApp/Telegram job offers, and viral fake news posts. Non-technical users often struggle to spot subtle indicators such as look-alike domains, shortened URLs, or psychological urgency tactics — and existing global tools rarely explain security threats in natural **Sinhala (සිංහල)**.

### 🛡️ The ScamShield AI Solution
**ScamShield AI** puts an intelligent cybersecurity analyst one right-click away inside Google Chrome:
- **Zero Friction:** Highlight any suspicious message on WhatsApp Web, Gmail, Facebook, or any webpage, right-click, and select **"Scan with ScamShield AI"**.
- **Instant Risk Triage:** Produces a deterministic **0–100 Risk Score**, a color-coded **Verdict** (`Safe`, `Suspicious`, `High Risk`), specific **Red Flags**, and concrete **Actionable Advice** in seconds.
- **Native Bilingual Intelligence:** Toggle seamlessly between **English** and **සිංහල (Sinhala)** so any user can understand *why* a message is dangerous and *what* to do next.

---

## ✨ Key Features

| Feature | Description |
| :--- | :--- |
| 🖱️ **Right-Click Context Menu Scan** | Select any suspicious text on any webpage → right-click → **"Scan with ScamShield AI"**. The extension popup launches and analyses the selection automatically. |
| 🔗 **One-Click Active URL Check** | Click **"Scan Current URL"** in the popup to evaluate the active browser tab's URL for homograph attacks, deceptive subdomains, or phishing redirects without copying. |
| 📊 **Granular Risk Score (0–100)** | Visual progress gauge with 3 threat tiers: 🟢 **Safe** (`0–34`) · 🟡 **Suspicious** (`35–69`) · 🔴 **High Risk** (`70–100`). |
| 🚩 **Structured Verdict & Red Flags** | Breaks down the exact manipulation cues detected (e.g., *artificial urgency, OTP/PIN solicitation, bank impersonation, unrealistic prize bait*). |
| 💡 **Actionable Defensive Advice** | Provides clear next steps tailored to the threat (e.g., *"Do not share your OTP. Call your bank via the official number on the back of your card."*). |
| 🇱🇰 **Bilingual Output (English / සිංහල)** | One-click toggle generates the summary, red flags, and safety advice in natural Sinhala or English. |
| 🔄 **Automatic AI Model Failover** | The Go backend automatically falls back from `gemini-3.6-flash` to `gemini-3.5-flash` on upstream 5xx errors or timeouts, ensuring reliability during live demos. |
| 🌑 **Cybersecurity Dark UI** | Responsive 360px popup interface crafted with React 18 and Tailwind CSS 3, including friendly offline backend detection. |

---

## 🏗️ System Architecture & Data Flow

```text
[User Text Selection / Active Tab URL]
                  │
                  ▼
  [Chrome MV3 Background Service Worker]
   (contextMenus / activeTab / storage)
                  │
                  ▼
     [React 18 + Tailwind Popup UI]
                  │
        POST http://localhost:8080/scan
        {"text": "...", "language": "English | Sinhala"}
                  │
                  ▼
       [Go (Golang) HTTP Backend]
   (Input validation, CORS, Prompt Builder)
                  │
        HTTPS POST (x-goog-api-key)
        Primary: gemini-3.6-flash
        Fallback: gemini-3.5-flash
                  │
                  ▼
         [Google Gemini REST API]
   (Temperature: 0.2, MIME: application/json)
                  │
                  ▼
       [Go Backend JSON Sanitizer]
   (Strips code fences & validates schema)
                  │
                  ▼
     [React Popup Renders Risk Report]
   (Score Gauge + Verdict + Red Flags + Advice)
```

---

## 🧱 Tech Stack

| Layer | Technology | Role & Justification |
| :--- | :--- | :--- |
| **Browser Platform** | **Chrome Extension (Manifest V3)** | Background service worker (`background.js`), right-click `contextMenus`, `activeTab` URL inspection, `chrome.storage.local` sync. |
| **Frontend UI** | **React 18 + Vite 5** | Component-driven popup UI (`App.jsx`), fast builds, instant language switching, clean error boundaries. |
| **Styling** | **Tailwind CSS 3 + PostCSS** | Dark cybersecurity aesthetic, color-coded severity badges, smooth risk meter animations. |
| **Backend Core** | **Go (Golang) 1.21+** | Standard library only (`net/http`, `encoding/json`, `bufio`). Zero external dependencies, sub-millisecond routing overhead, 60s timeout resilience. |
| **AI Engine** | **Google Gemini API** | `gemini-3.6-flash` (primary) with automatic failover to `gemini-3.5-flash`, structured JSON output schema enforcement. |
| **Documentation** | **HTML5 / Tailwind CSS** | Formal printable project report in [`doc/index.html`](doc/index.html). |
| **License** | **MIT License** | OSI-approved open-source license ([`LICENSE`](LICENSE)). |

---

## 📁 Project Directory Structure

```text
JIAT-Hack-Days-2026/
├── backend/
│   ├── main.go                 # Go HTTP microservice (:8080), POST /scan, GET /health, Gemini failover
│   ├── go.mod                  # Go module definition (standard library only)
│   └── .env.example            # Template for GEMINI_API_KEY
├── extension/
│   ├── public/
│   │   └── manifest.json       # Chrome Extension Manifest V3 configuration
│   ├── src/
│   │   ├── App.jsx             # React popup UI, risk meter, EN/SI language toggle
│   │   ├── background.js       # MV3 service worker & right-click context menu handler
│   │   ├── main.jsx            # React DOM entry point
│   │   └── index.css           # Tailwind CSS layers & custom scrollbar styling
│   ├── index.html              # Popup HTML shell
│   ├── vite.config.js          # Vite bundler config (copies manifest & background.js to dist/)
│   ├── tailwind.config.js      # Tailwind theme configuration
│   ├── postcss.config.js       # PostCSS configuration
│   └── package.json            # Frontend dependencies & build scripts
├── doc/
│   └── index.html              # Official printable Technical Project Report (HTML/PDF ready)
├── .gitignore                  # Excludes .env, node_modules/, dist/, binaries
├── LICENSE                     # OSI-approved MIT License
└── README.md                   # Complete project documentation
```

---

## 🚀 Local Setup, Build & Run Guide

### Prerequisites

- **[Go 1.21+](https://go.dev/dl/)** installed and on your `PATH`
- **[Node.js 18+](https://nodejs.org/)** and `npm`
- **Google Chrome** (v127+ recommended so right-click scans can auto-open the popup)
- A **Google Gemini API Key** from [Google AI Studio](https://aistudio.google.com/app/apikey)

---

### 1️⃣ Start the Go Backend (`backend/`)

```bash
cd backend

# Create your local .env file from the template
# Windows (PowerShell):
Copy-Item .env.example .env

# macOS / Linux / Git Bash:
cp .env.example .env
```

Open `backend/.env` and paste your Gemini API key:

```env
GEMINI_API_KEY=your_real_gemini_api_key_here
```

Run the backend server:

```bash
go run main.go
```

You should see:

```text
ScamShield AI backend listening on http://localhost:8080
```

**Optional Health & Scan Verification:**

```bash
# Check server health & API key readiness
curl http://localhost:8080/health
# Expected: {"api_key_ready":true,"service":"ScamShield AI","status":"ok"}

# Test a live scam scan from the terminal
curl -X POST http://localhost:8080/scan \
  -H "Content-Type: application/json" \
  -d "{\"text\":\"Your bank account is suspended. Send your OTP immediately to unlock.\",\"language\":\"English\"}"
```

---

### 2️⃣ Build the Chrome Extension (`extension/`)

Open a new terminal window:

```bash
cd extension
npm install
npm run build
```

This compiles the React application and outputs a ready-to-load `extension/dist/` folder containing `manifest.json`, `background.js`, and the bundled popup UI.

---

### 3️⃣ Load the Extension in Google Chrome

1. Open Google Chrome and navigate to `chrome://extensions`.
2. Toggle on **Developer mode** (top-right corner).
3. Click **Load unpacked** (top-left).
4. Select the **`extension/dist`** folder inside this repository.
5. Pin **ScamShield AI** (🛡️) to your Chrome toolbar.

---

### 4️⃣ How to Use ScamShield AI

- **Method A — Right-Click Context Scan:** Highlight any suspicious message, email, or social post on any webpage → right-click → click **"Scan with ScamShield AI"**.
- **Method B — Manual Text Scan:** Click the **ScamShield AI** toolbar icon, choose **English** or **සිංහල**, paste any text into the input box, and click **Scan Text** (`පෙළ පරීක්ෂා කරන්න`).
- **Method C — Current Tab URL Check:** Click **Scan Current URL** (`වර්තමාන URL පරීක්ෂා කරන්න`) to inspect the webpage URL you are currently viewing.

> 💡 *Note: Keep `go run main.go` running in your terminal while using the extension. If the backend is stopped, the popup gracefully displays a helpful offline instructions banner.*

---

## 🔌 REST API Reference

### `POST /scan`

Analyses a piece of text or a URL for phishing, financial fraud, social engineering, or fake news indicators.

**Request Headers:**
```http
Content-Type: application/json
```

**Request Body:**
```json
{
  "text": "Congratulations! You won Rs. 500,000 in the national mobile draw. Send your OTP and bank account number to claim.",
  "language": "English"
}
```
*(Set `"language"` to `"English"` or `"Sinhala"`)*

**Success Response (`200 OK`):**
```json
{
  "risk_score": 94,
  "verdict": "High Risk",
  "summary": "A classic prize-bait phishing scam attempting to steal banking credentials and OTP codes.",
  "red_flags": [
    "Unsolicited lottery or prize notification",
    "Direct request for confidential OTP and banking details",
    "Uses artificial urgency to bypass critical thinking"
  ],
  "advice": "Do not reply or share any OTP or account numbers. Delete the message immediately and block the sender."
}
```

**Error Responses:**
- `400 Bad Request` — Invalid JSON body or empty `text` field.
- `500 Internal Server Error` — `GEMINI_API_KEY` is missing on the server.
- `502 Bad Gateway` — Upstream Gemini API error after primary and fallback attempts.

---

### `GET /health`

Lightweight readiness probe for debugging and monitoring.

**Success Response (`200 OK`):**
```json
{
  "api_key_ready": true,
  "service": "ScamShield AI",
  "status": "ok"
}
```

---

## 🧪 Empirical Test Matrix & Real-World Scenarios

| # | Threat Scenario | Sample Input Tested | Risk Score | Verdict | Key Red Flags Identified |
| :-: | :--- | :--- | :-: | :-: | :--- |
| 1 | **Banking OTP Phishing** | *"Your Commercial Bank account is temporarily suspended. Reply with your 6-digit OTP to verify identity."* | **92 / 100** | 🔴 **High Risk** | Bank impersonation, urgent threat, asks for confidential OTP. |
| 2 | **Lottery / Prize Fraud** | *"Congratulations! Your mobile number won Rs. 500,000. Pay Rs. 2,500 processing fee to claim now."* | **88 / 100** | 🔴 **High Risk** | Unsolicited prize claim, advance-fee fraud pattern. |
| 3 | **Sinhala Fake News / Panic** | *"හදිසි නිවේදනයයි: හෙට සිට සියලුම බැංකු වසා දැමේ. වහාම ඔබගේ මුදල් ලබාගන්න."* | **85 / 100** | 🔴 **High Risk** | Panic-inducing financial rumor, no official Central Bank source. |
| 4 | **Suspicious Shortened Link** | *"http://bit.ly/claim-govt-subsidy-2026-lk"* | **65 / 100** | 🟡 **Suspicious** | Obfuscated destination URL, government subsidy lure. |
| 5 | **Legitimate Payment Receipt** | *"Your monthly bill payment of Rs. 2,450.00 was received with thanks. Reference: 8947291."* | **08 / 100** | 🟢 **Safe** | Informational receipt only, no links, no credential requests. |

---

## 👥 Team Contributions & Ownership

| # | Member | Role | Primary Deliverables & Ownership |
| :-: | :--- | :--- | :--- |
| 1 | **Shonali** | Team Lead | Project leadership, system architecture planning, workflow coordination, and final integration review. |
| 2 | **Pathum** | Backend Developer | Go backend microservice architecture, Gemini API integration, model fallback logic, Git coordination. |
| 3 | **Dilshan** | Frontend Developer | React popup component development, Tailwind CSS dark UI styling, Sinhala language toggle, Chrome packaging. |
| 4 | **Pathum** | QA Specialist | Empirical validation matrix, testing real-world phishing and fake news scenarios, edge-case failure testing. |
| 5 | **Ruwani** | Documentation Lead | Technical report authoring, MIT License verification, AI disclosure documentation, OrganizerHQ entry. |
| 6 | **Shalini** | Media & Presentation | Track 2 video directing & scriptwriting, YouTube Unlisted submission, stage demo preparation and pitching. |

> **Guild Roster (5 Registered Members):** Shonali, Pathum *(Backend Developer & QA Specialist)*, Dilshan, Ruwani, Shalini. Every member actively contributed to the build, testing, documentation, and presentation in accordance with the JIAT Hack Days 2026 Handbook.

---

## 🤖 AI Disclosure & Security Compliance

In full compliance with **Section 5 (AI & One Project Rules)** and **Section 7 (Track 1 — Best Use of the Gemini API)** of the JIAT Hack Days Gampaha 2026 Participant Handbook:

1. **AI Running Inside the Product:**
   - **Google Gemini API (`gemini-3.6-flash` with automatic fallback to `gemini-3.5-flash`)** runs live inside the product via the Go backend (`POST /scan`).
   - It performs all real-time semantic analysis of suspicious messages and URLs, calculates the `0–100` risk score, classifies the verdict (`Safe`, `Suspicious`, `High Risk`), extracts red flags, and synthesizes actionable safety guidance in **English** or **Sinhala (සිංහල)**.
2. **AI Tools Used During Development:**
   - **Antigravity IDE** and **Google Gemini** were used during the hackathon to assist with project scaffolding, code debugging, and documentation formatting.
3. **Gemini API Project Number (Track 1 Requirement):**
   - **Google Cloud / Gemini API Project Number:** `734368450504`
4. **Secret & Key Hygiene:**
   - **Zero API keys, tokens, or passwords** are committed to this repository.
   - `backend/.env` is strictly excluded via [`.gitignore`](.gitignore), and only a safe placeholder template ([`backend/.env.example`](backend/.env.example)) is tracked in Git.

---

## 📄 License & Disclaimer

### License
This project is open-source software released under the **[MIT License](LICENSE)** (OSI-approved).

### Disclaimer
**ScamShield AI** is an AI-powered decision-support tool built during **JIAT Hack Days Gampaha 2026**. While it accurately highlights common phishing and fraud patterns, generative AI models can occasionally make mistakes. Always verify critical banking, financial, or personal requests directly with the official organization through verified contact channels.
