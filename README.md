# figo-me-prometheus-exporter

WIP!!!!!!

Export various things from your bank accounts at figo.me as prometheus metrics

Currently this just sums your account balances and transactions and exposes them.
Maybe more coming soon.

## Usage

Contact `developer@figo.me` and get a client_id + secret. Make sure they activate the `offline` scope for you.

Then run the auth.go file a few times (I left out the `-client-id` and `-client-secret` parameters, please add)
1. Run `go run auth.go`, open the printed URL in your browser and login. You will be redirected to a localhost URL. Copy that URL
2. Run `go run auth.go '-callback=$URL'`. This will transform your login token into a real access token. This is again printed out.
3. Run the server with `go run auth.go -token=$TOKEN` which will start scraping your accounts regularly and expose the metrics at `0.0.0.0:8080/metrics`.


## Status

There is no logic yet to utilize a refresh token and reread the latest balance. The current state is a one-time read. 