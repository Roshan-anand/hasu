package testing

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/Roshan-anand/godploy/internal/db"
	"github.com/Roshan-anand/godploy/internal/handlers"
	"github.com/Roshan-anand/godploy/internal/lib/types"
	"github.com/google/uuid"
)

func TestAppService(t *testing.T) {
	_, h, err := GetDummyServerHandler()
	if err != nil {
		t.Fatal(err)
	}

	user := mockUserRejister(h, t, true)
	if err != nil {
		t.Fatal(err)
	}

	createAppServiceReq := &handlers.CreateAppServiceReq{
		ProjectID:   user.ProjectID,
		Name:        "newapp",
		GitProvider: "github",
		Public:      true,
		BuildPath:   "/",
		WatchPath:   "/",
		DockerBuild: &handlers.DockerBuildReq{},
	}

	var appServiceID uuid.UUID
	var deploymentID uuid.UUID

	t.Run("get all github apps", func(t *testing.T) {
		rec, err := TestEchoHandler(&TestEchoBody{T: t, H: h.Git.GetAllGithubApps, IsAuth: true})
		if err != nil {
			t.Fatal(err)
		}
		body := rec.Result().Body
		defer body.Close()

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status code %d, got %d", http.StatusOK, rec.Code)
		}

		var res types.Res[[]db.GetAllGhAppsByEmailRow]
		if err := readAndUnmarshl(body, &res); err != nil {
			t.Fatal(err)
		}

		if len(res.Data) == 0 {
			t.Fatal("expected at least one github app, got 0")
		}

		createAppServiceReq.GhAppID = res.Data[0].AppID
	})

	t.Run("get all repos of the github app", func(t *testing.T) {
		query := url.Values{}
		query.Add("app_id", fmt.Sprint(createAppServiceReq.GhAppID))
		rec, err := TestEchoHandler(&TestEchoBody{T: t, H: h.Git.GetGithubRepoList, Query: query, IsAuth: true})
		if err != nil {
			t.Fatal(err)
		}
		body := rec.Result().Body
		defer body.Close()

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status code %d, got %d", http.StatusOK, rec.Code)
		}

		var res types.Res[[]handlers.GetGithubRepoListRes]
		if err := readAndUnmarshl(body, &res); err != nil {
			t.Fatal(err)
		}

		if len(res.Data) == 0 {
			t.Fatal("expected at least one repository, got 0")
		}

		createAppServiceReq.GhRepoID = res.Data[0].ID
		createAppServiceReq.DefaultBranch = res.Data[0].DefaultBranch
	})

	t.Run("create new app service", func(t *testing.T) {
		rec, err := TestEchoHandler(&TestEchoBody{T: t, H: h.Service.CreateAppService, IsAuth: true, Body: createAppServiceReq})
		if err != nil {
			t.Fatal(err)
		}
		body := rec.Result().Body
		defer body.Close()

		if rec.Code != http.StatusOK {
			msg, err := readOnly(body)
			if err != nil {
				t.Fatal(err)
			}
			t.Log(msg)
			t.Fatalf("expected status code %d, got %d", http.StatusOK, rec.Code)
		}

		var res types.Res[uuid.UUID]
		if err := readAndUnmarshl(body, &res); err != nil {
			t.Fatal(err)
		}
		appServiceID = res.Data
	})

	t.Run("get all deployments", func(t *testing.T) {
		query := url.Values{}
		query.Add("service_id", appServiceID.String())
		rec, err := TestEchoHandler(&TestEchoBody{T: t, H: h.Deployment.GetServiceDeployments, IsAuth: true, Query: query})
		if err != nil {
			t.Fatal(err)
		}
		body := rec.Result().Body
		defer body.Close()

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status code %d, got %d", http.StatusOK, rec.Code)
		}

		var res types.Res[[]db.GetDeploymentsByServiceIDRow]
		if err := readAndUnmarshl(body, &res); err != nil {
			t.Fatal(err)
		}

		if len(res.Data) == 0 {
			t.Fatal("expected at least one deployment, got 0")
		}

		deploymentID = res.Data[0].ID
		t.Log("deployemnt :", deploymentID)
	})

	// TODO : write propoer test for sse and delete app service
	// t.Run("wait till SSE ends", func(t *testing.T) {
	// 	query := url.Values{}
	// 	query.Add("deployment_id", deploymentID.String())
	// 	_, err := TestEchoHandler(&TestEchoBody{T: t, H: h.Deployment.SubscribeServiceDeploymentLogs, IsAuth: true, Query: query})
	// 	if err != nil {
	// 		t.Fatal(err)
	// 	}
	// })
}
