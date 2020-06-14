package store

import (
	"encoding/json"
	"fmt"

	"github.com/rbicker/gooser/internal/utils"
	"go.mongodb.org/mongo-driver/bson"
)

// PageToken represents a token for pagination.
// It contains all the needed values to query the data for the next page.
type PageToken struct {
	FilterString     string
	OrderBy          string
	PaginationFilter bson.D
}

// PageTokenFromString takes the given json string and returns
// a corresponding page token.
func PageTokenFromString(s string) (*PageToken, error) {
	if s == "" {
		return nil, nil
	}
	var p *PageToken
	err := json.Unmarshal([]byte(s), p)
	if err != nil {
		return nil, fmt.Errorf("unable to json unmarshal: %w", err)
	}
	return p, nil
}

// PageTokenFromEncryptedString takes the encrypted json string, decrypts it
// and returns a corresponding page token.
func PageTokenFromEncryptedString(key, s string) (*PageToken, error) {
	if s == "" {
		return nil, nil
	}
	s, err := utils.Decrypt(key, s)
	if err != nil {
		return nil, fmt.Errorf("unable to decrypt given string: %w", err)
	}
	return PageTokenFromString(s)
}

// String converts the page token to a json string.
func (p *PageToken) String() string {
	b, _ := json.Marshal(p)
	return string(b)
}

// EncryptedString converts an encrypted string for the page token, based
// on the given key.
func (p *PageToken) EncryptedString(key string) (string, error) {
	b, _ := json.Marshal(p)
	return utils.Encrypt(key, string(b))
}
