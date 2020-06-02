package auth

import (
	"encoding/json"
	"fmt"
	"net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UserLookup describes functions for querying user.
type UserLookup interface {
	GetUserIDbyToken(token string) (string, error)
}

// OAuth implements the UserLookup interface.
type OAuth struct {
	url string
}

// ensure OAuth implements the UserLookup interface.
var _ UserLookup = &OAuth{}

// NewOAuthClient returns a new hydra client for the given url.
func NewOAuthClient(url string) (*OAuth, error) {
	return &OAuth{
		url: url,
	}, nil
}

// GetUserIDbyToken queries the user id from UserLookup using the given access token.
func (a *OAuth) GetUserIDbyToken(accessToken string) (string, error) {
	headers := map[string][]string{
		"Accept":        []string{"application/json"},
		"Authorization": []string{fmt.Sprintf("Bearer %s", accessToken)},
	}
	// query user info from oauthClient
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/userinfo", a.url), nil)
	if err != nil {
		return "", status.Errorf(codes.Internal, "unable to create http request for querying user info")
	}
	req.Header = headers
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return "", status.Errorf(codes.Internal, "unable to query user info")
	}
	defer res.Body.Close()
	switch res.StatusCode {
	case 200:
		var userInfo struct {
			Subject string `json:"sub"`
		}
		if err = json.NewDecoder(res.Body).Decode(&userInfo); err != nil {
			return "", status.Errorf(codes.Internal, "unable to read user info response")
		}
		return userInfo.Subject, nil
	case 401:
		return "", status.Errorf(codes.Unauthenticated, "invalid access token")
	default:
		return "", status.Errorf(codes.Internal, "unexpected status code while querying user info")
	}
}
