# 🛡️ ScamShield AI

> AI-powered, real-time detection of phishing, financial fraud and fake news — right inside your browser.

ScamShield AI is a hackathon project that puts a Gemini-powered scam detector one right-click away.
Select any suspicious SMS, email, social post or link on a web page, choose **"Scan with ScamShield AI"**
from the context menu, and get an instant risk score with the red flags and the advice you need.

---

## ✨ Features

- **Right-click scanning** — select text on any page and scan it from the context menu.
- **Risk score 0–100** with a colour-coded meter (🟢 Safe < 35 · 🟡 Suspicious 35–69 · 🔴 High Risk 70+).
- **Verdict, summary, red flags and actionable advice** for every scan.
- **Bilingual output** — full **English** and **සිංහල (Sinhala)** support.
- **Scan the current URL** without copying anything.
- **Sleek dark cybersecurity UI** built with Tailwind CSS.

---

## 🧱 Tech Stack

| Layer | Technology |
| --- | --- |
| Backend | **Go (Golang)** — standard library only (`net/http`, `encoding/json`) |
| Frontend | **React 18** + **Vite 5** |
| Styling | **Tailwind CSS 3** (dark, 360px popup) |
| Extension | **Chrome Extension Manifest V3** (service worker background) |
| AI | **Google Gemini API** — `gemini-3.6-flash` |

---

## 📁 Project Structure

```
scamshield-backend/
├── backend/
│   ├── main.go            # HTTP server on :8080, POST /scan -> Gemini
│   ├── go.mod
│   └── .env.example       # GEMINI_API_KEY template
├── extension/
│   ├── public/manifest.json   # MV3 manifest (copied to dist/)
│   ├── src/
│   │   ├── App.jsx            # React popup UI
│   │   ├── background.js      # service worker + context menu
│   │   ├── main.jsx
│   │   └── index.css
│   ├── index.html
│   ├── vite.config.js
│   ├── tailwind.config.js
│   └── package.json
├── .gitignore
├── LICENSE
└── README.md
```

---

## 🚀 Setup & Run

### Prerequisites

- [Go 1.21+](https://go.dev/dl/)
- [Node.js 18+](https://nodejs.org/)
- Google Chrome (version 127 or newer is recommended, so the popup can auto-open)
- A Gemini API key from [Google AI Studio](https://aistudio.google.com/app/apikey)

### 1. Backend (Go)

```bash
cd backend

# Windows (PowerShell)
Copy-Item .env.example .env

# macOS / Linux
cp .env.example .env
```

Open `backend/.env` and put your key in it:

```
GEMINI_API_KEY=your_real_gemini_api_key
```

Then start the server:

```bash
go run main.go
```

You should see:

```
ScamShield AI backend listening on http://localhost:8080
```

Quick sanity check (optional):

```bash
curl -X POST http://localhost:8080/scan \
  -H "Content-Type: application/json" \
  -d "{\"text\":\"Your bank account is blocked. Send your OTP now to unlock it.\",\"language\":\"English\"}"
```

### 2. Chrome Extension (React + Vite)

```bash
cd extension
npm install
npm run build
```

The build produces a `dist/` folder containing `manifest.json`, `background.js` and the React popup.

### 3. Load it in Chrome

1. Open `chrome://extensions`
2. Turn on **Developer mode** (top-right)
3. Click **Load unpacked**
4. Select the `extension/dist` folder
5. Pin **ScamShield AI** to the toolbar

### 4. Use it

- **Option A — Right click:** select any suspicious text on a page → right click → **Scan with ScamShield AI**.
- **Option B — Popup:** click the toolbar icon, paste the text, hit **Scan Text**.
- **Option C — URL:** click **Scan Current URL** to analyse the page you are on.

> During development you can run `npm run dev` for hot reload, but the popup only gets access to
> `chrome.*` APIs when it is loaded as an unpacked extension from `dist/`.

---

## 🔌 API Reference

### `POST /scan`

Request:

```json
{
  "text": "Congratulations! You won Rs. 500,000. Send your OTP to claim.",
  "language": "English"
}
```

Response:

```json
{
  "risk_score": 92,
  "verdict": "High Risk",
  "summary": "A classic prize-bait SMS that tries to harvest your OTP.",
  "red_flags": [
    "Unsolicited prize claim",
    "Requests an OTP / banking credentials",
    "Creates artificial urgency"
  ],
  "advice": "Do not reply or share any code. Delete the message and report the sender."
}
```

### `GET /health`

Returns `{"status":"ok","service":"ScamShield AI","api_key_ready":true}` — handy for debugging.

---

## 👥 Team Contributions

| # | Team Member | Role | Contributions |
| --- | --- | --- | --- |
| 1 | _Member 1_ | Team Lead | Project planning, architecture, integration |
| 2 | _Member 2_ | Backend Developer | Go API, Gemini integration, prompt engineering |
| 3 | _Member 3_ | Frontend Developer | React popup UI, Tailwind styling, i18n toggle |
| 4 | _Member 4_ | Extension Developer | Manifest V3, service worker, context menu |
| 5 | _Member 5_ | QA & Documentation | Testing, README, demo video, presentation |

---

## 🤖 AI Disclosure

- **Google Gemini API (`gemini-3.6-flash`)** powers the core of this project. All real-time scam,
  phishing, financial-fraud and fake-news analysis — the risk score, verdict, summary, red flags
  and advice — is generated by Gemini. The Go backend only prepares the prompt, forwards the text
  and validates the JSON that comes back.
- **Antigravity** was used for scaffolding the repository (project structure, boilerplate and
  configuration files).
- The system prompt and response schema are hand-written to keep every answer machine-readable:
  `{"risk_score": 0-100, "verdict": "Safe | Suspicious | High Risk", "summary": "...", "red_flags": ["..."], "advice": "..."}`

**Security note:** the Gemini API key is never hardcoded. It is read from the `GEMINI_API_KEY`
environment variable (or a local `backend/.env` that is git-ignored).

---

## 📄 License

Released under the [MIT License](LICENSE).

---

## ⚠️ Disclaimer

ScamShield AI is a decision-support tool built for a hackathon. AI can be wrong — always verify
critical financial or personal decisions with the official source before acting.
