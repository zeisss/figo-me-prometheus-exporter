package main

import (
	"net/http"
	"fmt"
	"flag"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	. "./figo"
)

var (
	clientID = flag.String("client-id", "", "the client id to use for oauth")
	clientSecret = flag.String("client-secret","", "the client secret to use for oauth")
	scope = flag.String("scope", "accounts=ro balance=ro transactions=ro", "Scope to request access to")
	baseURL = flag.String("baseurl", "https://api.figo.me", "Base URL to talk to")

	addr = flag.String("addr", ":8080", "Address to listen on.")

	username = flag.String("user", "", "Username to login with figo.me")
	password = flag.String("pw", "", "Password to login with figo.me")
)

func do_collect_metrics_loop(token string) error {
	transactions, err := GetTransactions(token)
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

	accounts, err := GetAccounts(token)
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

func do_collect_metrics() error {
	token, err := CredentialsAuth(*username, *password, *scope)
	if err != nil {
		return err
	}
	fmt.Println("Token: ", token)
	return do_collect_metrics_loop(token)
}

func main() {
	flag.Parse()
	BaseURL = *baseURL
	ClientID = *clientID
	ClientSecret = *clientSecret

	var err error
	err = do_collect_metrics()
	
	if err != nil {
		fmt.Printf("ERROR: %v", err)
	}
}

