package testing

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/Roshan-anand/godploy/internal/db"
	"github.com/Roshan-anand/godploy/internal/handlers"
	deployjob "github.com/Roshan-anand/godploy/internal/jobs/deployment"
	"github.com/Roshan-anand/godploy/internal/lib/docker"
	"github.com/Roshan-anand/godploy/internal/lib/security"
	"github.com/Roshan-anand/godploy/internal/lib/types"
	"github.com/Roshan-anand/godploy/internal/lib/utils"
	"github.com/google/uuid"
)

func TestAppWorker(t *testing.T) {
	server, h, err := GetDummyServerHandler()
	if err != nil {
		t.Fatal(err)
	}

	user := mockUserRejister(h, t, true)
	if err != nil {
		t.Fatal(err)
	}

	aps := &handlers.CreateAppServiceReq{
		InstanceID:    user.InstanceID,
		Name:          "newapp",
		GitProvider:   "github",
		Public:        true,
		Port:          80,
		BuildPath:     "/",
		WatchPath:     "/",
		Env:           []string{},
		BuildArgs:     []string{},
		BuildSecrets:  []string{},
		DockerBuild:   &handlers.DockerBuildReq{},
		DefaultBranch: "main",
	}

	var appServiceID uuid.UUID
	// var deploymentID uuid.UUID

	t.Run("create new app service", func(t *testing.T) {
		q := server.DB.Queries
		qCtx := context.Background()

		url := "/home/roshan-anand/workspace/personal/godploy_workspace/samples/portfolio"

		// used as unique image and service name
		unique := docker.GenerateServiceAndImgName(aps.Name, aps.DefaultBranch)

		// convert into bytes
		envByte, err := utils.MarshalServiceEnv(&utils.ServiceEnvArray{
			Env:          aps.Env,
			BuildArgs:    aps.BuildArgs,
			BuildSecrets: aps.BuildSecrets,
		})
		if err != nil {
			t.Fatal(err)
		}

		// create app internal url (accessible within the same Docker network)
		internalURL := fmt.Sprintf("http://%s:%d", unique.ServiceName, aps.Port)

		ghApp, err := q.GetAllGhAppsByEmail(context.Background(), user.Email)
		if err != nil {
			t.Fatal(err)
		}

		// create a new service
		service, err := q.CreateAppService(qCtx, db.CreateAppServiceParams{
			ID:                security.GeneratePrimaryKey(),
			InstanceID:        aps.InstanceID,
			Type:              types.AppServiceType,
			Name:              aps.Name,
			GitProvider:       aps.GitProvider,
			GhAppID:           ghApp[0].AppID,
			GhRepoID:          aps.GhRepoID,
			GhRepoName:        "portfolio",
			GhRepoUrl:         url,
			BuildPath:         aps.BuildPath,
			WatchPath:         aps.WatchPath,
			Env:               envByte.Env,
			BuildArgs:         envByte.BuildArgs,
			BuildSecrets:      envByte.BuildSecrets,
			DockerFilepath:    aps.DockerBuild.FilePath,
			DockerContextpath: aps.DockerBuild.ContextPath,
			DockerBuildstage:  aps.DockerBuild.BuildStage,
			IsPublic:          aps.Public,
			Branch:            aps.DefaultBranch,
			SwarmService:      unique.ServiceName,
			Port:              aps.Port,
			InternalUrl:       internalURL,
		})
		if err != nil {
			t.Fatal(err)
		}

		// create a new deployment for the app service
		dID, err := q.CreateDeployment(qCtx, db.CreateDeploymentParams{
			ID:         security.GeneratePrimaryKey(),
			ServiceID:  service.ID,
			CommitHash: "commit",
			CommitMsg:  "commit",
			IsCurrent:  true,
		})
		if err != nil {
			t.Fatal(err)
		}

		done := make(chan error, 1)

		// push a new deployment job to the queue
		if err := server.Services.Deployment.AssignDeploy(context.Background(), &deployjob.DeploymentServiceParams{
			DeploymentID:      dID,
			InstanceID:        aps.InstanceID,
			ServiceID:         service.ID,
			Token:             "not_needed",
			Url:               url,
			Branch:            aps.DefaultBranch,
			SwarmService:      unique.ServiceName,
			BuildPath:         aps.BuildPath,
			DockerFilePath:    aps.DockerBuild.FilePath,
			DockerContextPath: aps.DockerBuild.ContextPath,
			DockerBuildStage:  aps.DockerBuild.BuildStage,
			ImgName:           unique.ServiceName,
			Env:               aps.Env,
			BuildArgs:         aps.BuildArgs,
			BuildSecrets:      aps.BuildSecrets,
			IsPublic:          aps.Public,
			GitProvider:       types.GitLocalProvider,
		}, done); err != nil {
			t.Fatal(err)
		}

		err = <-done
		if err != nil {
			t.Fatalf("done chan err %v", err)
		}

		t.Logf("created new app service with ID: %s", service.ID)

		appServiceID = service.ID
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
			printRaw(body, t)
			t.Fatalf("expected status code %d, got %d", http.StatusOK, rec.Code)
		}

		var res types.Res[[]db.Deployment]
		if err := readAndUnmarshl(body, &res); err != nil {
			t.Fatal(err)
		}

		if len(res.Data) == 0 {
			t.Fatal("expected at least one deployment, got 0")
		}

		// deploymentID = res.Data[0].ID
	})

	t.Run("get domain and port", func(t *testing.T) {
		query := url.Values{}
		query.Add("service_id", appServiceID.String())
		rec, err := TestEchoHandler(&TestEchoBody{T: t, H: h.Service.GetDomainPort, IsAuth: true, Query: query})
		if err != nil {
			t.Fatal(err)
		}
		body := rec.Result().Body
		defer body.Close()

		if rec.Code != http.StatusOK {
			printRaw(body, t)
			t.Fatalf("expected status code %d, got %d", http.StatusOK, rec.Code)
		}

		var res types.Res[db.GetDomainAndPortByServiceIdRow]
		if err := readAndUnmarshl(body, &res); err != nil {
			t.Fatal(err)
		}

		if res.Data.Port != aps.Port {
			t.Fatalf("expected port %d, got %d", aps.Port, res.Data.Port)
		}
	})

	t.Run("update domain and port", func(t *testing.T) {
		updateReq := &handlers.UpdateDomainReq{
			ServiceID: appServiceID,
			Domain:    "example.com",
			Port:      8080,
			IsPublic:  true,
		}
		rec, err := TestEchoHandler(&TestEchoBody{T: t, H: h.Service.UpdateAppServiceDomain, IsAuth: true, Body: updateReq})
		if err != nil {
			t.Fatal(err)
		}
		body := rec.Result().Body
		defer body.Close()

		if rec.Code != http.StatusOK {
			printRaw(body, t)
			t.Fatalf("expected status code %d, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("get service env", func(t *testing.T) {
		query := url.Values{}
		query.Add("service_id", appServiceID.String())
		rec, err := TestEchoHandler(&TestEchoBody{T: t, H: h.Service.GetServiceEnv, IsAuth: true, Query: query})
		if err != nil {
			t.Fatal(err)
		}
		body := rec.Result().Body
		defer body.Close()

		if rec.Code != http.StatusOK {
			printRaw(body, t)
			t.Fatalf("expected status code %d, got %d", http.StatusOK, rec.Code)
		}

		var res types.Res[handlers.GetEnvRes]
		if err := readAndUnmarshl(body, &res); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("update service env", func(t *testing.T) {
		updateReq := &handlers.UpdateEnvReq{
			ServiceID:    appServiceID,
			Env:          []string{"KEY=value"},
			BuildArgs:    []string{},
			BuildSecrets: []string{},
		}
		rec, err := TestEchoHandler(&TestEchoBody{T: t, H: h.Service.UpdateAppServiceEnv, IsAuth: true, Body: updateReq})
		if err != nil {
			t.Fatal(err)
		}
		body := rec.Result().Body
		defer body.Close()

		if rec.Code != http.StatusOK {
			printRaw(body, t)
			t.Fatalf("expected status code %d, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("get app service settings", func(t *testing.T) {
		query := url.Values{}
		query.Add("service_id", appServiceID.String())
		rec, err := TestEchoHandler(&TestEchoBody{T: t, H: h.Service.GetAppServiceSettings, IsAuth: true, Query: query})
		if err != nil {
			t.Fatal(err)
		}
		body := rec.Result().Body
		defer body.Close()

		if rec.Code != http.StatusOK {
			printRaw(body, t)
			t.Fatalf("expected status code %d, got %d", http.StatusOK, rec.Code)
		}

		var res types.Res[handlers.AppServiceSettingsRes]
		if err := readAndUnmarshl(body, &res); err != nil {
			t.Fatal(err)
		}

		if res.Data.BuildPath != "/" {
			t.Fatalf("expected build_path '/', got '%s'", res.Data.BuildPath)
		}
		if res.Data.WatchPath != "/" {
			t.Fatalf("expected watch_path '/', got '%s'", res.Data.WatchPath)
		}
		if res.Data.Port != 8080 {
			t.Fatalf("expected port 8080, got %d", res.Data.Port)
		}
		if !res.Data.IsPublic {
			t.Fatal("expected is_public to be true")
		}
	})

	t.Run("update build settings", func(t *testing.T) {
		updateReq := &handlers.UpdateAppServiceBuildSettingsReq{
			ServiceID:         appServiceID,
			BuildPath:         "/app",
			WatchPath:         "/app/src",
			DockerFilepath:    "prod.Dockerfile",
			DockerContextpath: ".",
			DockerBuildstage:  "builder",
		}
		rec, err := TestEchoHandler(&TestEchoBody{T: t, H: h.Service.UpdateAppServiceBuildSettings, IsAuth: true, Body: updateReq})
		if err != nil {
			t.Fatal(err)
		}
		body := rec.Result().Body
		defer body.Close()

		if rec.Code != http.StatusOK {
			printRaw(body, t)
			t.Fatalf("expected status code %d, got %d", http.StatusOK, rec.Code)
		}

		// verify the updated values via GetAppServiceSettings
		query := url.Values{}
		query.Add("service_id", appServiceID.String())
		rec, err = TestEchoHandler(&TestEchoBody{T: t, H: h.Service.GetAppServiceSettings, IsAuth: true, Query: query})
		if err != nil {
			t.Fatal(err)
		}
		body = rec.Result().Body
		defer body.Close()

		if rec.Code != http.StatusOK {
			printRaw(body, t)
			t.Fatalf("expected status code %d, got %d", http.StatusOK, rec.Code)
		}

		var res types.Res[handlers.AppServiceSettingsRes]
		if err := readAndUnmarshl(body, &res); err != nil {
			t.Fatal(err)
		}

		if res.Data.BuildPath != "/app" {
			t.Fatalf("expected build_path '/app', got '%s'", res.Data.BuildPath)
		}
		if res.Data.WatchPath != "/app/src" {
			t.Fatalf("expected watch_path '/app/src', got '%s'", res.Data.WatchPath)
		}
		if res.Data.DockerFilepath != "prod.Dockerfile" {
			t.Fatalf("expected docker_filepath 'prod.Dockerfile', got '%s'", res.Data.DockerFilepath)
		}
		if res.Data.DockerBuildstage != "builder" {
			t.Fatalf("expected docker_buildstage 'builder', got '%s'", res.Data.DockerBuildstage)
		}
	})

	t.Run("scale app service", func(t *testing.T) {
		scaleReq := &handlers.ScaleAppServiceReq{
			ServiceId: appServiceID,
			Replicas:  2,
		}
		rec, err := TestEchoHandler(&TestEchoBody{T: t, H: h.Service.ScaleAppService, IsAuth: true, Body: scaleReq})
		if err != nil {
			t.Fatal(err)
		}
		body := rec.Result().Body
		defer body.Close()

		if rec.Code != http.StatusOK {
			printRaw(body, t)
			t.Fatalf("expected status code %d, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("pause app service", func(t *testing.T) {
		pauseReq := &handlers.PauseAppServiceReq{
			ServiceID: appServiceID,
		}
		rec, err := TestEchoHandler(&TestEchoBody{T: t, H: h.Service.PauseAppService, IsAuth: true, Body: pauseReq})
		if err != nil {
			t.Fatal(err)
		}
		body := rec.Result().Body
		defer body.Close()

		if rec.Code != http.StatusOK {
			printRaw(body, t)
			t.Fatalf("expected status code %d, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("resume app service", func(t *testing.T) {
		resumeReq := &handlers.ResumeAppServiceReq{
			ServiceID: appServiceID,
		}
		rec, err := TestEchoHandler(&TestEchoBody{T: t, H: h.Service.ResumeAppService, IsAuth: true, Body: resumeReq})
		if err != nil {
			t.Fatal(err)
		}
		body := rec.Result().Body
		defer body.Close()

		if rec.Code != http.StatusOK {
			printRaw(body, t)
			t.Fatalf("expected status code %d, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("delete app service", func(t *testing.T) {
		deleteReq := &handlers.ServiceReq{ServiceId: appServiceID}
		rec, err := TestEchoHandler(&TestEchoBody{T: t, H: h.Service.DeleteAppService, IsAuth: true, Body: deleteReq})
		if err != nil {
			t.Fatal(err)
		}
		body := rec.Result().Body
		defer body.Close()

		if rec.Code != http.StatusOK {
			printRaw(body, t)
			t.Fatalf("expected status code %d, got %d", http.StatusOK, rec.Code)
		}

		query := url.Values{}
		query.Add("instance_id", user.InstanceID.String())
		rec, err = TestEchoHandler(&TestEchoBody{T: t, H: h.Service.GetAllServices, IsAuth: true, Query: query})
		if err != nil {
			t.Fatal(err)
		}
		body = rec.Result().Body
		defer body.Close()

		if rec.Code != http.StatusOK {
			printRaw(body, t)
			t.Fatalf("expected status code %d, got %d", http.StatusOK, rec.Code)
		}

		var res types.Res[[]db.GetAllServiceRow]
		if err := readAndUnmarshl(body, &res); err != nil {
			t.Fatal(err)
		}

		for _, service := range res.Data {
			if service.ID == appServiceID {
				t.Fatalf("expected app service %s to be deleted", appServiceID)
			}
		}
	})
}
