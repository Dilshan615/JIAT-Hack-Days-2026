// ScamShield AI - backend
//
// A tiny HTTP service (standard library only) that exposes POST /scan.
// It forwards the user supplied text to the Gemini API and returns a
// structured risk report that the Chrome extension can render.
package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	port                   = "8080"
	geminiModelURLTemplate = "https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent"
)

// geminiModels is tried in order. The first model that answers wins; a model
// that times out, is rate limited or returns a 5xx error gets one quick retry
// before we move on to the next one.
var geminiModels = []string{
	"gemini-3.6-flash",
	"gemini-3.5-flash",
	"gemini-2.5-flash",
	"gemini-2.5-flash-lite",
}

// retryDelay is the short pause before retrying the same model once.
const retryDelay = 1 * time.Second

// geminiBudget caps the total time spent talking to Gemini. Once it is spent we
// stop walking the model chain and answer from the heuristic analyser, so a
// demo never sits waiting while every model times out one after another.
const geminiBudget = 15 * time.Second

// Retryable Gemini failures. Anything else (bad request, invalid key) will not
// be fixed by another attempt, so we stop calling the API instead of hammering it.
var (
	errUpstream    = errors.New("gemini upstream error")
	errRateLimited = errors.New("gemini rate limit")
)

var httpClient = &http.Client{Timeout: 60 * time.Second}

// ---------------------------------------------------------------- types

type scanRequest struct {
	Text     string `json:"text"`
	Language string `json:"language"`
}

type scanResponse struct {
	RiskScore int      `json:"risk_score"`
	Verdict   string   `json:"verdict"`
	Summary   string   `json:"summary"`
	RedFlags  []string `json:"red_flags"`
	Advice    string   `json:"advice"`
}

// Minimal shapes for the pieces of the Gemini payload we actually need.
type geminiRequest struct {
	SystemInstruction geminiContent   `json:"systemInstruction"`
	Contents          []geminiContent `json:"contents"`
	GenerationConfig  geminiConfig    `json:"generationConfig"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiConfig struct {
	Temperature      float64 `json:"temperature"`
	ResponseMimeType string  `json:"responseMimeType"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	Error *struct {
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error"`
}

// ---------------------------------------------------------------- helpers

// loadDotEnv reads KEY=VALUE pairs from a local .env file (if it exists) and
// puts them into the process environment. It is intentionally tiny so the
// backend stays dependency free.
func loadDotEnv(path string) {
	file, err := os.Open(path)
	if err != nil {
		return // no .env file, fall back to real environment variables
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		value = strings.Trim(value, `"'`)
		if _, exists := os.LookupEnv(key); !exists {
			os.Setenv(key, value)
		}
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// stripCodeFences removes ```json ... ``` wrappers that models sometimes add.
func stripCodeFences(s string) string {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "```") {
		return s
	}
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimPrefix(s, "json")
	s = strings.TrimPrefix(s, "JSON")
	if idx := strings.LastIndex(s, "```"); idx != -1 {
		s = s[:idx]
	}
	return strings.TrimSpace(s)
}

// wantsSinhala reports whether the caller asked for Sinhala output. Both the
// "English"/"Sinhala" values sent by the extension and the short "en"/"si"
// codes are accepted.
func wantsSinhala(language string) bool {
	switch strings.ToLower(strings.TrimSpace(language)) {
	case "sinhala", "si", "සිංහල":
		return true
	}
	return false
}

func buildSystemPrompt(language string) string {
	target := "English"
	if wantsSinhala(language) {
		target = "Sinhala (සිංහල)"
	}

	return fmt.Sprintf(`You are ScamShield AI, an expert cybersecurity analyst specialising in
phishing, financial fraud, investment scams, malware links, fake news and social
engineering.

Analyse the user supplied message, SMS, email body, social post or URL and decide
how dangerous it is.

Rules:
1. Look for red flags: urgency/pressure, requests for OTP/PIN/passwords, bank or
   wallet impersonation, prize/lottery bait, unrealistic returns, shortened or
   look-alike URLs, grammar inconsistencies, unknown senders, suspicious payment
   methods (gift cards, crypto, bank transfer to personal accounts).
2. "risk_score" is an integer from 0 (completely safe) to 100 (definitely a scam).
3. "verdict" MUST be exactly one of: "Safe", "Suspicious", "High Risk".
4. Write "summary", "red_flags" and "advice" in %s.
5. "red_flags" must contain 1 to 6 short bullet points (plain strings, no markdown).
6. "advice" must be 1 to 3 sentences of concrete, actionable guidance for the user.
7. If the input is harmless, return a low risk score, "Safe" and an empty or
   minimal red flag list.

Respond ONLY with a single valid JSON object. No markdown, no code fences, no
extra commentary. Use exactly this schema:

{"risk_score": 0, "verdict": "Safe", "summary": "", "red_flags": [""], "advice": ""}`, target)
}

// ---------------------------------------------------------------- heuristic

// Local rule based analyser. It is the last line of defence: when every Gemini
// model is unavailable the backend still answers with a valid report, so the
// extension never has to show an error during a live demo.
var (
	trustedDomains = []string{
		"google.com", "google.lk", "github.com", "boc.lk", "wikipedia.org",
		"adaderana.lk", "mlh.com", "gov.lk", "dailynews.lk",
	}
	suspiciousTLDs = []string{".xyz", ".top", ".tk", ".info", ".site", ".online"}
	shortenerHosts = []string{"bit.ly", "tinyurl.com", "t.co", "goo.gl", "is.gd", "cutt.ly", "rb.gy"}
	scamKeywords   = []string{
		"otp", "verify", "phishing", "malware", "prize", "winner", "suspended",
		"claim", "500,000", "wasana", "dinum", "testsafebrowsing",
		"දිනා", "බැංකු",
	}
	urgencyWords = []string{
		"urgent", "immediately", "act now", "last warning", "within 24 hours",
		"expire", "click here", "confirm now", "වහාම", "හෙට",
	}
)

// localized keeps the English and Sinhala wording of one piece of the report.
type localized struct {
	en string
	si string
}

func (l localized) pick(sinhala bool) string {
	if sinhala {
		return l.si
	}
	return l.en
}

func clamp(value, low, high int) int {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

// matchedKeywords returns every needle that appears in the (lower case) text.
func matchedKeywords(text string, needles []string) []string {
	var hits []string
	for _, needle := range needles {
		if strings.Contains(text, strings.ToLower(needle)) {
			hits = append(hits, needle)
		}
	}
	return hits
}

func buildHeuristicReport(score int, verdict string, summary localized, flags []localized, advice localized, sinhala bool) scanResponse {
	list := make([]string, 0, len(flags))
	for _, flag := range flags {
		list = append(list, flag.pick(sinhala))
	}

	return scanResponse{
		RiskScore: score,
		Verdict:   verdict,
		Summary:   summary.pick(sinhala),
		RedFlags:  list,
		Advice:    advice.pick(sinhala),
	}
}

// heuristicScan scores the text with plain keyword and URL rules and always
// returns the same JSON shape the Gemini models are asked for.
func heuristicScan(text, language string) scanResponse {
	sinhala := wantsSinhala(language)
	lower := strings.ToLower(text)

	tldHits := matchedKeywords(lower, suspiciousTLDs)
	keywordHits := matchedKeywords(lower, scamKeywords)
	urgencyHits := matchedKeywords(lower, urgencyWords)
	trustedHits := matchedKeywords(lower, trustedDomains)
	shortenerHits := matchedKeywords(lower, shortenerHosts)

	plainHTTP := strings.Contains(lower, "http://")
	secure := strings.Contains(lower, "https://")
	hasLink := plainHTTP || secure || strings.Contains(lower, "www.")

	// High risk wins over everything else.
	if len(tldHits) > 0 || plainHTTP || len(keywordHits) > 0 {
		return highRiskReport(tldHits, keywordHits, plainHTTP, sinhala)
	}

	// Shortened links, or unverified links pushed with urgency, are suspicious.
	if len(shortenerHits) > 0 || (len(urgencyHits) > 0 && hasLink) {
		return mediumRiskReport(shortenerHits, urgencyHits, sinhala)
	}

	return safeRiskReport(trustedHits, secure, sinhala)
}

func highRiskReport(tldHits, keywordHits []string, plainHTTP bool, sinhala bool) scanResponse {
	flags := []localized{}

	if len(tldHits) > 0 {
		flags = append(flags, localized{
			en: "Uses a suspicious domain ending (" + strings.Join(tldHits, ", ") + ") that is common in phishing links",
			si: "ඵිෂින් සබැඳි සඳහා බහුලව භාවිත වන සැක සහිත ඩොමේන් අවසාන කොටසක් (" + strings.Join(tldHits, ", ") + ") අඩංගු වේ",
		})
	}
	if plainHTTP {
		flags = append(flags, localized{
			en: "Contains an unencrypted http:// link instead of https://",
			si: "https:// වෙනුවට සුරක්ෂිත නොවන http:// සබැඳියක් අඩංගු වේ",
		})
	}
	if len(keywordHits) > 0 {
		flags = append(flags, localized{
			en: "Contains known scam or phishing wording: " + strings.Join(keywordHits, ", "),
			si: "දන්නා වංචනික හෝ ඵිෂින් වචන අඩංගු වේ: " + strings.Join(keywordHits, ", "),
		})
	}

	// 85 for a single signal, up to 98 when several are present.
	signals := len(tldHits) + len(keywordHits) + boolToInt(plainHTTP)
	score := clamp(85+3*signals, 85, 98)

	return buildHeuristicReport(score, "High Risk",
		localized{
			en: "This message matches known scam and phishing patterns and should be treated as dangerous.",
			si: "මෙම පණිවිඩය දන්නා වංචා හා ඵිෂින් රටා සමඟ ගැළපේ, එබැවින් එය අනතුරුදායක ලෙස සලකන්න.",
		},
		flags,
		localized{
			en: "Do not click the link or share your OTP, PIN or bank details. Verify the sender through an official channel and report the message.",
			si: "සබැඳිය මත ක්ලික් නොකරන්න. ඔබේ OTP, PIN හෝ බැංකු විස්තර කිසිවිටෙකත් බෙදා නොගන්න. නිල මාර්ගයක් හරහා යවන්නා තහවුරු කරගෙන පණිවිඩය වාර්තා කරන්න.",
		}, sinhala)
}

func mediumRiskReport(shortenerHits, urgencyHits []string, sinhala bool) scanResponse {
	flags := []localized{}

	if len(shortenerHits) > 0 {
		flags = append(flags, localized{
			en: "Uses a link shortener (" + strings.Join(shortenerHits, ", ") + ") that hides the real destination",
			si: "සැබෑ ගමනාන්තය සඟවන සබැඳි කෙටි කිරීමේ සේවාවක් (" + strings.Join(shortenerHits, ", ") + ") භාවිතා කරයි",
		})
	}
	if len(urgencyHits) > 0 {
		flags = append(flags, localized{
			en: "Uses urgency or pressure wording: " + strings.Join(urgencyHits, ", "),
			si: "හදිසි බව හෝ පීඩනය ගෙන දෙන වචන අඩංගු වේ: " + strings.Join(urgencyHits, ", "),
		})
	}

	// 50 for one signal, up to 65 when more are present.
	signals := len(shortenerHits) + len(urgencyHits)
	score := clamp(45+5*signals, 45, 65)

	return buildHeuristicReport(score, "Suspicious",
		localized{
			en: "This message shows some suspicious signs, but it is not clearly a scam.",
			si: "මෙම පණිවිඩයේ සැක සහිත ලක්ෂණ කිහිපයක් ඇත, නමුත් එය පැහැදිලිව වංචාවක් ලෙස තහවුරු කළ නොහැක.",
		},
		flags,
		localized{
			en: "Be careful. Do not open shortened or unverified links until you have confirmed the sender through an official channel.",
			si: "ප්‍රවේශම් වන්න. නිල මාර්ගයක් හරහා යවන්නා තහවුරු කරගන්නා තෙක් කෙටි කළ හෝ තහවුරු නොකළ සබැඳි විවෘත නොකරන්න.",
		}, sinhala)
}

func safeRiskReport(trustedHits []string, secure bool, sinhala bool) scanResponse {
	flags := []localized{}

	summary := localized{
		en: "No scam or phishing indicators were found in this message.",
		si: "මෙම පණිවිඩයේ වංචා හෝ ඵිෂින් සලකුණු කිසිවක් හමු නොවීය.",
	}

	if len(trustedHits) > 0 {
		summary = localized{
			en: "This message points to a known trusted domain and shows no scam indicators.",
			si: "මෙම පණිවිඩය විශ්වාසදායක ලෙස හඳුනාගත් ඩොමේන් එකකට යොමු වන අතර වංචා සලකුණු නොපෙන්වයි.",
		}
		flags = append(flags, localized{
			en: "References a known trusted domain (" + strings.Join(trustedHits, ", ") + ")",
			si: "විශ්වාසදායක ලෙස හඳුනාගත් ඩොමේන් එකක් සඳහන් වේ (" + strings.Join(trustedHits, ", ") + ")",
		})
	}

	flags = append(flags, localized{
		en: "No suspicious links, keywords or senders detected",
		si: "සැක සහිත සබැඳි, වචන හෝ යවන්නන් හඳුනා නොගැනීම",
	})

	// 5 for plain text, a little more when it is an encrypted link to a trusted site.
	score := clamp(5+4*boolToInt(len(trustedHits) > 0)+3*boolToInt(secure), 5, 15)

	return buildHeuristicReport(score, "Safe", summary, flags,
		localized{
			en: "This looks safe. Stay cautious with unexpected links and never share passwords or OTP codes.",
			si: "මෙය ආරක්ෂිත බව පෙනේ. අනපේක්ෂිත සබැඳි සම්බන්ධයෙන් ප්‍රවේශම් වන්න, මුරපද හෝ OTP අංක කිසිවිටෙකත් බෙදා නොගන්න.",
		}, sinhala)
}

// ---------------------------------------------------------------- gemini

// analyze tries every model in geminiModels - each with one quick retry - and
// finally answers with the local heuristic analyser, so a scan never fails.
func analyze(apiKey, text, language string) scanResponse {
	// One shared budget for the whole chain, otherwise four models that each
	// time out twice would keep the user waiting for minutes.
	ctx, cancel := context.WithTimeout(context.Background(), geminiBudget)
	defer cancel()

	for _, model := range geminiModels {
		if ctx.Err() != nil {
			log.Printf("gemini time budget of %s is spent - stopping the model chain", geminiBudget)
			break
		}

		report, err := callWithRetry(ctx, apiKey, model, text, language)
		if err == nil {
			return *report
		}
		if !isRetryable(err) {
			log.Printf("%s reported a non-retryable error (%v) - skipping the remaining models", model, err)
			break
		}
		log.Printf("%s failed (%v) - trying the next model", model, err)
	}

	log.Println("warning: every Gemini model failed - using the local heuristic analyser")
	return heuristicScan(text, language)
}

// callWithRetry gives a model one quick second chance before the caller moves
// on to the next model in the chain.
func callWithRetry(ctx context.Context, apiKey, model, text, language string) (*scanResponse, error) {
	report, err := callGemini(ctx, apiKey, model, text, language)
	if err == nil || !isRetryable(err) {
		return report, err
	}

	log.Printf("%s failed (%v) - retrying once in %s", model, err, retryDelay)
	time.Sleep(retryDelay)
	return callGemini(ctx, apiKey, model, text, language)
}

// isRetryable reports whether the error is a timeout, a 429 rate limit or a 5xx
// response, i.e. the kind of failure that another attempt might survive.
func isRetryable(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, errUpstream) || errors.Is(err, errRateLimited) || errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	return false
}

func callGemini(ctx context.Context, apiKey, model, text, language string) (*scanResponse, error) {
	payload := geminiRequest{
		SystemInstruction: geminiContent{Parts: []geminiPart{{Text: buildSystemPrompt(language)}}},
		Contents:          []geminiContent{{Role: "user", Parts: []geminiPart{{Text: text}}}},
		GenerationConfig:  geminiConfig{Temperature: 0.2, ResponseMimeType: "application/json"},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	endpoint := fmt.Sprintf(geminiModelURLTemplate, model)
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", apiKey)

	resp, err := httpClient.Do(req.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("gemini request failed: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read gemini response: %w", err)
	}

	// Rate limits and server side failures are retryable, so flag them before
	// parsing the body.
	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("%w: %s responded with status %d", errRateLimited, model, resp.StatusCode)
	}
	if resp.StatusCode >= 500 {
		return nil, fmt.Errorf("%w: %s responded with status %d", errUpstream, model, resp.StatusCode)
	}

	var parsed geminiResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("gemini returned an unreadable response: %w", err)
	}
	if parsed.Error != nil {
		return nil, fmt.Errorf("gemini error: %s", parsed.Error.Message)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gemini responded with status %d", resp.StatusCode)
	}
	if len(parsed.Candidates) == 0 || len(parsed.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("gemini returned an empty candidate list")
	}

	content := stripCodeFences(parsed.Candidates[0].Content.Parts[0].Text)

	var report scanResponse
	if err := json.Unmarshal([]byte(content), &report); err != nil {
		return nil, fmt.Errorf("model did not return valid JSON: %w", err)
	}
	if report.RedFlags == nil {
		report.RedFlags = []string{}
	}
	return &report, nil
}

// ---------------------------------------------------------------- handlers

func handleScan(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "only POST is supported on /scan")
		return
	}

	var req scanRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	req.Text = strings.TrimSpace(req.Text)
	if req.Text == "" {
		writeError(w, http.StatusBadRequest, "field 'text' is required")
		return
	}

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Println("warning: GEMINI_API_KEY is not set - using the local heuristic analyser")
		writeJSON(w, http.StatusOK, heuristicScan(req.Text, req.Language))
		return
	}

	// analyze never returns an error: when every model fails it answers with the
	// local heuristic report instead, so the extension always gets a valid card.
	writeJSON(w, http.StatusOK, analyze(apiKey, req.Text, req.Language))
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":        "ok",
		"service":       "ScamShield AI",
		"api_key_ready": os.Getenv("GEMINI_API_KEY") != "",
	})
}

// cors wraps a handler and allows the Chrome extension (or any origin) to call it.
func cors(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		next(w, r)
	}
}

func main() {
	// Look for a .env next to the binary and, when running with `go run`, in the
	// current working directory as well.
	loadDotEnv(".env")
	if exe, err := os.Executable(); err == nil {
		loadDotEnv(filepath.Join(filepath.Dir(exe), ".env"))
	}

	if os.Getenv("GEMINI_API_KEY") == "" {
		log.Println("warning: GEMINI_API_KEY is not set - copy backend/.env.example to backend/.env and add your key")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/scan", cors(handleScan))
	mux.HandleFunc("/health", cors(handleHealth))

	addr := ":" + port
	log.Printf("ScamShield AI backend listening on http://localhost%s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
