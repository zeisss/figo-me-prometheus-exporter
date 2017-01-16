package figo

import (
	"net/url"
	"net/http"
	"io/ioutil"
)

func Obtain_login_url(clientID, scope string) (string, error) {
	params := url.Values{}
	params.Set("client_id", clientID)
	params.Set("response_type", "code")
	params.Set("scope", scope)
	params.Set("state", "no-state")

	req, err := http.NewRequest("GET",BaseURL + "/auth/code?" + params.Encode(), nil)
	if err != nil {
		return "", err
	}
	resp, err := Client.Do(req)
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

// CredentialsAuth authorized by username and password
func CredentialsAuth(username, password string, scope string) (string, error) {
	params := url.Values{}
	params.Set("grant_type", "password")
	params.Set("username", username)
	params.Set("password", password)
	params.Set("scope", scope)

	var response struct {
		AccessToken string `json:"access_token"`
	}

	err := do_authed_request_form("/auth/token", "", params, &response)
	return response.AccessToken, err
}