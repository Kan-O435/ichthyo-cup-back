package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

// authRequest performs a POST request to the specified URL with username and password.
func authRequest(url, username, password string, onSuccess, onError func(string)) {
	go func() {
		requestBody, err := json.Marshal(map[string]string{
			"username": username,
			"password": password,
		})
		if err != nil {
			if onError != nil {
				onError("Failed to create request body: " + err.Error())
			}
			return
		}

		resp, err := http.Post(url, "application/json", bytes.NewBuffer(requestBody))
		if err != nil {
			if onError != nil {
				onError("Request failed: " + err.Error())
			}
			return
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			if onError != nil {
				onError("Failed to read response body: " + err.Error())
			}
			return
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			if onSuccess != nil {
				onSuccess(string(body))
			}
		} else {
			if onError != nil {
				onError(fmt.Sprintf("API Error (status %d): %s", resp.StatusCode, string(body)))
			}
		}
	}()
}
