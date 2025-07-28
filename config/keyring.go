package config

import (
	"encoding/json"
	"fmt"

	"github.com/zalando/go-keyring"
	"golang.org/x/oauth2"
)

const (
	ServiceName = "meetingbar"
	TokenPrefix = "oauth_token_"
)

func StoreToken(accountID string, token *oauth2.Token) error {
	tokenJSON, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}
	
	key := TokenPrefix + accountID
	return keyring.Set(ServiceName, key, string(tokenJSON))
}

func GetToken(accountID string) (*oauth2.Token, error) {
	key := TokenPrefix + accountID
	tokenJSON, err := keyring.Get(ServiceName, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get token from keyring: %w", err)
	}
	
	var token oauth2.Token
	if err := json.Unmarshal([]byte(tokenJSON), &token); err != nil {
		return nil, fmt.Errorf("failed to unmarshal token: %w", err)
	}
	
	return &token, nil
}

func DeleteToken(accountID string) error {
	key := TokenPrefix + accountID
	return keyring.Delete(ServiceName, key)
}