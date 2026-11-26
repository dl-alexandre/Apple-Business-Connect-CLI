package auth

import "testing"

func TestStoreRequiresBothCredentials(t *testing.T) {
	for _, creds := range []Credentials{{}, {ClientID: "id"}, {ClientSecret: "secret"}} {
		if err := Store(creds); err == nil {
			t.Errorf("Store(%+v) succeeded, want validation error", creds)
		}
	}
}
