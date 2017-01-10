package main

import (
	"net/url"
	"strings"
	"net/http"
	"fmt"
	"io/ioutil"
	"encoding/json"
	"flag"
	"time"
	"io"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var client = http.Client{}

func init() {
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
}

func do_request(method, path string, accesscode string, body io.Reader, target interface{}) error {
	req, err := http.NewRequest(
		method,
		*baseURL + path,
		body,
	)
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", "Bearer " + accesscode)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	return json.Unmarshal(b, &target)
}

var (
	clientID = flag.String("client-id", "", "the client id to use for oauth")
	clientSecret = flag.String("client-secret","", "the client secret to use for oauth")
	scope = flag.String("scope", "accounts=ro balance=ro transactions=ro", "Scope to request access to")
	baseURL = flag.String("baseurl", "https://api.figo.me", "Base URL to talk to")

	addr = flag.String("addr", ":8080", "Address to listen on.")

	callbackurl = flag.String("callback", "", "Callback URL from login")
	token = flag.String("token", "", "Token to use for accessing figo")
)

func obtain_login_url() (string, error) {
	params := url.Values{}
	params.Set("client_id", *clientID)
	params.Set("response_type", "code")
	params.Set("scope", *scope)
	params.Set("state", "no-state")

	req, err := http.NewRequest("GET",*baseURL + "/auth/code?" + params.Encode(), nil)
	if err != nil {
		return "", err
	}
	fmt.Printf("Req: %v\n", req)
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	//fmt.Printf("%v", string(b))

	return resp.Header.Get("Location"), nil
}

func do_login() error {
	authURL, err := obtain_login_url()
	if err != nil {
		return err
	}

	fmt.Println("Please open the following URL and authenticate with FIGO to get an auth token")
	fmt.Println("")
	fmt.Println("\t" + authURL)
	fmt.Println("")
	fmt.Println("Next run 'go run auth.go -client-id=$CLIENTID -client-secret=$SECRET -callback=REDIRECTURL'")

	return nil
}

func do_get_auth_token() error {
	u, err := url.Parse(*callbackurl)
	if err != nil {
		return err
	}

	params := url.Values{}
	params.Set("grant_type", "authorization_code")
	params.Set("code", u.Query().Get("code"))

	req, err := http.NewRequest("POST", *baseURL + "/auth/token", strings.NewReader(params.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("content-type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(*clientID, *clientSecret)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	fmt.Printf("Response: %s\n", string(b))

	type respS struct {
		AccessToken string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	var r respS
	if err := json.Unmarshal(b, &r); err != nil {
		return err
	}

	fmt.Printf("Your AccessToken: %s\n", r.AccessToken)
	fmt.Printf("Your RefreshToken: %s\n", r.RefreshToken)

	return nil
}

type FIGOTransaction struct {
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

func get_transactions(accesscode string) ([]FIGOTransaction, error) {
	var response struct {
		Transactions []FIGOTransaction `json:"transactions"`
	}
	
	if err := do_request("GET", "/rest/transactions", accesscode, nil, &response); err != nil {
		return nil, err
	}
	return response.Transactions, nil
}

type FIGOSyncStatus struct {
	Code int `json:"code"`
	Message string `json:"message"`
	SyncTimestamp *time.Time `json:"sync_timestamp"`
	SuccessTimestamp *time.Time `json:"success_timestamp"`
}

type FIGOAccount struct {
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
	Status FIGOSyncStatus `json:"status"`
	Balance *FIGOBalance `json:"balance"`
}

type FIGOBalance struct {
	Balance float64 `json:"balance"`
	BalanceDate *time.Time `json:"balance_date"`
	CreditLine float64 `json:"credit_line"`
	MonthlySpendingLimit float64 `json:"monthy_spending_limit"`
	Status FIGOSyncStatus `json:"status"`
}


func get_accounts(accesscode string) ([]FIGOAccount, error) {
	var response struct {
		Accounts []FIGOAccount `json:"accounts"`
	}
	if err := do_request("GET", "/rest/accounts", accesscode, nil, &response); err != nil {
		return nil, err
	}
	return response.Accounts, nil
}

func do_run_transactions() error {
	transactions, err := get_transactions(*token)
	if err != nil {
		return err
	}

	transaction_amount := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "figo_transaction_amount", 
		Help: "Transaction amount",
		},
		[]string{"accountid", "type", "currency"},
	)
	prometheus.MustRegister(transaction_amount)

	for _, t := range transactions {
		transaction_amount.WithLabelValues(t.AccountID, t.Type, t.Currency).Add(t.Amount)
	}

	accounts, err := get_accounts(*token)
	if err != nil {
		return err
	}
	account_balance := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "figo_account_balance", 
		Help: "Account Balance",
		},
		[]string{"accountid", "name", "bankid", "type", "currency"},
	)
	account_sync_enabled := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "figo_account_sync_enabled", 
		Help: "Account Sync Enabled",
		},
		[]string{"accountid", "name", "bankid", "type"},
	)
	account_sync_status := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "figo_account_sync_status_error", 
		Help: "Account Sync Status",
		},
		[]string{"accountid", "name"},
	)
	
	prometheus.MustRegister(account_balance, account_sync_enabled, account_sync_status)
	
	for _, a := range accounts {
		var v float64
		if a.SyncEnabled {
			v = 1
		}
		account_sync_enabled.WithLabelValues(a.AccountID, a.Name, a.BankID, a.Type).Set(v)
		account_balance.WithLabelValues(a.AccountID, a.Name, a.BankID, a.Type, a.Currency).Add(a.Balance.Balance)

		if a.Status.Code != 1 {
			account_sync_status.WithLabelValues(a.AccountID, a.Name).Set(-1 * float64(a.Status.Code))
		} else {
			account_sync_status.WithLabelValues(a.AccountID, a.Name).Set(0)
		}
	}

	fmt.Println("Listening at " + *addr)
	http.Handle("/metrics", promhttp.Handler())

	return http.ListenAndServe(*addr, nil)
}

func main() {
	flag.Parse()

	var err error
	if *token != "" {
		err  = do_run_transactions()
	} else if *callbackurl != "" {
		err = do_get_auth_token()
	} else {
		err = do_login()
	}
	if err != nil {
		fmt.Printf("ERROR: %v", err)
	}
}

