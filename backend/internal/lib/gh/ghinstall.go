package gh

import (
	"context"
	"net/http"

	"github.com/Roshan-anand/godploy/internal/lib/security"
	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v84/github"
)

// creates a new ghinstallation transport.
func GetNewTransport(appID int64, installationID int64, hashPem string) (*ghinstallation.Transport, error) {
	pemKey, err := security.DecryptPEM(hashPem)
	if err != nil {
		return nil, err
	}

	itr, err := ghinstallation.New(
		http.DefaultTransport,
		appID,
		installationID,
		pemKey,
	)
	if err != nil {
		return nil, err
	}

	return itr, nil
}

// generates an github installation access token.
func GetGhToken(appID int64, installationID int64, hashPem string) (string, error) {

	itr, err := GetNewTransport(appID, installationID, hashPem)
	if err != nil {
		return "", err
	}

	token, err := itr.Token(context.Background())
	if err != nil {
		return "", err
	}

	return token, nil
}

// creates an installation-scoped GitHub client.
// Used for repo operations (list repos, clone, etc.) scoped to a specific installation.
func CreateGithubClient(ctx context.Context, appID int64, installationID int64, hashPem string) (*github.Client, error) {

	itr, err := GetNewTransport(appID, installationID, hashPem)
	if err != nil {
		return nil, err
	}

	client := github.NewClient(&http.Client{Transport: itr})
	return client, nil
}

// creates an app-level GitHub client authenticated as the GitHub App itself (JWT).
// Required for app-level API calls like GetInstallation, ListInstallations — these endpoints
func CreateAppClient(appID int64, hashPem string) (*github.Client, error) {

	pemKey, err := security.DecryptPEM(hashPem)
	if err != nil {
		return nil, err
	}

	itr, err := ghinstallation.NewAppsTransport(
		http.DefaultTransport,
		appID,
		pemKey,
	)
	if err != nil {
		return nil, err
	}

	client := github.NewClient(&http.Client{Transport: itr})
	return client, nil
}
