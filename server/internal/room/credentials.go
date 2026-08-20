package room

import (
	"crypto/subtle"
	"errors"
)

var ErrInvalidCredential = errors.New("invalid reconnect token")

func cloneCredentials(source map[string]string) map[string]string {
	credentials := make(map[string]string, len(source))
	for playerID, token := range source {
		if token != "" {
			credentials[playerID] = token
		}
	}
	return credentials
}

func validCredential(credentials map[string]string, playerID, token string) bool {
	expected, ok := credentials[playerID]
	if !ok || expected == "" || token == "" || len(expected) != len(token) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(expected), []byte(token)) == 1
}

func pruneCredentials(credentials map[string]string, statePlayers map[string]struct{}) {
	for playerID := range credentials {
		if _, exists := statePlayers[playerID]; !exists {
			delete(credentials, playerID)
		}
	}
}
