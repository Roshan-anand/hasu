package handlers

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/Roshan-anand/godploy/internal/db"
	ghservice "github.com/Roshan-anand/godploy/internal/lib/gh"
	"github.com/Roshan-anand/godploy/internal/lib/security"
	"github.com/Roshan-anand/godploy/internal/lib/types"
	"github.com/Roshan-anand/godploy/internal/lib/utils"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v5"
)

// Generates a unique GitHub App name per manifest to avoid collisions across setup attempts.
func generateGitHubManifestAppName() (string, error) {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	randomBytes := make([]byte, 6)
	suffix := make([]byte, 6)

	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}

	for i, b := range randomBytes {
		suffix[i] = chars[int(b)%len(chars)]
	}

	return "godploy-" + string(suffix), nil
}

// remove the session data
func removeSession(query *db.Queries, state string) {
	if err := query.DeleteRedirectSession(context.Background(), state); err != nil {
		fmt.Println("Error deleting redirect session:", err)
	}
}

// binds and validate the given data
func BindAndValidate(b any, c *echo.Context, v *validator.Validate) *types.Res[struct{}] {

	if err := c.Bind(b); err != nil {
		fmt.Printf("Error binding request data: %v\n", err)
		return &types.Res[struct{}]{Message: "Invalid Data"}
	}

	if err := v.Struct(b); err != nil {
		return &types.Res[struct{}]{Message: fmt.Sprintf("validation error : %v", err)}
	}

	return nil
}

// get github app manifest data
func getManifestData(url string, state string) (string, error) {
	appName, err := generateGitHubManifestAppName()
	if err != nil {
		return "", err
	}

	manifest := map[string]interface{}{
		"name": appName,
		"url":  url,
		"hook_attributes": map[string]string{
			"url": url + "/api/github/webhook",
		},
		"redirect_url": url + "/api/provider/github/app/callback",
		// "callback_urls": []string{"http://localhost:8080/api/provider/github/app/callback"},
		"setup_url": fmt.Sprintf("%s/api/provider/github/app/setup?state=%s", url, state),
		"public":    true,
		"default_permissions": map[string]string{
			"contents":      "read",
			"metadata":      "read",
			"pull_requests": "read",
		},
		"default_events": []string{"push"},
	}

	manifestDataB, err := json.Marshal(manifest)
	if err != nil {
		return "", err
	}

	return string(manifestDataB), nil
}

// auto-submitting form template — POST to GitHub with manifest in body (required by GitHub manifest flow)
const githubManifestFormTmpl = `<!DOCTYPE html>
<html>
<body>
  <form id="mf" action="https://github.com/settings/apps/new?state={{.State}}" method="POST">
    <input type="hidden" name="manifest" value="{{.Manifest}}">
  </form>
  <script>document.getElementById("mf").submit();</script>
</body>
</html>`

type ServiceEnvArray struct {
	Env          []string
	BuildArgs    []string
	BuildSecrets []string
}

type ServiceEnvByte struct {
	Env          []byte
	BuildArgs    []byte
	BuildSecrets []byte
}

// remove all the empty string from the array and return pointer to new array
func cleanArray(arr []string) []string {
	return slices.DeleteFunc(arr, func(s string) bool {
		return s == ""
	})
}

// to unmarshal all the evn into array of string
func UnmarshalServiceEnv(e *ServiceEnvByte) (*ServiceEnvArray, error) {
	var env []string
	var build_args []string
	var build_secrets []string

	if err := json.Unmarshal(e.Env, &env); err != nil {
		return nil, fmt.Errorf("Failed to unmarshal env: %v", err)
	}

	if err := json.Unmarshal(e.BuildArgs, &build_args); err != nil {
		return nil, fmt.Errorf("Failed to unmarshal build args: %v", err)
	}

	if err := json.Unmarshal(e.BuildSecrets, &build_secrets); err != nil {
		return nil, fmt.Errorf("Failed to unmarshal build secrets: %v", err)
	}

	return &ServiceEnvArray{
		Env:          cleanArray(env),
		BuildArgs:    cleanArray(build_args),
		BuildSecrets: cleanArray(build_secrets),
	}, nil
}

// to marshal all the env into byte array
func MarshalServiceEnv(e *ServiceEnvArray) (*ServiceEnvByte, error) {
	envByte, err := json.Marshal(e.Env)
	if err != nil {
		return nil, fmt.Errorf("Failed to marshal env: %v", err)
	}

	buildArgsByte, err := json.Marshal(e.BuildArgs)
	if err != nil {
		return nil, fmt.Errorf("Failed to marshal build args: %v", err)
	}

	buildSecretsByte, err := json.Marshal(e.BuildSecrets)
	if err != nil {
		return nil, fmt.Errorf("Failed to marshal build secrets: %v", err)
	}

	return &ServiceEnvByte{
		Env:          envByte,
		BuildArgs:    buildArgsByte,
		BuildSecrets: buildSecretsByte,
	}, nil
}

type GenerateNameRes struct {
	ServiceName string
	ImgName     string
}

// used as uniquecontainer name and code storing path
func generateServiceAndImgName(name string, branch string) *GenerateNameRes {
	branch = strings.ReplaceAll(branch, "/", "-")

	base := fmt.Sprintf("%s-%s", name, branch)
	id := security.GenerateRandomID(3, true)
	serviceName := fmt.Sprintf("%s-%s", base, id)
	imgName := strings.ToLower(fmt.Sprintf("%s-dyp_%s", base, id))

	return &GenerateNameRes{
		ServiceName: serviceName,
		ImgName:     imgName,
	}
}

// Bundles GitHub client creation, repo fetch, and latest commit fetch into one call.
type GitHubDeployData struct {
	Token        string
	Owner        string
	RepoName     string
	RepoFullName string
	RepoURL      string
	RepoURLPath  string
	CommitHash   string
	CommitMsg    string
}

func GetGitHubDeployData(q *db.Queries, ghAppID int64, ghRepoID int64, branch string) (*GitHubDeployData, error) {
	gh, err := ghservice.New(q, ghAppID)
	if err != nil {
		return nil, err
	}

	repo, err := gh.GetRepo(ghRepoID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch repository info: %w", err)
	}

	commit, err := gh.GetLatestCommit(repo.Owner, repo.Name, branch)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch latest commit: %w", err)
	}

	urlPath, err := utils.GetUrltHostNPath(repo.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid repository url: %w", err)
	}

	return &GitHubDeployData{
		Token:        gh.Token,
		Owner:        repo.Owner,
		RepoName:     repo.Name,
		RepoFullName: repo.FullName,
		RepoURL:      repo.URL,
		RepoURLPath:  urlPath,
		CommitHash:   commit.Hash,
		CommitMsg:    commit.Message,
	}, nil
}
