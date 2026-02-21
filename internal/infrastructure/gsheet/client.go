package gsheet

import (
	"context"
	"fmt"
	"log"

	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

// Client wraps the Google Sheets service
type Client struct {
	Service       *sheets.Service
	SpreadsheetID string
}

// NewClient creates a new Google Sheets client using a service account JSON file
func NewClient(serviceAccountFile, spreadsheetID string) (*Client, error) {
	ctx := context.Background()

	srv, err := sheets.NewService(ctx,
		option.WithCredentialsFile(serviceAccountFile),
		option.WithScopes(
			sheets.SpreadsheetsScope,
			sheets.DriveScope,
		),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to create sheets service: %w", err)
	}

	log.Println("✅ Google Sheets client initialized successfully")

	return &Client{
		Service:       srv,
		SpreadsheetID: spreadsheetID,
	}, nil
}
