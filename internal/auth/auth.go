// Package auth provides secure credential storage using the OS keyring/keychain.
// It supports macOS Keychain, Windows Credential Manager, and Linux Secret Service.
package auth

import (
	"fmt"

	"github.com/zalando/go-keyring"
)

const (
	// ServiceName is the identifier used in the keychain for this application
	ServiceName = "com.github.dl-alexandre.abc"
	// ClientIDKey is the key for storing the client ID
	ClientIDKey = "client_id"
	// ClientSecretKey is the key for storing the client secret
	ClientSecretKey = "client_secret"
)

// Credentials holds OAuth2 credentials
type Credentials struct {
	ClientID     string
	ClientSecret string
}

// Store saves credentials to the OS keyring
func Store(creds Credentials) error {
	if creds.ClientID == "" || creds.ClientSecret == "" {
		return fmt.Errorf("both client_id and client_secret are required")
	}

	// Store client ID
	if err := keyring.Set(ServiceName, ClientIDKey, creds.ClientID); err != nil {
		return fmt.Errorf("failed to store client_id in keyring: %w", err)
	}

	// Store client secret
	if err := keyring.Set(ServiceName, ClientSecretKey, creds.ClientSecret); err != nil {
		return fmt.Errorf("failed to store client_secret in keyring: %w", err)
	}

	return nil
}

// Retrieve gets credentials from the OS keyring
func Retrieve() (*Credentials, error) {
	clientID, err := keyring.Get(ServiceName, ClientIDKey)
	if err != nil {
		if err == keyring.ErrNotFound {
			return nil, fmt.Errorf("no credentials found in keyring; run 'abc auth login' to store credentials")
		}
		return nil, fmt.Errorf("failed to retrieve client_id from keyring: %w", err)
	}

	clientSecret, err := keyring.Get(ServiceName, ClientSecretKey)
	if err != nil {
		if err == keyring.ErrNotFound {
			return nil, fmt.Errorf("no credentials found in keyring; run 'abc auth login' to store credentials")
		}
		return nil, fmt.Errorf("failed to retrieve client_secret from keyring: %w", err)
	}

	return &Credentials{
		ClientID:     clientID,
		ClientSecret: clientSecret,
	}, nil
}

// Delete removes credentials from the OS keyring
func Delete() error {
	// Delete client ID (ignore errors if not found)
	if err := keyring.Delete(ServiceName, ClientIDKey); err != nil {
		// Ignore ErrNotFound since we're trying to delete anyway
		if err != keyring.ErrNotFound {
			return fmt.Errorf("failed to delete client_id from keyring: %w", err)
		}
	}

	// Delete client secret (ignore errors if not found)
	if err := keyring.Delete(ServiceName, ClientSecretKey); err != nil {
		// Ignore ErrNotFound since we're trying to delete anyway
		if err != keyring.ErrNotFound {
			return fmt.Errorf("failed to delete client_secret from keyring: %w", err)
		}
	}

	return nil
}

// Check verifies if credentials exist in the keyring
func Check() bool {
	_, err := keyring.Get(ServiceName, ClientIDKey)
	if err != nil {
		return false
	}
	_, err = keyring.Get(ServiceName, ClientSecretKey)
	return err == nil
}
