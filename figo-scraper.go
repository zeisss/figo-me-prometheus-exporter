package main

import (
	"net/http"
	"time"
	"fmt"
	"log"
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

var (
	transaction_amount = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "figo_transaction_amount", 
		Help: "Transaction amount",
		},
		[]string{"accountid", "type", "currency"},
	)

	account_balance = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "figo_account_balance", 
		Help: "Account Balance",
		},
		[]string{"accountid", "name", "bankid", "type", "currency"},
	)
	account_sync_enabled = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "figo_account_sync_enabled", 
		Help: "Account Sync Enabled",
		},
		[]string{"accountid", "name", "bankid", "type"},
	)
	account_sync_status = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "figo_account_sync_status_error", 
		Help: "Account Sync Status",
		},
		[]string{"accountid", "name"},
	)

	scraping_errors = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "figo_scrape_errors",
		Help: "Number of failed scrapes",
	})

	scraping_duration = prometheus.NewSummary(prometheus.SummaryOpts{
		Name: "figo_scrape_duration_seconds",
		Help: "Duration for scraping all needed data from Figo",
	})
)
func init() {
	prometheus.MustRegister(
		transaction_amount, account_balance, account_sync_enabled, account_sync_status,
		scraping_errors, scraping_duration,
	)
}

func do_collect_metrics_loop(token string) error {
	transactions, err := GetTransactions(token)
	if err != nil {
		return err
	}

	for _, t := range transactions {
		transaction_amount.WithLabelValues(t.AccountID, t.Type, t.Currency).Add(t.Amount)
	}

	accounts, err := GetAccounts(token)
	if err != nil {
		return err
	}

	for _, a := range accounts {
		var v float64
		if a.SyncEnabled {
			v = 1
		}
		account_sync_enabled.WithLabelValues(a.AccountID, a.Name, a.BankID, a.Type).Set(v)
		account_balance.WithLabelValues(a.AccountID, a.Name, a.BankID, a.Type, a.Currency).Set(a.Balance.Balance)

		if a.Status.Code != 1 {
			account_sync_status.WithLabelValues(a.AccountID, a.Name).Set(-1 * float64(a.Status.Code))
		} else {
			account_sync_status.WithLabelValues(a.AccountID, a.Name).Set(0)
		}
	}
	return nil
}

func do_collect_metrics_wrapper(token string) error {
	start := time.Now()
	err := do_collect_metrics_loop(token)
	dur := time.Since(start)
	scraping_duration.Observe(dur.Seconds())
	return err
}

func do_collect_metrics() <-chan error {
	var token string
	
	reauth := func() error {
		new_token, err := CredentialsAuth(*username, *password, *scope)
		if err != nil {
			return err
		}
		log.Printf("authenticated successful - token=%s\n", new_token)

		token = new_token
		return nil
	}

	do_collect := func() error {
		err := do_collect_metrics_wrapper(token)
		if err != nil {
			if err == UnauthorizedErr {
				if err := reauth(); err != nil {
					return err
				}
			} else {
				scraping_errors.Inc()
				log.Printf("ERROR: scraping failed: %v\n", err)
			}
		}
		return nil
	}
	
	errChan := make(chan error, 1)
	if err := reauth(); err != nil {
		errChan <- err
	}
	go func() {
		if err := do_collect(); err != nil {
			errChan <- err
		}
		t := time.Tick(5 * time.Minute)
		for range t {
			if err := do_collect(); err != nil {
				errChan <- err
			}
		}
	}()
	
	return errChan
}

var tmpl = `
<html>
	<head><title>Figo Prometheus Scraper</title></head>
	<body>
		<h1>Figo Prometheus Scraper</h1>
		<ol>
			<li><a href="/metrics">Metrics</a></li>
			<li><a href="https://home.figo.me/">Figo Home</a></li>
	</body>
</html>
`

func landingpage_handler(resp http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(resp, tmpl)
}

func main() {
	flag.Parse()
	BaseURL = *baseURL
	ClientID = *clientID
	ClientSecret = *clientSecret

	errChan := do_collect_metrics()

	go func() {
		select {
		case err := <-errChan:
			log.Fatalf("ERROR: failed to collect metrics: %v", err)
		}
	}()

	log.Println("Listening at " + *addr)
	http.HandleFunc("/", landingpage_handler)
	http.Handle("/metrics", promhttp.Handler())

	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatalf("ERROR: %v", err)
	}
}

