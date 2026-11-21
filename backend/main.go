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

// primaryModel is used first for every scan. If it times out or the service
// answers with a 5xx error, fallbackModel is tried automatically.
const (
	primaryModel  = "gemini-3.6-flash"
	fallbackModel = "gemini-3.5-flash"
)

// errUpstream marks a retryable server side (5xx) failure from Gemini.
var errUpstream = errors.New("gemini upstream error")

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
	w.Header().Set("Content-Type", "application/json")
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

func buildSystemPrompt(language string) string {
	target := "English"
	if strings.EqualFold(strings.TrimSpace(language), "Sinhala") {
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

// ---------------------------------------------------------------- gemini

// analyzeWithGemini asks the primary model first and retries once with the
// fallback model when the primary times out or returns a 5xx error.
func analyzeWithGemini(apiKey, text, language string) (*scanResponse, error) {
	report, err := callGemini(apiKey, primaryModel, text, language)
	if err == nil {
		return report, nil
	}

	if !isRetryable(err) {
		return nil, err
	}

	log.Printf("%s failed (%v) - retrying with %s", primaryModel, err, fallbackModel)
	return callGemini(apiKey, fallbackModel, text, language)
}

// isRetryable reports whether the error is a timeout or a 5xx response, i.e.
// the kind of failure that is worth retrying on another model.
func isRetryable(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, errUpstream) || errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	return false
}

func callGemini(apiKey, model, text, language string) (*scanResponse, error) {
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

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gemini request failed: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read gemini response: %w", err)
	}

	// Server side failures are retryable, so flag them before parsing.
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
		writeError(w, http.StatusInternalServerError, "GEMINI_API_KEY is not configured on the server")
		return
	}

	report, err := analyzeWithGemini(apiKey, req.Text, req.Language)
	if err != nil {
		log.Printf("scan failed: %v", err)
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, report)
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
