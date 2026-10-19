package usecase

import (
	"testing"

	"byeboros-backend/internal/adapter/http/model/request"
)

func TestDeleteTransactionValidation(t *testing.T) {
	// The repository is only touched after validation, so a nil repo is safe here.
	u := NewTransactionUsecase(nil)

	cases := []struct {
		name string
		req  request.DeleteTransactionRequest
	}{
		{"unknown id prefix", request.DeleteTransactionRequest{ID: "foo_1", Type: "expense"}},
		{"non numeric id", request.DeleteTransactionRequest{ID: "txn_exp_abc", Type: "expense"}},
		{"type mismatch", request.DeleteTransactionRequest{ID: "txn_exp_1", Type: "income"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := u.DeleteTransaction("sheet-id", "Januari", tc.req); err == nil {
				t.Fatalf("expected error for %+v, got nil", tc.req)
			}
		})
	}
}
