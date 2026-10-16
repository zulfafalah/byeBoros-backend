// Package voice parses free-form Indonesian (and simple English) spoken
// transaction phrases into structured fields that map onto the sheet schema.
package voice

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Type mirrors the transaction kind stored in the spreadsheet.
type Type string

const (
	TypeIncome  Type = "income"
	TypeExpense Type = "expense"
)

// Priority values match the canonical labels used by the analysis usecase.
const (
	PriorityHigh   = "high"
	PriorityMedium = "medium"
	PriorityLow    = "low"
)

// Parsed is the structured result of parsing a transcript.
type Parsed struct {
	Raw         string    `json:"raw"`
	Amount      int64     `json:"amount"`
	Description string    `json:"description"`
	Type        Type      `json:"type"`
	Priority    string    `json:"priority,omitempty"`
	Notes       string    `json:"notes,omitempty"`
	Date        time.Time `json:"date"`
	DateSet     bool      `json:"date_set"`
}

var units = map[string]int64{
	"nol": 0, "kosong": 0, "satu": 1, "dua": 2, "tiga": 3, "empat": 4,
	"lima": 5, "enam": 6, "tujuh": 7, "delapan": 8, "sembilan": 9,
	"sepuluh": 10, "sebelas": 11,
	"one": 1, "two": 2, "three": 3, "four": 4, "five": 5, "six": 6,
	"seven": 7, "eight": 8, "nine": 9, "ten": 10,
}

var scales = map[string]int64{
	"ribu": 1_000, "rb": 1_000, "k": 1_000, "rebu": 1_000,
	"juta": 1_000_000, "jt": 1_000_000,
	"milyar": 1_000_000_000, "miliar": 1_000_000_000,
	"thousand": 1_000, "million": 1_000_000,
}

var hundredWords = map[string]bool{"ratus": true, "hundred": true}

// fillers are dropped from the description.
var fillers = map[string]bool{
	"catat": true, "catet": true, "tambah": true, "tambahkan": true,
	"input": true, "masukkan": true, "tulis": true,
	"pengeluaran": true, "pemasukan": true, "pendapatan": true, "pemasukkan": true,
	"sebesar": true, "senilai": true, "seharga": true, "rupiah": true, "rp": true,
	"untuk": true, "aku": true, "saya": true, "ku": true, "yang": true,
	"expense": true, "income": true, "spend": true, "spent": true,
	"paid": true, "pay": true,
}

// verbs are action words dropped from the description (the noun is kept).
var verbs = map[string]bool{
	"beli": true, "bayar": true, "jajan": true, "belanja": true, "beliin": true,
	"terima": true, "dapat": true, "dapet": true, "gajian": true,
}

var incomeWords = map[string]bool{
	"pemasukan": true, "pendapatan": true, "pemasukkan": true, "gaji": true,
	"bonus": true, "terima": true, "dapat": true, "dapet": true, "masuk": true,
	"income": true, "salary": true, "dividen": true, "dividends": true, "freelance": true,
}

var expenseWords = map[string]bool{
	"pengeluaran": true, "belanja": true, "bayar": true, "beli": true,
	"jajan": true, "keluar": true, "habis": true, "expense": true,
	"spend": true, "spent": true, "paid": true, "pay": true,
}

var priorityWords = map[string]bool{
	"prioritas": true, "prioritasnya": true, "proiritas": true, "prioritas:": true,
}

var priorityHigh = map[string]bool{
	"tinggi": true, "mendesak": true, "urgent": true, "darurat": true,
	"penting": true, "high": true, "segera": true,
}

var priorityMedium = map[string]bool{
	"sedang": true, "medium": true, "normal": true, "biasa": true, "menengah": true,
}

var priorityLow = map[string]bool{
	"rendah": true, "low": true, "santai": true, "remeh": true,
}

var conjunctions = map[string]bool{
	"sama": true, "dengan": true, "bareng": true, "bersama": true,
}

var numericTokenRe = regexp.MustCompile(`^\d[\d.,]*$`)
var rpRe = regexp.MustCompile(`\brp\b`)
var rbRe = regexp.MustCompile(`(\d)\s*(rb|rebu)\b`)
var jtRe = regexp.MustCompile(`(\d)\s*(jt)\b`)
var kRe = regexp.MustCompile(`(\d)\s*k\b`)
var cleanRe = regexp.MustCompile(`[^a-z0-9.,\s]`)

func isNumericToken(tok string) bool { return numericTokenRe.MatchString(tok) }

func numericValue(tok string) int64 {
	cleaned := strings.NewReplacer(".", "", ",", "").Replace(tok)
	v, err := strconv.ParseInt(cleaned, 10, 64)
	if err != nil {
		return 0
	}
	return v
}

// normalize lowercases, expands numeric shorthand (50rb, 5jt, 10k) and splits
// into tokens, mirroring the original TypeScript implementation.
func normalize(transcript string) []string {
	s := strings.ToLower(transcript)
	s = rpRe.ReplaceAllString(s, " ")
	s = rbRe.ReplaceAllString(s, "$1 ribu")
	s = jtRe.ReplaceAllString(s, "$1 juta")
	s = kRe.ReplaceAllString(s, "$1 ribu")
	s = cleanRe.ReplaceAllString(s, " ")
	return strings.Fields(s)
}

func parseSegment(tokens []string) int64 {
	var value int64
	for i := 0; i < len(tokens); i++ {
		tok := tokens[i]
		if isNumericToken(tok) {
			value += numericValue(tok)
			continue
		}
		switch tok {
		case "seratus":
			value += 100
			continue
		case "seribu":
			value += 1_000
			continue
		case "sejuta":
			value += 1_000_000
			continue
		}
		if unit, ok := units[tok]; ok {
			var next string
			if i+1 < len(tokens) {
				next = tokens[i+1]
			}
			switch {
			case next == "ratus" || next == "hundred":
				value += unit * 100
				i++
			case next == "puluh":
				value += unit * 10
				i++
			case next == "belas":
				value += unit + 10
				i++
			default:
				value += unit
			}
			continue
		}
		if tok == "puluh" || tok == "belas" {
			value += 10
		} else if hundredWords[tok] {
			value += 100
		}
	}
	return value
}

// ParseAmount converts a token slice into a rupiah amount (0 when absent).
func ParseAmount(tokens []string) int64 {
	var total int64
	var group []string

	flush := func(scale int64) {
		groupValue := parseSegment(group)
		if scale > 0 {
			if groupValue == 0 {
				groupValue = 1
			}
			total += groupValue * scale
		} else {
			total += groupValue
		}
		group = nil
	}

	for _, tok := range tokens {
		if scale, ok := scales[tok]; ok && scale >= 1_000 {
			flush(scale)
		} else {
			group = append(group, tok)
		}
	}
	flush(0)
	return total
}

func isNumberWord(tok string) bool {
	if isNumericToken(tok) {
		return true
	}
	if _, ok := units[tok]; ok {
		return true
	}
	if _, ok := scales[tok]; ok {
		return true
	}
	return hundredWords[tok] || tok == "puluh" || tok == "belas" ||
		tok == "seratus" || tok == "seribu" || tok == "sejuta"
}

// detectDate scans tokens for relative date phrases and returns the target date
// plus the set of consumed token indices.
func detectDate(tokens []string, consumed map[int]bool, now time.Time) (time.Time, bool) {
	target := now
	found := false

	for i := 0; i < len(tokens); i++ {
		cur := tokens[i]
		next := ""
		if i+1 < len(tokens) {
			next = tokens[i+1]
		}
		switch cur {
		case "kemarin", "kemaren", "kmrn":
			target = now.AddDate(0, 0, -1)
			consumed[i] = true
			found = true
			if next == "lusa" || next == "dulu" {
				target = now.AddDate(0, 0, -2)
				consumed[i+1] = true
				i++
			}
		case "lusa":
			target = now.AddDate(0, 0, 2)
			consumed[i] = true
			found = true
		case "besok", "esok":
			target = now.AddDate(0, 0, 1)
			consumed[i] = true
			found = true
		case "hari":
			if next == "ini" {
				target = now
				consumed[i], consumed[i+1] = true, true
				found = true
				i++
			}
		case "tadi":
			target = now
			consumed[i] = true
			found = true
		case "minggu":
			if next == "lalu" || next == "kemarin" || next == "kemaren" {
				target = now.AddDate(0, 0, -7)
				consumed[i], consumed[i+1] = true, true
				found = true
				i++
			}
		case "bulan":
			if next == "lalu" || next == "kemarin" || next == "kemaren" {
				target = now.AddDate(0, -1, 0)
				consumed[i], consumed[i+1] = true, true
				found = true
				i++
			}
		}
	}

	return target, found
}

func detectPriority(tokens []string, consumed map[int]bool) string {
	for i, tok := range tokens {
		if priorityHigh[tok] {
			consumed[i] = true
			return PriorityHigh
		}
		if priorityMedium[tok] {
			consumed[i] = true
			return PriorityMedium
		}
		if priorityLow[tok] {
			consumed[i] = true
			return PriorityLow
		}
	}
	return ""
}

// detectNotes captures a trailing "sama/dengan/bareng <name>" phrase and marks
// the consumed indices.
func detectNotes(tokens []string, consumed map[int]bool) string {
	start := -1
	for i, tok := range tokens {
		if conjunctions[tok] {
			start = i
		}
	}
	if start < 0 || start == len(tokens)-1 {
		return ""
	}

	var parts []string
	consumed[start] = true
	for j := start + 1; j < len(tokens); j++ {
		tok := tokens[j]
		if isNumberWord(tok) || fillers[tok] || verbs[tok] {
			continue
		}
		if priorityHigh[tok] || priorityMedium[tok] || priorityLow[tok] || priorityWords[tok] {
			continue
		}
		consumed[j] = true
		parts = append(parts, tok)
	}
	return strings.Join(parts, " ")
}

// Parse converts a transcript into structured transaction fields. The now
// argument is the reference point for relative dates.
func Parse(transcript string, now time.Time) Parsed {
	if now.IsZero() {
		now = time.Now()
	}

	tokens := normalize(transcript)
	if len(tokens) == 0 {
		return Parsed{Raw: transcript}
	}

	result := Parsed{Raw: transcript}
	result.Amount = ParseAmount(tokens)

	for _, tok := range tokens {
		if incomeWords[tok] {
			result.Type = TypeIncome
			break
		}
	}
	if result.Type == "" {
		for _, tok := range tokens {
			if expenseWords[tok] {
				result.Type = TypeExpense
				break
			}
		}
	}

	consumed := map[int]bool{}
	if date, ok := detectDate(tokens, consumed, now); ok {
		result.Date = date
		result.DateSet = true
	}
	result.Priority = detectPriority(tokens, consumed)
	result.Notes = detectNotes(tokens, consumed)

	var desc []string
	for i, tok := range tokens {
		if consumed[i] {
			continue
		}
		if isNumberWord(tok) || fillers[tok] || verbs[tok] || priorityWords[tok] {
			continue
		}
		desc = append(desc, tok)
	}
	result.Description = strings.TrimSpace(strings.Join(desc, " "))

	return result
}

// MatchCategory scores description tokens against candidate category and
// sub-category names and returns the best match, if any.
func MatchCategory(description string, candidates []string) (string, bool) {
	descTokens := normalize(description)
	if len(descTokens) == 0 {
		return "", false
	}

	best := ""
	bestScore := 0
	for _, candidate := range candidates {
		if strings.TrimSpace(candidate) == "" {
			continue
		}
		score := 0
		for _, catTok := range normalize(candidate) {
			catTok = strings.TrimSuffix(catTok, "s")
			for _, descTok := range descTokens {
				if tokensRelated(descTok, catTok) {
					score++
					break
				}
			}
		}
		if score > bestScore {
			bestScore = score
			best = candidate
		}
	}
	return best, bestScore > 0
}

// tokensRelated reports whether two tokens are equal or share a common prefix
// of at least four characters (handles makan/makanan, tagih/tagihan).
func tokensRelated(a, b string) bool {
	if a == b {
		return true
	}
	const minPrefix = 4
	if len(a) < minPrefix || len(b) < minPrefix {
		return false
	}
	n := minPrefix
	if len(a) < n {
		n = len(a)
	}
	if len(b) < n {
		n = len(b)
	}
	return a[:n] == b[:n]
}
