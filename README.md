# Byeboros Backend

Go backend service using Echo framework with Google OAuth login and Google Sheets API as data store.

## Tech Stack

- **Go** + **Echo** framework
- **Google OAuth 2.0** for authentication
- **Google Sheets API** as data storage (no traditional database)
- **JWT** for session management

## Project Structure

```
├── cmd/web/              # Application entry point
├── config/               # Configuration loader
├── internal/
│   ├── adapter/
│   │   ├── http/
│   │   │   ├── controller/   # HTTP handlers
│   │   │   ├── middleware/    # JWT middleware
│   │   │   └── model/        # Request/Response models
│   │   └── repository/       # Google Sheets CRUD
│   ├── domain/model/         # Domain models
│   ├── infrastructure/
│   │   └── gsheet/           # Google Sheets client
│   └── usecase/              # Business logic
└── pkg/                      # Shared packages
```

## Setup

1. Copy environment variables:
   ```bash
   cp .env.example .env
   ```

2. Configure `.env` with your Google OAuth credentials and service account.

3. Place your Google service account JSON file in the project root.

4. Run the server:
   ```bash
   make run
   ```

## Voice transaction entry (STT)

Record audio in the frontend, send it to the backend, which transcribes it with
an OpenAI-compatible speech-to-text provider and parses the text into a
transaction.

- `POST /api/voice/transcribe` — STT + parse only, returns structured fields.
- `POST /api/voice/transaction` — STT + parse + **save** the transaction.

Both take multipart form-data with the audio in the `file` field (16 kHz mono WAV,
max 5 MB) and the usual auth headers (`Authorization`, `X-Spreadsheet-ID`,
`X-Sheet-Name`).

### Provider

`STT_URL` is a full endpoint URL, so any OpenAI-compatible provider works.

**Option A — Groq cloud (recommended, no local compute).** Fast and accurate
(`whisper-large-v3-turbo`, ~12% WER on Indonesian):

```
STT_ENABLED=true
STT_URL=https://api.groq.com/openai/v1/audio/transcriptions
STT_API_KEY=your-groq-api-key
STT_MODEL=whisper-large-v3-turbo
STT_LANGUAGE=id
STT_BEAM_SIZE=0            # Groq rejects beam_size
```

**Option B — self-hosted whisper.cpp.** Runs entirely on your machine; see
below. Set `STT_MODEL=` empty and `STT_BEAM_SIZE=5`.

```
STT_ENABLED=true
STT_URL=http://whisper-stt:8080/inference
STT_LANGUAGE=id
STT_BEAM_SIZE=5
STT_PROMPT=Catatan keuangan berbahasa Indonesia. Contoh nominal: sepuluh ribu, lima puluh ribu.
```

### Run the STT sidecar (self-hosted)

The `whisper-stt` service is built from source (`Dockerfile.stt`) so it works on
arm64 as well as amd64 (the published whisper.cpp image is amd64-only).

```bash
make stt-model   # downloads models/ggml-large-v3-turbo-q5_0.bin (~547MB)
make stt-up      # builds + starts the whisper-stt service on the npm-net network
```

#### Model choice

Accuracy on Indonesian (word error rate, lower is better; measured on a FLEURS
sample):

| model | WER | download | notes |
| --- | --- | --- | --- |
| `ggml-base.bin` | ~29% | 141 MB | fastest, but mis-parses numbers/words |
| `ggml-small.bin` | ~17% | 465 MB | good balance |
| `ggml-medium.bin` | ~16% | 1.5 GB | little gain over small |
| `ggml-large-v3-turbo-q5_0.bin` | ~12% | 547 MB | **default**, best accuracy |

Set `STT_MODEL_FILE` in `.env` and re-run `make stt-model stt-up`. Lighter
shortcuts: `make stt-model-small` / `make stt-model-base`.

### Category classification (LLM)

After transcription, the backend picks the transaction category. If `LLM_ENABLED`
is set, it asks an OpenAI-compatible chat model to choose one of the user's
categories from the spoken description (much better than keyword matching for
phrases like "naik ojek"). On any error, timeout or invalid answer it falls back
to the keyword matcher in `pkg/voice`.

```
LLM_ENABLED=true
LLM_URL=https://api.groq.com/openai/v1/chat/completions
LLM_API_KEY=            # empty falls back to STT_API_KEY
LLM_MODEL=openai/gpt-oss-20b
LLM_TIMEOUT_SECONDS=15
```

Any OpenAI-compatible `/chat/completions` endpoint works (Groq, OpenCode Zen,
OpenAI, local vLLM, ...). Note that OpenCode Zen's `-free` models are restricted
to the OpenCode app and cannot be called from a server.

### Parser

The text → transaction parsing lives in `pkg/voice` (amounts, income/expense,
relative dates like "kemarin", priorities, notes) and is unit-tested:

```bash
go test ./pkg/voice/
```

