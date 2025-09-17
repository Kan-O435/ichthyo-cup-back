package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"syscall/js"
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

// LoginResponse represents the response from the login API
type LoginResponse struct {
	Token string `json:"token"`
}

// JWTPayload represents the JWT payload
type JWTPayload struct {
	UserID   string `json:"userId"`
	Username string `json:"username"`
}

// parseJWT parses a JWT token and returns the payload (without verification)
func parseJWT(token string) (*JWTPayload, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid JWT format")
	}

	// Decode the payload (second part)
	payload := parts[1]
	
	// Add padding if needed
	for len(payload)%4 != 0 {
		payload += "="
	}

	decoded, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to decode JWT payload: %v", err)
	}

	var jwtPayload JWTPayload
	if err := json.Unmarshal(decoded, &jwtPayload); err != nil {
		return nil, fmt.Errorf("failed to parse JWT payload: %v", err)
	}

	return &jwtPayload, nil
}

// storeUserData stores the token and user data in localStorage
func storeUserData(token string, userID string) {
	localStorage := js.Global().Get("localStorage")
	localStorage.Call("setItem", "jwt_token", token)
	localStorage.Call("setItem", "ichthyo_user", userID)
}
