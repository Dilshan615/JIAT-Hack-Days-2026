# 🛡️ ScamShield AI

**ScamShield AI** is a real-time cybersecurity and fact-checking **Chrome Browser Extension (Manifest V3)**
that protects users from phishing, financial fraud and fake news — in both **English** and **සිංහල (Sinhala)**.

The extension talks to a lightweight **Go (Golang)** backend, which forwards the suspect text or URL to the
**Google Gemini API (`gemini-3.6-flash`)** and returns a structured, easy-to-read risk report in under a few seconds.

> Built for **JIAT Hack Days Gampaha 2026** 🏆

---

## ✨ Key Features

| Feature | Description |
| --- | --- |
| 🖱️ **Right-click text scanning** | Select any suspicious text on a page → right-click → **"Scan with ScamShield AI"**. The popup opens and scans it automatically. |
| 🔗 **Current URL phishing check** | One click analyses the URL of the tab you are on for look-alike, shortened or malicious domains. |
| 📊 **Color-coded Risk Score (0–100%)** | 🟢 Safe (< 35) · 🟡 Suspicious (35–69) · 🔴 High Risk (70+) |
| 🧾 **Detailed verdict report** | Every scan returns a **Verdict**, a plain-language **Summary**, a **Red Flags** bullet list and actionable **Advice**. |
| 🌐 **English / සිංහල toggle** | The entire analysis — summary, red flags and advice — can be generated in Sinhala or English. |
| 🌑 **Sleek dark UI** | A 360px cybersecurity-styled popup built with Tailwind CSS. |

---

## 🧱 Tech Stack

| Layer | Technology |
| --- | --- |
| Browser Extension | **Chrome Extension — Manifest V3** (service worker background, `contextMenus`, `storage`, `activeTab`) |
| Frontend | **React 18** + **Vite** + **Tailwind CSS** |
| Backend | **Go (Golang)** — standard library only (`net/http`, `encoding/json`) |
| AI Engine | **Google Gemini API** — `gemini-3.6-flash` |
| License | **MIT** |

---

## 🚀 Local Setup

### Prerequisites

- [Go 1.21+](https://go.dev/dl/)
- [Node.js 18+](https://nodejs.org/)
- Google Chrome (v127+ recommended, so the popup can auto-open)
- A Gemini API key from [Google AI Studio](https://aistudio.google.com/app/apikey)

### 1️⃣ Backend (`backend/`)

```bash
cd backend

# Create your local environment file
cp .env.example .env        # Windows PowerShell: Copy-Item .env.example .env
```

Open `backend/.env` and add your key:

```
GEMINI_API_KEY=your_api_key_here
```

Start the server:

```bash
go run main.go
```

You should see `ScamShield AI backend listening on http://localhost:8080`.

**Verify it is alive:**

```bash
curl http://localhost:8080/health
# {"api_key_ready":true,"service":"ScamShield AI","status":"ok"}
```

### 2️⃣ Extension (`extension/`)

```bash
cd extension
npm install
npm run build
```

This produces a ready-to-load `dist/` folder containing `manifest.json`, `background.js` and the React popup.

### 3️⃣ Load it in Chrome

1. Go to `chrome://extensions`
2. Enable **Developer mode** (top-right)
3. Click **Load unpacked**
4. Select the **`extension/dist`** folder
5. Pin **ScamShield AI** to the toolbar — you're ready to scan

> Keep the Go backend running in a terminal whenever you use the extension. If it is not running the
> popup shows a friendly "backend offline" message instead of crashing.

---

## 👥 Team Contributions

| # | Member | Role | Contributions |
| --- | --- | --- | --- |
| 1 | **Pathum** | Team Lead | Go (Golang) backend architecture, Gemini API integration, Chrome Extension core setup, and GitHub deployment |
| 2 | **Manja** | Frontend Developer | Frontend UI/UX polish (React & Tailwind CSS), Dark Mode styling, and extension testing |
| 3 | **Shonali** | Quality Assurance | QA, real-world scam/phishing & Sinhala fake-news test cases, and demo preparation |
| 4 | **Shalini** | Documentation | Documentation (`README.md`, MIT License verification), AI disclosure, and OrganizerHQ submission management |
| 5 | **Ruwini** | Media & Presentation | Track 2 Social Media Campaign video direction, scriptwriting, filming, and YouTube presentation |

---

## 🤖 AI Disclosure

- **Google Gemini API (`gemini-3.6-flash`)** runs inside the Go backend through the `POST /scan` endpoint
  and performs all real-time scam, phishing and fake-news analysis. The backend only builds the prompt,
  forwards the text and validates the JSON that comes back.
- **Gemini API Project Number:** `734368450504`. API keys are **never** committed — they are excluded
  via `.gitignore` and loaded at runtime from the `GEMINI_API_KEY` environment variable (or a local
  `backend/.env`).
- **Antigravity IDE** and **Google Gemini** were used to assist with scaffolding, debugging and documentation.

---

## 📄 License

This project is released under the **MIT License**. See the [LICENSE](LICENSE) file for the full text.

---

## ⚠️ Disclaimer

ScamShield AI is a decision-support tool built for a hackathon. AI models can be wrong — always verify
critical financial or personal decisions with the official source before acting.
