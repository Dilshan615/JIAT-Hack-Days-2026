import { useCallback, useEffect, useRef, useState } from 'react'

const API_BASE = 'http://localhost:8080'

const TEXT = {
  English: {
    tagline: 'Real-time scam & fake news detection',
    placeholder:
      'Paste a suspicious SMS, email, message or link here...\n\nExample: "Congratulations! You won Rs. 500,000. Send your OTP and bank details to claim."',
    scanText: 'Scan Text',
    scanUrl: 'Scan Current URL',
    scanning: 'Analyzing with Gemini AI...',
    result: 'Analysis Result',
    riskScore: 'Risk Score',
    verdict: 'Verdict',
    summary: 'Summary',
    redFlags: 'Red Flags',
    advice: 'Actionable Advice',
    noFlags: 'No red flags detected.',
    empty: 'Please paste some text or scan the current URL first.',
    offline:
      "Can't reach the ScamShield AI backend at " +
      API_BASE +
      '. Start it with "go run main.go" inside the backend folder.',
    failed: 'Scan failed',
    brand: 'ScamShield AI',
  },
  Sinhala: {
    tagline: 'තථ්‍ය කාලීන වංචා හා ව්‍යාජ ප්‍රවෘත්ති හඳුනාගැනීම',
    placeholder:
      'සැක සහිත SMS, ඊමේල්, පණිවිඩයක් හෝ සබැඳියක් මෙහි අලවන්න...\n\nඋදා: "සුබ පැතුම්! ඔබට රු. 500,000 ක් දිනා ඇත. OTP අංකය සහ බැංකු විස්තර එවන්න."',
    scanText: 'පෙළ පරීක්ෂා කරන්න',
    scanUrl: 'වර්තමාන URL පරීක්ෂා කරන්න',
    scanning: 'Gemini AI මගින් විශ්ලේෂණය කරමින්...',
    result: 'විශ්ලේෂණ ප්‍රතිඵලය',
    riskScore: 'අවදානම් අගය',
    verdict: 'නිගමනය',
    summary: 'සාරාංශය',
    redFlags: 'අනතුරු ඇඟවීම්',
    advice: 'ප්‍රායෝගික උපදෙස්',
    noFlags: 'අනතුරු ඇඟවීමක් හමු නොවිණි.',
    empty: 'කරුණාකර පළමුව පෙළක් අලවන්න හෝ URL පරීක්ෂා කරන්න.',
    offline:
      'ScamShield AI backend (' +
      API_BASE +
      ') වෙත සම්බන්ධ විය නොහැක. backend ෆෝල්ඩරයේ "go run main.go" ධාවනය කරන්න.',
    failed: 'පරීක්ෂාව අසාර්ථක විය',
    brand: 'ScamShield AI',
  },
}

const VERDICT_STYLES = {
  Safe: {
    badge: 'bg-emerald-500/15 text-emerald-300 border-emerald-500/40',
    bar: 'bg-emerald-500',
    glow: 'shadow-[0_0_25px_-8px_rgba(16,185,129,0.8)]',
    icon: '✓',
  },
  Suspicious: {
    badge: 'bg-amber-500/15 text-amber-300 border-amber-500/40',
    bar: 'bg-amber-500',
    glow: 'shadow-[0_0_25px_-8px_rgba(245,158,11,0.8)]',
    icon: '!',
  },
  'High Risk': {
    badge: 'bg-red-500/15 text-red-300 border-red-500/40',
    bar: 'bg-red-500',
    glow: 'shadow-[0_0_25px_-8px_rgba(239,68,68,0.8)]',
    icon: '✕',
  },
}

function riskBand(score) {
  if (score >= 70) return 'High Risk'
  if (score >= 35) return 'Suspicious'
  return 'Safe'
}

export default function App() {
  const [language, setLanguage] = useState('English')
  const [text, setText] = useState('')
  const [loading, setLoading] = useState(false)
  const [result, setResult] = useState(null)
  const [error, setError] = useState(null)
  const [sourceUrl, setSourceUrl] = useState('')
  const bootRef = useRef(false)

  const t = TEXT[language]

  const runScan = useCallback(
    async (rawText, lang = language) => {
      const value = (rawText ?? text).trim()
      if (!value) {
        setError(t.empty)
        return
      }

      setLoading(true)
      setError(null)
      setResult(null)

      try {
        const res = await fetch(`${API_BASE}/scan`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ text: value, language: lang }),
        })

        let data = null
        try {
          data = await res.json()
        } catch {
          data = null
        }

        if (!res.ok) {
          throw new Error((data && data.error) || `${t.failed} (HTTP ${res.status})`)
        }

        setResult({
          risk_score: Number(data?.risk_score) || 0,
          verdict: data?.verdict || riskBand(Number(data?.risk_score) || 0),
          summary: data?.summary || '',
          red_flags: Array.isArray(data?.red_flags) ? data.red_flags : [],
          advice: data?.advice || '',
        })
      } catch (err) {
        const isNetworkError = err instanceof TypeError
        setError(isNetworkError ? t.offline : err.message || t.failed)
      } finally {
        setLoading(false)
      }
    },
    [language, text, t],
  )

  const scanCurrentUrl = useCallback(async () => {
    setError(null)

    if (typeof chrome === 'undefined' || !chrome.tabs?.query) {
      setError(t.offline)
      return
    }

    try {
      const [tab] = await chrome.tabs.query({ active: true, currentWindow: true })
      const url = tab?.url || ''
      if (!url || url.startsWith('chrome://') || url.startsWith('edge://')) {
        setError(t.empty)
        return
      }
      setSourceUrl(url)
      setText(url)
      await runScan(url)
    } catch {
      setError(t.offline)
    }
  }, [runScan, t])

  // Pick up text handed over by the right-click context menu.
  useEffect(() => {
    if (bootRef.current) return
    bootRef.current = true

    if (typeof chrome === 'undefined' || !chrome.storage?.local) return

    chrome.storage.local.get(['pendingScan', 'autoScan'], (data) => {
      if (!data?.pendingScan) return
      const pending = data.pendingScan
      const shouldAutoScan = Boolean(data.autoScan)
      chrome.storage.local.remove(['pendingScan', 'autoScan'])

      setText(pending)
      if (shouldAutoScan) runScan(pending)
    })
  }, [runScan])

  const band = result ? riskBand(result.risk_score) : null
  const styles = band ? VERDICT_STYLES[result.verdict] || VERDICT_STYLES[band] : null

  return (
    <div className="shield-grid min-h-[520px] w-[360px] bg-shield-bg text-slate-200">
      {/* ------------------------------------------------ header */}
      <header className="flex items-center justify-between gap-3 border-b border-shield-border bg-shield-panel/80 px-4 py-3 backdrop-blur">
        <div className="flex items-center gap-2.5">
          <span className="relative flex h-9 w-9 items-center justify-center rounded-xl bg-gradient-to-br from-cyan-400 to-blue-600 text-lg shadow-[0_0_20px_-6px_rgba(34,211,238,0.9)]">
            🛡️
          </span>
          <div className="leading-tight">
            <h1 className="text-[15px] font-bold tracking-tight text-white">
              {t.brand}
            </h1>
            <p className="text-[10px] font-medium text-cyan-300/80">{t.tagline}</p>
          </div>
        </div>

        <div className="flex rounded-lg border border-shield-border bg-shield-bg p-0.5 text-[11px] font-semibold">
          {['English', 'Sinhala'].map((lang) => (
            <button
              key={lang}
              onClick={() => setLanguage(lang)}
              className={
                'rounded-md px-2.5 py-1 transition-all duration-200 ' +
                (language === lang
                  ? 'bg-cyan-500/20 text-cyan-200 shadow-[0_0_12px_-4px_rgba(34,211,238,0.9)]'
                  : 'text-slate-400 hover:text-slate-200')
              }
            >
              {lang === 'English' ? 'English' : 'සිංහල'}
            </button>
          ))}
        </div>
      </header>

      <main className="space-y-3 p-4">
        {/* ------------------------------------------------ input */}
        <textarea
          value={text}
          onChange={(e) => setText(e.target.value)}
          spellCheck={false}
          placeholder={t.placeholder}
          className="h-28 w-full resize-none rounded-xl border border-shield-border bg-shield-card/70 p-3 text-[12.5px] leading-relaxed text-slate-100 placeholder-slate-500 outline-none transition focus:border-cyan-400/60 focus:ring-2 focus:ring-cyan-400/20"
        />

        <div className="flex gap-2">
          <button
            onClick={() => runScan()}
            disabled={loading}
            className="flex-1 rounded-xl bg-gradient-to-r from-cyan-500 to-blue-600 px-3 py-2.5 text-[12.5px] font-bold text-white transition active:scale-[0.98] disabled:cursor-not-allowed disabled:opacity-50"
          >
            {t.scanText}
          </button>
          <button
            onClick={scanCurrentUrl}
            disabled={loading}
            className="flex-1 rounded-xl border border-shield-border bg-shield-card px-3 py-2.5 text-[12.5px] font-bold text-slate-200 transition hover:border-cyan-400/50 hover:text-cyan-200 active:scale-[0.98] disabled:cursor-not-allowed disabled:opacity-50"
          >
            {t.scanUrl}
          </button>
        </div>

        {sourceUrl && (
          <p className="truncate rounded-lg bg-shield-card/60 px-2.5 py-1.5 text-[10px] text-slate-400">
            {sourceUrl}
          </p>
        )}

        {/* ------------------------------------------------ loading */}
        {loading && (
          <div className="flex items-center gap-3 rounded-xl border border-cyan-500/30 bg-cyan-500/5 px-3 py-4">
            <span className="h-4 w-4 animate-spin rounded-full border-2 border-cyan-400/30 border-t-cyan-400" />
            <span className="text-[12px] font-medium text-cyan-200">{t.scanning}</span>
          </div>
        )}

        {/* ------------------------------------------------ error */}
        {error && !loading && (
          <div className="rounded-xl border border-red-500/40 bg-red-500/10 px-3 py-3">
            <p className="text-[11.5px] font-semibold text-red-300">{t.failed}</p>
            <p className="mt-1 text-[11px] leading-relaxed text-red-200/80">{error}</p>
          </div>
        )}

        {/* ------------------------------------------------ result */}
        {result && !loading && styles && (
          <section
            className={
              'animate-[fadeIn_0.35s_ease-out] space-y-3 rounded-xl border border-shield-border bg-shield-card/80 p-3.5 ' +
              styles.glow
            }
          >
            <div className="flex items-center justify-between">
              <span className="text-[10px] font-bold uppercase tracking-widest text-slate-400">
                {t.result}
              </span>
              <span
                className={
                  'rounded-full border px-2.5 py-0.5 text-[10.5px] font-bold ' + styles.badge
                }
              >
                {styles.icon} {result.verdict}
              </span>
            </div>

            {/* risk meter */}
            <div>
              <div className="mb-1.5 flex items-center justify-between text-[11px]">
                <span className="font-semibold text-slate-300">{t.riskScore}</span>
                <span className="font-bold text-white">
                  {result.risk_score}
                  <span className="text-slate-500">/100</span>
                </span>
              </div>
              <div className="h-2 w-full overflow-hidden rounded-full bg-shield-bg">
                <div
                  className={'h-full rounded-full transition-all duration-700 ' + styles.bar}
                  style={{ width: `${Math.min(Math.max(result.risk_score, 0), 100)}%` }}
                />
              </div>
            </div>

            {result.summary && (
              <div>
                <p className="text-[10px] font-bold uppercase tracking-widest text-slate-400">
                  {t.summary}
                </p>
                <p className="mt-1 text-[12px] leading-relaxed text-slate-200">
                  {result.summary}
                </p>
              </div>
            )}

            <div>
              <p className="text-[10px] font-bold uppercase tracking-widest text-slate-400">
                {t.redFlags}
              </p>
              {result.red_flags.length === 0 ? (
                <p className="mt-1 text-[11.5px] text-emerald-300">{t.noFlags}</p>
              ) : (
                <ul className="mt-1.5 space-y-1.5">
                  {result.red_flags.map((flag, index) => (
                    <li key={index} className="flex gap-2 text-[11.5px] leading-relaxed text-slate-200">
                      <span className="mt-[3px] h-1.5 w-1.5 shrink-0 rounded-full bg-red-400" />
                      <span>{flag}</span>
                    </li>
                  ))}
                </ul>
              )}
            </div>

            {result.advice && (
              <div className="rounded-lg border border-cyan-500/25 bg-cyan-500/10 p-2.5">
                <p className="text-[10px] font-bold uppercase tracking-widest text-cyan-300">
                  {t.advice}
                </p>
                <p className="mt-1 text-[11.5px] leading-relaxed text-cyan-100">{result.advice}</p>
              </div>
            )}
          </section>
        )}
      </main>

      <footer className="border-t border-shield-border px-4 py-2 text-center text-[9.5px] text-slate-500">
        Powered by Gemini 3.6 Flash · ScamShield AI
      </footer>
    </div>
  )
}
