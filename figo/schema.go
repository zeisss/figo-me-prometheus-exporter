package figo

import (
	"encoding/json"
	"fmt"
	"time"
)

type Error struct {
	Status int `json:"status"`
	ErrorDetails struct {
		Code int `json:"code"`
		Group string `json:"group"`
		Name string `json:"name"`
		Message string `json:"message"`
		Data json.RawMessage `json:"data"`
		Description string `json:"description"`
	} `json:"error"`
}

func (err Error) Error() string {
	return fmt.Sprintf("%s: %s (%d)", err.ErrorDetails.Name, err.ErrorDetails.Message, err.ErrorDetails.Code)
}

type Transaction struct {
	// Figo
	AccountID string `json:"account_id"`
	TransactionID string `json:"transaction_id"`
	
	// Transaction
	Purpose string `json:"purpose"`
	BookingDate *time.Time `json:"booking_date"`
	Name string `json:"name"`
	Amount float64 `json:"amount"`
	Currency string `json:"currency"`
	AccountNumber string `json:"account_number"`
	Type string `json:"type"`
	BookingText string `json:"booking_text"`
	BankCode string `json:"bank_code"`
	BankName string `json:"bank_name"`
}


type SyncStatus struct {
	Code int `json:"code"`
	Message string `json:"message"`
	SyncTimestamp *time.Time `json:"sync_timestamp"`
	SuccessTimestamp *time.Time `json:"success_timestamp"`
}

type Account struct {
	// Figo
	AccountID string `json:"account_id"`
	BankID string `json:"bank_id"`

	// Account
	Name string `json:"name"`
	Owner string `json:"owner"`
	AccountNumber string `json:"account_number"`
	BankCode string `json:"bank_code"`
	BankName string `json:"bank_name"`
	Currency string `json:"currency"`
	IBAN string `json:"iban"`
	BIC string `json:"bic"`
	Type string `json:"type"`
	SyncEnabled bool `json:"sync_enabled"`

	InTotalBalance bool `json:"in_total_balance"`
	SavePin bool `json:"save_pin"`
	Status SyncStatus `json:"status"`
	Balance *Balance `json:"balance"`
}

type Balance struct {
	Balance float64 `json:"balance"`
	BalanceDate *time.Time `json:"balance_date"`
	CreditLine float64 `json:"credit_line"`
	MonthlySpendingLimit float64 `json:"monthy_spending_limit"`
	Status SyncStatus `json:"status"`
}