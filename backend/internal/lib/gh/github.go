package ghservice

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

	"github.com/Roshan-anand/godploy/internal/db"
	"github.com/Roshan-anand/godploy/internal/lib/security"
	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v84/github"
)

type CommitInfo struct {
	Hash    string
	Message string
}

type RepoInfo struct {
	Name     string
	FullName string
	URL      string
	Owner    string
}

type GithubService struct {
	Client *github.Client // for custom github operations
	Token  string         // auth tokenfor github manual access
}

// creates a new github client instance
// note: client is installation-scoped GitHub client.
// Used for repo operations (list repos, clone, etc.) scoped to a specific installation.
//
// for app-level operations (list/get installation details), use NewGithubAppClient instead.
func New(q *db.Queries, ghAppId int64) (*GithubService, error) {
	// get the github app details from db
	ghApp, err := q.GetGhAppByAppId(context.Background(), ghAppId)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}
		return nil, err
	}

	if !ghApp.InstallationID.Valid {
		return nil, fmt.Errorf("github app with app id %d is not installed ", ghAppId)
	}

	itr, err := GetNewTransport(ghApp.AppID, ghApp.InstallationID.Int64, ghApp.PemKey)
	if err != nil {
		return nil, err
	}

	token, err := itr.Token(context.Background())
	if err != nil {
		return nil, err
	}

	client := github.NewClient(&http.Client{Transport: itr})

	return &GithubService{
		Client: client,
		Token:  token,
	}, nil
}

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

// creates a new app-level GitHub client instance.
//
// note: this client is app-scoped GitHub client.
// Used for app-level API calls like GetInstallation, ListInstallations, etc.
//
// for installation-scoped API calls (list repos, clone, etc.), use NewGithubService instead which returns an installation-scoped client.
func NewGithubAppClient(q *db.Queries, appID int64) (*github.Client, error) {
	// get the github app details from db
	ghApp, err := q.GetGhAppByAppId(context.Background(), appID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}
		return nil, err
	}

	pemKey, err := security.DecryptPEM(ghApp.PemKey)
	if err != nil {
		return nil, err
	}

	itr, err := ghinstallation.NewAppsTransport(
		http.DefaultTransport,
		ghApp.AppID,
		pemKey,
	)
	if err != nil {
		return nil, err
	}

	client := github.NewClient(&http.Client{Transport: itr})
	return client, nil
}

// to get the latest commit of the
func (gh *GithubService) GetLatestCommit(owner string, repoName string, branch string) (*CommitInfo, error) {

	commits, _, err := gh.Client.Repositories.ListCommits(context.Background(), owner, repoName, &github.CommitsListOptions{
		SHA: branch,
		ListOptions: github.ListOptions{
			PerPage: 1,
			Page:    1,
		},
	})
	if err != nil || len(commits) == 0 {
		return nil, fmt.Errorf("failed to fetch latest commit info from github: %v", err)
	}

	latestCommitHash := commits[0].GetSHA()
	latestCommitMsg := commits[0].GetCommit().GetMessage()

	return &CommitInfo{
		Hash:    latestCommitHash,
		Message: latestCommitMsg,
	}, nil
}

// to get the github repository details
func (gh *GithubService) GetRepo(repoID int64) (*RepoInfo, error) {
	repo, _, err := gh.Client.Repositories.GetByID(context.Background(), repoID)
	if err != nil {
		// return c.JSON(http.StatusBadRequ/est, types.Res[struct{}]{Message: "failed to fetch repository info from github"})
		return nil, fmt.Errorf("failed to fetch repository info from github: %v", err)
	}

	return &RepoInfo{
		Name:     repo.GetName(),
		FullName: repo.GetFullName(),
		URL:      repo.GetHTMLURL(),
		Owner:    repo.GetOwner().GetLogin(),
	}, nil
}
