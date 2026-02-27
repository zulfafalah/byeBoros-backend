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
