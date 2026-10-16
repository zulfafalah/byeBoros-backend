package controller

import (
	"errors"
	"io"
	"net/http"

	"byeboros-backend/internal/usecase"

	"github.com/labstack/echo/v4"
)

const maxVoiceAudioBytes = 5 << 20 // 5 MiB

// VoiceController handles voice transcription endpoints.
type VoiceController struct {
	voiceUsecase *usecase.VoiceUsecase
}

// NewVoiceController creates a new VoiceController.
func NewVoiceController(voiceUsecase *usecase.VoiceUsecase) *VoiceController {
	return &VoiceController{voiceUsecase: voiceUsecase}
}

func contextualIDs(c echo.Context) (spreadsheetID, sheetName string, ok bool) {
	spreadsheetID, _ = c.Get("spreadsheet_id").(string)
	sheetName, _ = c.Get("sheet_name").(string)
	return spreadsheetID, sheetName, spreadsheetID != "" && sheetName != ""
}

func readAudio(c echo.Context) (string, []byte, error) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		fileHeader, err = c.FormFile("audio")
	}
	if err != nil {
		return "", nil, echo.NewHTTPError(http.StatusBadRequest, "missing audio file (field 'file')")
	}
	if fileHeader.Size > maxVoiceAudioBytes {
		return "", nil, echo.NewHTTPError(http.StatusRequestEntityTooLarge, "audio file too large (max 5MB)")
	}

	src, err := fileHeader.Open()
	if err != nil {
		return "", nil, echo.NewHTTPError(http.StatusBadRequest, "cannot open audio file")
	}
	defer src.Close()

	data, err := io.ReadAll(io.LimitReader(src, maxVoiceAudioBytes+1))
	if err != nil {
		return "", nil, echo.NewHTTPError(http.StatusBadRequest, "cannot read audio file")
	}
	if len(data) > maxVoiceAudioBytes {
		return "", nil, echo.NewHTTPError(http.StatusRequestEntityTooLarge, "audio file too large (max 5MB)")
	}
	return fileHeader.Filename, data, nil
}

// Transcribe handles POST /api/voice/transcribe — STT + parse, no persistence.
func (h *VoiceController) Transcribe(c echo.Context) error {
	if !h.voiceUsecase.Enabled() {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"error": "voice transcription is not enabled on this server",
		})
	}

	spreadsheetID, sheetName, ok := contextualIDs(c)
	if !ok {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing spreadsheet context"})
	}

	filename, audio, err := readAudio(c)
	if err != nil {
		return err
	}

	parsed, text, err := h.voiceUsecase.Transcribe(c.Request().Context(), filename, audio)
	if err != nil {
		return c.JSON(http.StatusBadGateway, map[string]string{"error": "transcription failed: " + err.Error()})
	}

	category, _, err := h.voiceUsecase.ResolveCategory(spreadsheetID, sheetName, parsed)
	if err != nil {
		category = ""
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status": "success",
		"data":   h.voiceUsecase.BuildResponse(parsed, text, category),
	})
}

// CreateTransaction handles POST /api/voice/transaction — STT + parse + save.
func (h *VoiceController) CreateTransaction(c echo.Context) error {
	if !h.voiceUsecase.Enabled() {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"error": "voice transcription is not enabled on this server",
		})
	}

	spreadsheetID, sheetName, ok := contextualIDs(c)
	if !ok {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing spreadsheet context"})
	}

	filename, audio, err := readAudio(c)
	if err != nil {
		return err
	}

	createdBy, _ := c.Get("email").(string)

	resp, err := h.voiceUsecase.CreateFromVoice(c.Request().Context(), spreadsheetID, sheetName, createdBy, filename, audio)
	if err != nil {
		var validationErr *usecase.VoiceValidationError
		if errors.As(err, &validationErr) {
			return c.JSON(http.StatusUnprocessableEntity, map[string]interface{}{
				"error":      validationErr.Message,
				"candidates": validationErr.Candidates,
				"data":       resp,
			})
		}
		return c.JSON(http.StatusBadGateway, map[string]string{"error": "voice transaction failed: " + err.Error()})
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"status": "success",
		"data":   resp,
	})
}
