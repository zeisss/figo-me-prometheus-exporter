package figo

import (
	"net/http"
	"fmt"
	"net/url"
	"strings"

	"io"
	"io/ioutil"
	"encoding/json"
)

var (
	UnauthorizedErr = fmt.Errorf("Request was not authorized")
)

var Client = http.Client{
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

var BaseURL = "https://api.figo.me"

var (
	ClientID string
	ClientSecret string
)


func do_authed_request_form(path, token string, body url.Values, target interface{}) error {
	return do_request("POST", path, token, 
		"application/x-www-form-urlencoded", strings.NewReader(body.Encode()),
		true, target,
	)
}

func do_request_get(path, token string, target interface{}) error {
	return do_request("GET", path, token, "", nil, false, &target)
}

func do_request(method, path, accesscode, contentType string, body io.Reader, authHeader bool, target interface{}) error {
	req, err := http.NewRequest(
		method,
		BaseURL + path,
		body,
	)
	if err != nil {
		return err
	}
	if accesscode != "" {
		req.Header.Add("Authorization", "Bearer " + accesscode)
	}
	if contentType != "" {
		req.Header.Add("content-type", contentType)
	}
	if authHeader {
		req.SetBasicAuth(ClientID, ClientSecret)
	}
	resp, err := Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return UnauthorizedErr
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	// fmt.Printf("%#v\n", string(b))

	if resp.StatusCode >= 400 {
		var apiErr Error
		if err := json.Unmarshal(b, &apiErr); err != nil {
			return err
		}
		return apiErr
	}

	return json.Unmarshal(b, &target)
}

func GetAccounts(accesscode string) ([]Account, error) {
	var response struct {
		Accounts []Account `json:"accounts"`
	}
	if err := do_request_get("/rest/accounts", accesscode, &response); err != nil {
		return nil, err
	}
	return response.Accounts, nil
}


func GetTransactions(accesscode string) ([]Transaction, error) {
	var response struct {
		Transactions []Transaction `json:"transactions"`
	}
	
	if err := do_request_get("/rest/transactions", accesscode, &response); err != nil {
		return nil, err
	}
	return response.Transactions, nil
}

