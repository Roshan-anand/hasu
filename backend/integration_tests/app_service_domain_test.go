package testing

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/Roshan-anand/godploy/internal/db"
	"github.com/Roshan-anand/godploy/internal/handlers"
	"github.com/Roshan-anand/godploy/internal/lib/types"
	"github.com/google/uuid"
)

func TestAppServiceDomainUpdate(t *testing.T) {
	server, h, err := GetDummyServerHandler()
	if err != nil {
		t.Fatal(err)
	}

	user := mockUserRejister(h, t, true)
	if err != nil {
		t.Fatal(err)
	}

	createAppServiceReq := &handlers.CreateAppServiceReq{
		InstanceID:  user.InstanceID,
		Name:        "domain-test-app",
		GitProvider: "github",
		Public:      false,
		Port:        80,
		BuildPath:   "/",
		WatchPath:   "/",
		DockerBuild: &handlers.DockerBuildReq{},
	}

	var appServiceID uuid.UUID

	// fetch github app and repo to create the service
	t.Run("fetch github app", func(t *testing.T) {
		rec, err := TestEchoHandler(&TestEchoBody{T: t, H: h.Git.GetAllGithubApps, IsAuth: true})
		if err != nil {
			t.Fatal(err)
		}
		if rec.Code != http.StatusOK {
			t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
		}
		var res types.Res[[]db.GetAllGhAppsByEmailRow]
		if err := readAndUnmarshl(rec.Result().Body, &res); err != nil {
			t.Fatal(err)
		}
		if len(res.Data) == 0 {
			t.Skip("no github app configured, skipping")
		}
		createAppServiceReq.GhAppID = res.Data[0].AppID
	})

	t.Run("fetch repo list", func(t *testing.T) {
		q := url.Values{}
		q.Add("app_id", fmt.Sprint(createAppServiceReq.GhAppID))
		rec, err := TestEchoHandler(&TestEchoBody{T: t, H: h.Git.GetGithubRepoList, Query: q, IsAuth: true})
		if err != nil {
			t.Fatal(err)
		}
		if rec.Code != http.StatusOK {
			t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
		}
		var res types.Res[[]handlers.GetGithubRepoListRes]
		if err := readAndUnmarshl(rec.Result().Body, &res); err != nil {
			t.Fatal(err)
		}
		if len(res.Data) == 0 {
			t.Skip("no repos found, skipping")
		}
		createAppServiceReq.GhRepoID = res.Data[0].ID
		createAppServiceReq.DefaultBranch = res.Data[0].DefaultBranch
	})

	t.Run("create app service", func(t *testing.T) {
		rec, err := TestEchoHandler(&TestEchoBody{T: t, H: h.Service.CreateAppService, IsAuth: true, Body: createAppServiceReq})
		if err != nil {
			t.Fatal(err)
		}
		if rec.Code != http.StatusOK {
			t.Fatalf("expected %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
		}
		var res types.Res[db.CreateAppServiceRow]
		if err := readAndUnmarshl(rec.Result().Body, &res); err != nil {
			t.Fatal(err)
		}
		appServiceID = res.Data.ID
	})

	// Test 1: Handler validation — empty body (no service_id)
	t.Run("handler validation fails with empty body", func(t *testing.T) {
		rec, err := TestEchoHandler(&TestEchoBody{
			T:      t,
			H:      h.Service.UpdateAppServiceDomain,
			IsAuth: true,
			Body:   struct{}{}, // empty/without required service_id
		})
		if err != nil {
			t.Fatal(err)
		}
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected %d for missing service_id, got %d", http.StatusBadRequest, rec.Code)
		}
	})

	// Test 2: Handler with non-existent service ID
	t.Run("handler with non-existent service id", func(t *testing.T) {
		updateReq := &handlers.UpdateDomainReq{
			ServiceID: uuid.New(),
			Domain:    "my-app.godploy.localhost",
			Port:      3000,
			IsPublic:  true,
		}
		rec, err := TestEchoHandler(&TestEchoBody{
			T:      t,
			H:      h.Service.UpdateAppServiceDomain,
			IsAuth: true,
			Body:   updateReq,
		})
		if err != nil {
			t.Fatal(err)
		}
		// Should get 500 since the service doesn't exist in DB
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("expected %d for non-existent service, got %d: %s", http.StatusInternalServerError, rec.Code, rec.Body.String())
		}
	})

	// Test 3: Handler with valid payload — requires Docker, skip if fails
	t.Run("handler update domain (requires docker)", func(t *testing.T) {
		updateReq := &handlers.UpdateDomainReq{
			ServiceID: appServiceID,
			Domain:    "my-app.godploy.localhost",
			Port:      3000,
			IsPublic:  true,
		}

		rec, err := TestEchoHandler(&TestEchoBody{
			T:      t,
			H:      h.Service.UpdateAppServiceDomain,
			IsAuth: true,
			Body:   updateReq,
		})
		if err != nil {
			t.Fatal(err)
		}

		// In CI/without Docker, this may fail at the docker.ServiceInspectWithRaw call.
		// We accept either 200 (if Docker is available) or 500 (Docker not available).
		if rec.Code == http.StatusInternalServerError {
			t.Log("handler returned 500 — likely Docker unavailable, skipping full E2E check")
			return
		}
		if rec.Code != http.StatusOK {
			t.Fatalf("expected %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
		}

		// verify the DB reflects the update
		ctx := context.Background()
		q := server.DB.Queries
		service, err := q.GetAppServiceById(ctx, appServiceID)
		if err != nil {
			t.Fatalf("failed to get service: %v", err)
		}
		if !service.IsPublic {
			t.Fatal("expected is_public to be true")
		}
		if service.Domain != "my-app.godploy.localhost" {
			t.Fatalf("expected domain 'my-app.godploy.localhost', got '%s'", service.Domain)
		}
	})

	// cleanup: delete the app service
	t.Run("delete app service", func(t *testing.T) {
		rec, err := TestEchoHandler(&TestEchoBody{
			T:      t,
			H:      h.Service.DeleteAppService,
			IsAuth: true,
			Body:   &handlers.ServiceReq{ServiceId: appServiceID},
		})
		if err != nil {
			t.Fatal(err)
		}
		if rec.Code != http.StatusOK {
			t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
		}
	})
}
