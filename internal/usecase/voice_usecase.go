package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"byeboros-backend/internal/adapter/http/model/request"
	"byeboros-backend/internal/adapter/http/model/response"
	"byeboros-backend/internal/infrastructure/llm"
	"byeboros-backend/internal/infrastructure/stt"
	"byeboros-backend/pkg/voice"
)

// transactionAtLayout matches the d/MM/yyyy H:mm:ss format the sheet expects.
const transactionAtLayout = "2/1/2006 15:4:5"

// VoiceUsecase transcribes audio and turns it into a transaction.
type VoiceUsecase struct {
	enabled            bool
	sttClient          *stt.Client
	analyzer           *llm.CategoryClassifier
	categoryUsecase    *CategoryUsecase
	transactionUsecase *TransactionUsecase
}

// NewVoiceUsecase wires the voice feature dependencies. analyzer may be nil.
func NewVoiceUsecase(enabled bool, sttClient *stt.Client, analyzer *llm.CategoryClassifier, categoryUsecase *CategoryUsecase, transactionUsecase *TransactionUsecase) *VoiceUsecase {
	return &VoiceUsecase{
		enabled:            enabled,
		sttClient:          sttClient,
		analyzer:           analyzer,
		categoryUsecase:    categoryUsecase,
		transactionUsecase: transactionUsecase,
	}
}

// Enabled reports whether the STT backend is configured.
func (u *VoiceUsecase) Enabled() bool { return u.enabled && u.sttClient != nil }

// VoiceValidationError signals a recoverable problem with the parsed result
// (e.g. a missing amount or an unresolvable category) so the HTTP layer can
// return a helpful 422 instead of a 500.
type VoiceValidationError struct {
	Message    string
	Candidates []string
}

func (e *VoiceValidationError) Error() string { return e.Message }

// Transcribe runs STT and parses the resulting text.
func (u *VoiceUsecase) Transcribe(ctx context.Context, filename string, audio []byte) (voice.Parsed, string, error) {
	if !u.Enabled() {
		return voice.Parsed{}, "", fmt.Errorf("voice transcription is not enabled")
	}
	text, err := u.sttClient.Transcribe(ctx, filename, audio)
	if err != nil {
		return voice.Parsed{}, "", err
	}
	parsed := voice.Parse(text, time.Now())
	return parsed, text, nil
}

// Enrich fills in the category, title, priority and type of a parsed
// transaction. When the LLM analyzer is configured it rewrites the description
// into a clean title and picks the category/priority/type from the transcript;
// any field it cannot resolve falls back to the keyword parser.
func (u *VoiceUsecase) Enrich(ctx context.Context, spreadsheetID, sheetName string, parsed *voice.Parsed) (string, []string, error) {
	var candidates []string
	if parsed.Type == voice.TypeIncome {
		cats, err := u.categoryUsecase.GetIncomeCategory(spreadsheetID, sheetName)
		if err != nil {
			return "", nil, err
		}
		candidates = cats
	} else {
		res, err := u.categoryUsecase.GetCategory(spreadsheetID, sheetName)
		if err != nil {
			return "", nil, err
		}
		candidates = make([]string, 0, len(res.Categories)*2)
		for _, c := range res.Categories {
			if c.SubCategoryName != "" {
				candidates = append(candidates, c.SubCategoryName)
			}
			if c.CategoryName != "" {
				candidates = append(candidates, c.CategoryName)
			}
		}
	}

	if u.analyzer != nil && u.analyzer.Enabled() {
		if analysis, err := u.analyzer.Analyze(ctx, parsed.Raw, parsed.Description, string(parsed.Type), candidates); err == nil {
			if analysis.Title != "" {
				parsed.Description = analysis.Title
			}
			if analysis.Priority != "" {
				parsed.Priority = analysis.Priority
			}
			if parsed.Type == "" && analysis.Type != "" {
				parsed.Type = voice.Type(analysis.Type)
			}
			if analysis.Category != "" {
				return analysis.Category, candidates, nil
			}
		}
	}

	match, _ := voice.MatchCategory(parsed.Description, candidates)
	return match, candidates, nil
}

// toResponse maps a parsed result into the API response shape.
func toResponse(parsed voice.Parsed, text, category string) response.VoiceTransactionResponse {
	resp := response.VoiceTransactionResponse{
		Text:        text,
		Amount:      parsed.Amount,
		Description: parsed.Description,
		Type:        string(parsed.Type),
		Priority:    parsed.Priority,
		Notes:       parsed.Notes,
		Category:    category,
	}
	if parsed.DateSet {
		resp.Date = parsed.Date.Format("2006-01-02")
	}
	return resp
}

// BuildResponse maps a parsed result into the API response shape.
func (u *VoiceUsecase) BuildResponse(parsed voice.Parsed, text, category string) response.VoiceTransactionResponse {
	return toResponse(parsed, text, category)
}

// CreateFromVoice parses and persists a transaction, returning the stored row.
func (u *VoiceUsecase) CreateFromVoice(ctx context.Context, spreadsheetID, sheetName, createdBy string, filename string, audio []byte) (response.VoiceTransactionResponse, error) {
	parsed, text, err := u.Transcribe(ctx, filename, audio)
	if err != nil {
		return response.VoiceTransactionResponse{}, err
	}

	if parsed.Amount <= 0 {
		return toResponse(parsed, text, ""), &VoiceValidationError{Message: "nominal tidak terdeteksi dari ucapan"}
	}
	if parsed.Type == "" {
		parsed.Type = voice.TypeExpense
	}

	category, candidates, err := u.Enrich(ctx, spreadsheetID, sheetName, &parsed)
	if err != nil {
		return toResponse(parsed, text, ""), err
	}
	if parsed.Description == "" {
		return toResponse(parsed, text, ""), &VoiceValidationError{Message: "deskripsi tidak terdeteksi dari ucapan"}
	}
	if category == "" {
		return toResponse(parsed, text, ""), &VoiceValidationError{
			Message:    "kategori tidak cocok, pilih manual",
			Candidates: candidates,
		}
	}

	transactionAt := time.Now()
	if parsed.DateSet {
		transactionAt = parsed.Date
	}
	transactionAtStr := transactionAt.Format(transactionAtLayout)

	notes := strings.TrimSpace(parsed.Notes)
	var notesPtr *string
	if notes != "" {
		notesPtr = &notes
	}

	if parsed.Type == voice.TypeIncome {
		req := request.IncomeTransactionRequest{
			Description:   parsed.Description,
			Category:      category,
			Amount:        float64(parsed.Amount),
			Notes:         notesPtr,
			TransactionAt: transactionAtStr,
		}
		if err := u.transactionUsecase.AddIncomeTransaction(spreadsheetID, sheetName, req, createdBy); err != nil {
			return response.VoiceTransactionResponse{}, err
		}
	} else {
		priority := parsed.Priority
		if priority == "" {
			priority = voice.PriorityMedium
		}
		req := request.ExpenseTransactionRequest{
			Description:   parsed.Description,
			Category:      category,
			Priority:      priority,
			Amount:        float64(parsed.Amount),
			Notes:         notesPtr,
			TransactionAt: transactionAtStr,
		}
		if err := u.transactionUsecase.AddExpenseTransaction(spreadsheetID, sheetName, req, createdBy); err != nil {
			return response.VoiceTransactionResponse{}, err
		}
	}

	resp := toResponse(parsed, text, category)
	resp.Created = true
	resp.Message = "transaksi berhasil disimpan"
	return resp, nil
}
