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

Record audio in the frontend, send it to the backend, which transcribes it with a
self-hosted [whisper.cpp](https://github.com/ggml-org/whisper.cpp) server and
parses the text into a transaction.

- `POST /api/voice/transcribe` — STT + parse only, returns structured fields.
- `POST /api/voice/transaction` — STT + parse + **save** the transaction.

Both take multipart form-data with the audio in the `file` field (16 kHz mono WAV,
max 5 MB) and the usual auth headers (`Authorization`, `X-Spreadsheet-ID`,
`X-Sheet-Name`).

### Run the STT sidecar

```bash
make stt-model   # downloads models/ggml-base.bin (~142MB)
make stt-up      # starts the whisper-stt service on the npm-net network
```

Then set in `.env`:

```
STT_ENABLED=true
STT_URL=http://whisper-stt:8080/inference
STT_LANGUAGE=id
```

`STT_URL` is a full endpoint URL, so you can point it at any OpenAI-compatible
provider instead (e.g. Groq: `https://api.groq.com/openai/v1/audio/transcriptions`
with `STT_API_KEY`).

### Parser

The text → transaction parsing lives in `pkg/voice` (amounts, income/expense,
relative dates like "kemarin", priorities, notes) and is unit-tested:

```bash
go test ./pkg/voice/
```

