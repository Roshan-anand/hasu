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
	"github.com/labstack/echo/v5"
)

func TestDependency(t *testing.T) {
	server, h, err := GetDummyServerHandler()
	if err != nil {
		t.Fatal(err)
	}
	q := server.DB.Queries

	user := mockUserRejister(h, t, true)

	createAppServiceReq := &handlers.CreateAppServiceReq{
		InstanceID:   user.InstanceID,
		Name:         "api",
		GitProvider:  "github",
		Public:       false,
		Port:         80,
		BuildPath:    "/",
		WatchPath:    "/",
		Env:          []string{},
		BuildArgs:    []string{},
		BuildSecrets: []string{},
		DockerBuild:  &handlers.DockerBuildReq{},
	}

	var sourceServiceID uuid.UUID
	var targetServiceID uuid.UUID
	var psqlServiceID uuid.UUID
	var redisServiceID uuid.UUID

	t.Run("setup github app for service creation", func(t *testing.T) {
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

	t.Run("setup github repo for service creation", func(t *testing.T) {
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

	t.Run("create source app service", func(t *testing.T) {
		rec, err := TestEchoHandler(&TestEchoBody{T: t, H: h.Service.CreateAppService, Body: createAppServiceReq, IsAuth: true})
		if err != nil {
			t.Fatal(err)
		}
		if rec.Code != http.StatusOK {
			printRaw(rec.Result().Body, t)
			t.Fatalf("expected status code %d, got %d", http.StatusOK, rec.Code)
		}
		body := rec.Result().Body
		defer body.Close()
		var res types.Res[db.CreateAppServiceRow]
		if err := readAndUnmarshl(body, &res); err != nil {
			t.Fatal(err)
		}
		sourceServiceID = res.Data.ID
	})

	t.Run("create target app service", func(t *testing.T) {
		req := *createAppServiceReq
		req.Name = "database"
		rec, err := TestEchoHandler(&TestEchoBody{T: t, H: h.Service.CreateAppService, Body: &req, IsAuth: true})
		if err != nil {
			t.Fatal(err)
		}
		if rec.Code != http.StatusOK {
			printRaw(rec.Result().Body, t)
			t.Fatalf("expected status code %d, got %d", http.StatusOK, rec.Code)
		}
		body := rec.Result().Body
		defer body.Close()
		var res types.Res[db.CreateAppServiceRow]
		if err := readAndUnmarshl(body, &res); err != nil {
			t.Fatal(err)
		}
		targetServiceID = res.Data.ID
	})

	t.Run("get dependency targets", func(t *testing.T) {
		rec, err := TestEchoHandler(&TestEchoBody{
			T:      t,
			H:      h.Dependency.GetDependencyTargets,
			Query:  url.Values{"service_id": {sourceServiceID.String()}},
			IsAuth: true,
		})
		if err != nil {
			t.Fatal(err)
		}
		if rec.Code != http.StatusOK {
			t.Fatalf("expected status code %d, got %d", http.StatusOK, rec.Code)
		}
		body := rec.Result().Body
		defer body.Close()
		var res types.Res[handlers.ListDependencyTargetsRes]
		if err := readAndUnmarshl(body, &res); err != nil {
			t.Fatal(err)
		}
		found := false
		for _, target := range res.Data.Targets {
			if target.ID == targetServiceID {
				found = true
				if !containsStr(target.AllowedCols, "internal_url") {
					t.Fatal("target should have internal_url in allowed cols")
				}
			}
		}
		if !found {
			t.Fatal("target service not found in dependency targets")
		}
	})

	t.Run("create dependency", func(t *testing.T) {
		rec, err := TestEchoHandler(&TestEchoBody{
			T: t,
			H: h.Dependency.CreateServiceDependency,
			Body: &handlers.CreateDependencyReq{
				SourceServiceID: sourceServiceID,
				TargetServiceID: targetServiceID,
				TargetCol:       "internal_url",
				EnvKey:          "DB_URL",
			},
			IsAuth: true,
		})
		if err != nil {
			t.Fatal(err)
		}
		if rec.Code != http.StatusCreated {
			printRaw(rec.Result().Body, t)
			t.Fatalf("expected status code %d, got %d", http.StatusCreated, rec.Code)
		}
		body := rec.Result().Body
		defer body.Close()
		var res types.Res[handlers.ServiceDependencyRes]
		if err := readAndUnmarshl(body, &res); err != nil {
			t.Fatal(err)
		}
		if res.Data.EnvKey != "DB_URL" {
			t.Fatalf("expected env key DB_URL, got %s", res.Data.EnvKey)
		}
		if res.Data.TargetCol != "internal_url" {
			t.Fatalf("expected target_col internal_url, got %s", res.Data.TargetCol)
		}
	})

	t.Run("list dependencies", func(t *testing.T) {
		rec, err := TestEchoHandler(&TestEchoBody{
			T:      t,
			H:      h.Dependency.GetServiceDependencies,
			Query:  url.Values{"service_id": {sourceServiceID.String()}},
			IsAuth: true,
		})
		if err != nil {
			t.Fatal(err)
		}
		if rec.Code != http.StatusOK {
			t.Fatalf("expected status code %d, got %d", http.StatusOK, rec.Code)
		}
		body := rec.Result().Body
		defer body.Close()
		var res types.Res[handlers.ListDependenciesRes]
		if err := readAndUnmarshl(body, &res); err != nil {
			t.Fatal(err)
		}
		if len(res.Data.Dependencies) != 1 {
			t.Fatalf("expected 1 dependency, got %d", len(res.Data.Dependencies))
		}
	})

	t.Run("resolve dependency env at deploy time", func(t *testing.T) {
		rows, err := q.ResolveDependencyEnv(context.Background(), sourceServiceID)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, r := range rows {
			if r.EnvKey == "DB_URL" {
				found = true
				val, ok := r.ResolvedValue.(string)
				if !ok || val == "" {
					t.Fatal("expected non-empty string resolved value for DB_URL")
				}
			}
		}
		if !found {
			t.Fatal("DB_URL not found in resolved dependency env")
		}
	})

	t.Run("reject self dependency", func(t *testing.T) {
		rec, err := TestEchoHandler(&TestEchoBody{
			T: t,
			H: h.Dependency.CreateServiceDependency,
			Body: &handlers.CreateDependencyReq{
				SourceServiceID: sourceServiceID,
				TargetServiceID: sourceServiceID,
				TargetCol:       "internal_url",
				EnvKey:          "SELF_URL",
			},
			IsAuth: true,
		})
		if err != nil {
			t.Fatal(err)
		}
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status code %d, got %d", http.StatusBadRequest, rec.Code)
		}
	})

	t.Run("reject invalid env key", func(t *testing.T) {
		rec, err := TestEchoHandler(&TestEchoBody{
			T: t,
			H: h.Dependency.CreateServiceDependency,
			Body: &handlers.CreateDependencyReq{
				SourceServiceID: sourceServiceID,
				TargetServiceID: targetServiceID,
				TargetCol:       "internal_url",
				EnvKey:          "123_INVALID",
			},
			IsAuth: true,
		})
		if err != nil {
			t.Fatal(err)
		}
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status code %d, got %d", http.StatusBadRequest, rec.Code)
		}
	})

	t.Run("reject duplicate env key", func(t *testing.T) {
		rec, err := TestEchoHandler(&TestEchoBody{
			T: t,
			H: h.Dependency.CreateServiceDependency,
			Body: &handlers.CreateDependencyReq{
				SourceServiceID: sourceServiceID,
				TargetServiceID: targetServiceID,
				TargetCol:       "internal_url",
				EnvKey:          "DB_URL",
			},
			IsAuth: true,
		})
		if err != nil {
			t.Fatal(err)
		}
		if rec.Code != http.StatusInternalServerError {
			body := rec.Result().Body
			defer body.Close()
			var res types.Res[struct{}]
			_ = readAndUnmarshl(body, &res)
			t.Fatalf("expected status code %d, got %d: %s", http.StatusInternalServerError, rec.Code, res.Message)
		}
	})

	t.Run("reject cross-instance target", func(t *testing.T) {
		otherOrg, err := q.CreateOrg(context.Background(), db.CreateOrgParams{
			ID:   uuid.New(),
			Name: "other-org",
		})
		if err != nil {
			t.Fatal(err)
		}
		otherProject, err := q.CreateProject(context.Background(), db.CreateProjectParams{
			ID:             uuid.New(),
			OrganizationID: otherOrg.ID,
			Name:           "other-project",
		})
		if err != nil {
			t.Fatal(err)
		}
		otherInstanceID := uuid.New()
		err = q.CreateInstance(context.Background(), db.CreateInstanceParams{
			ID:           otherInstanceID,
			ProjectID:    otherProject.ID,
			IsProduction: false,
			Name:         "other-instance",
			Network:      "other-network",
		})
		if err != nil {
			t.Fatal(err)
		}

		_, err = q.CreateGithubApp(context.Background(), db.CreateGithubAppParams{
			ID:             uuid.New(),
			Name:           "test-app",
			OrganizationID: otherOrg.ID,
			AppID:          999,
			PemKey:         "test",
			WebhookSecret:  "test",
		})
		if err != nil {
			t.Fatal(err)
		}

		otherServiceID, err := q.CreateAppService(context.Background(), db.CreateAppServiceParams{
			ID:                uuid.New(),
			InstanceID:        otherInstanceID,
			Type:              "app",
			Name:              "other-service",
			GitProvider:       "github",
			GhAppID:           999,
			GhRepoID:          1,
			GhRepoName:        "other/repo",
			GhRepoUrl:         "https://github.com/other/repo",
			BuildPath:         "/",
			WatchPath:         "/",
			Env:               []byte("[]"),
			BuildArgs:         []byte("[]"),
			BuildSecrets:      []byte("[]"),
			DockerFilepath:    "Dockerfile",
			DockerContextpath: ".",
			DockerBuildstage:  "",
			IsPublic:          false,
			Branch:            "main",
			SwarmService:      "other-service",
			Port:              80,
			InternalUrl:       "http://other-service",
		})
		if err != nil {
			t.Fatal(err)
		}

		rec2, err := TestEchoHandler(&TestEchoBody{
			T: t,
			H: h.Dependency.CreateServiceDependency,
			Body: &handlers.CreateDependencyReq{
				SourceServiceID: sourceServiceID,
				TargetServiceID: otherServiceID.ID,
				TargetCol:       "internal_url",
				EnvKey:          "OTHER_URL",
			},
			IsAuth: true,
		})
		if err != nil {
			t.Fatal(err)
		}
		if rec2.Code != http.StatusBadRequest {
			t.Fatalf("expected status code %d, got %d", http.StatusBadRequest, rec2.Code)
		}
	})

	t.Run("reject invalid target column", func(t *testing.T) {
		rec, err := TestEchoHandler(&TestEchoBody{
			T: t,
			H: h.Dependency.CreateServiceDependency,
			Body: &handlers.CreateDependencyReq{
				SourceServiceID: sourceServiceID,
				TargetServiceID: targetServiceID,
				TargetCol:       "invalid_col",
				EnvKey:          "INVALID",
			},
			IsAuth: true,
		})
		if err != nil {
			t.Fatal(err)
		}
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status code %d, got %d", http.StatusBadRequest, rec.Code)
		}
	})

	t.Run("create public app service with empty domain", func(t *testing.T) {
		// not needed as a separate test - just need a public service with empty domain
	})

	// We'll create a public app service with empty domain inline for the domain rejection test
	// and a public app service with a domain for the domain success test

	var publicEmptyDomainServiceID uuid.UUID
	var publicWithDomainServiceID uuid.UUID

	t.Run("create public app service with empty domain", func(t *testing.T) {
		req := *createAppServiceReq
		req.Name = "public-empty"
		req.Public = true
		rec, err := TestEchoHandler(&TestEchoBody{T: t, H: h.Service.CreateAppService, Body: &req, IsAuth: true})
		if err != nil {
			t.Fatal(err)
		}
		if rec.Code != http.StatusOK {
			printRaw(rec.Result().Body, t)
			t.Fatalf("expected status code %d, got %d", http.StatusOK, rec.Code)
		}
		body := rec.Result().Body
		defer body.Close()
		var res types.Res[db.CreateAppServiceRow]
		if err := readAndUnmarshl(body, &res); err != nil {
			t.Fatal(err)
		}
		publicEmptyDomainServiceID = res.Data.ID
	})

	t.Run("create public app service with domain", func(t *testing.T) {
		req := *createAppServiceReq
		req.Name = "public-domain"
		req.Public = true
		rec, err := TestEchoHandler(&TestEchoBody{T: t, H: h.Service.CreateAppService, Body: &req, IsAuth: true})
		if err != nil {
			t.Fatal(err)
		}
		if rec.Code != http.StatusOK {
			printRaw(rec.Result().Body, t)
			t.Fatalf("expected status code %d, got %d", http.StatusOK, rec.Code)
		}
		body := rec.Result().Body
		defer body.Close()
		var res types.Res[db.CreateAppServiceRow]
		if err := readAndUnmarshl(body, &res); err != nil {
			t.Fatal(err)
		}
		publicWithDomainServiceID = res.Data.ID
		// Update domain via handler
		_, err = TestEchoHandler(&TestEchoBody{
			T:      t,
			H:      h.Service.UpdateAppServiceDomain,
			Body:   &handlers.UpdateDomainReq{ServiceID: publicWithDomainServiceID, Domain: "example.com", IsPublic: true},
			IsAuth: true,
		})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("reject domain for internal app service", func(t *testing.T) {
		rec, err := TestEchoHandler(&TestEchoBody{
			T: t,
			H: h.Dependency.CreateServiceDependency,
			Body: &handlers.CreateDependencyReq{
				SourceServiceID: sourceServiceID,
				TargetServiceID: targetServiceID,
				TargetCol:       "domain",
				EnvKey:          "DOMAIN",
			},
			IsAuth: true,
		})
		if err != nil {
			t.Fatal(err)
		}
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status code %d, got %d", http.StatusBadRequest, rec.Code)
		}
	})

	t.Run("reject domain for public app with empty domain", func(t *testing.T) {
		rec, err := TestEchoHandler(&TestEchoBody{
			T: t,
			H: h.Dependency.CreateServiceDependency,
			Body: &handlers.CreateDependencyReq{
				SourceServiceID: sourceServiceID,
				TargetServiceID: publicEmptyDomainServiceID,
				TargetCol:       "domain",
				EnvKey:          "DOMAIN2",
			},
			IsAuth: true,
		})
		if err != nil {
			t.Fatal(err)
		}
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status code %d, got %d", http.StatusBadRequest, rec.Code)
		}
	})

	t.Run("create dependency on psql service", func(t *testing.T) {
		// Create psql service via handler
		rec, err := TestEchoHandler(&TestEchoBody{
			T: t,
			H: h.Service.CreatePsqlService,
			Body: &handlers.CreatePsqlServiceReq{
				InstanceID: user.InstanceID,
				Name:       "test-psql",
				DbName:     "testdb",
				DbUser:     "testuser",
				DbPassword: "testpass",
				Image:      "postgres:15",
			},
			IsAuth: true,
		})
		if err != nil {
			t.Fatal(err)
		}
		if rec.Code != http.StatusOK {
			printRaw(rec.Result().Body, t)
			t.Fatalf("expected status code %d, got %d", http.StatusOK, rec.Code)
		}
		body := rec.Result().Body
		defer body.Close()
		var res types.Res[db.CreatePsqlServiceRow]
		if err := readAndUnmarshl(body, &res); err != nil {
			t.Fatal(err)
		}
		psqlServiceID = res.Data.ID

		rec, err = TestEchoHandler(&TestEchoBody{
			T: t,
			H: h.Dependency.CreateServiceDependency,
			Body: &handlers.CreateDependencyReq{
				SourceServiceID: sourceServiceID,
				TargetServiceID: psqlServiceID,
				TargetCol:       "db_password",
				EnvKey:          "DB_PASSWORD",
			},
			IsAuth: true,
		})
		if err != nil {
			t.Fatal(err)
		}
		if rec.Code != http.StatusCreated {
			printRaw(rec.Result().Body, t)
			t.Fatalf("expected status code %d, got %d", http.StatusCreated, rec.Code)
		}
		body = rec.Result().Body
		defer body.Close()
		var depRes types.Res[handlers.ServiceDependencyRes]
		if err := readAndUnmarshl(body, &depRes); err != nil {
			t.Fatal(err)
		}
		if depRes.Data.TargetCol != "db_password" {
			t.Fatalf("expected target_col db_password, got %s", depRes.Data.TargetCol)
		}
	})

	t.Run("create dependency on redis service", func(t *testing.T) {
		// Create redis service via handler
		rec, err := TestEchoHandler(&TestEchoBody{
			T: t,
			H: h.Service.CreateRedisService,
			Body: &handlers.CreateRedisServiceReq{
				InstanceID: user.InstanceID,
				Name:       "test-redis",
				Password:   "redispass",
				Image:      "redis:7",
			},
			IsAuth: true,
		})
		if err != nil {
			t.Fatal(err)
		}
		if rec.Code != http.StatusOK {
			printRaw(rec.Result().Body, t)
			t.Fatalf("expected status code %d, got %d", http.StatusOK, rec.Code)
		}
		body := rec.Result().Body
		defer body.Close()
		var res types.Res[db.CreateRedisServiceRow]
		if err := readAndUnmarshl(body, &res); err != nil {
			t.Fatal(err)
		}
		redisServiceID = res.Data.ID

		rec, err = TestEchoHandler(&TestEchoBody{
			T: t,
			H: h.Dependency.CreateServiceDependency,
			Body: &handlers.CreateDependencyReq{
				SourceServiceID: sourceServiceID,
				TargetServiceID: redisServiceID,
				TargetCol:       "password",
				EnvKey:          "REDIS_PASSWORD",
			},
			IsAuth: true,
		})
		if err != nil {
			t.Fatal(err)
		}
		if rec.Code != http.StatusCreated {
			printRaw(rec.Result().Body, t)
			t.Fatalf("expected status code %d, got %d", http.StatusCreated, rec.Code)
		}
		body = rec.Result().Body
		defer body.Close()
		var depRes types.Res[handlers.ServiceDependencyRes]
		if err := readAndUnmarshl(body, &depRes); err != nil {
			t.Fatal(err)
		}
		if depRes.Data.TargetCol != "password" {
			t.Fatalf("expected target_col password, got %s", depRes.Data.TargetCol)
		}
	})

	t.Run("update dependency", func(t *testing.T) {
		// First get the existing dependency
		rec, err := TestEchoHandler(&TestEchoBody{
			T:      t,
			H:      h.Dependency.GetServiceDependencies,
			Query:  url.Values{"service_id": {sourceServiceID.String()}},
			IsAuth: true,
		})
		if err != nil {
			t.Fatal(err)
		}
		if rec.Code != http.StatusOK {
			t.Fatalf("expected status code %d, got %d", http.StatusOK, rec.Code)
		}
		body := rec.Result().Body
		defer body.Close()
		var listRes types.Res[handlers.ListDependenciesRes]
		if err := readAndUnmarshl(body, &listRes); err != nil {
			t.Fatal(err)
		}
		if len(listRes.Data.Dependencies) == 0 {
			t.Fatal("expected at least one dependency")
		}
		depID := listRes.Data.Dependencies[0].ID

		// Update the dependency
		params := echo.PathValues{}
		params = append(params, echo.PathValue{Name: "id", Value: depID.String()})

		rec, err = TestEchoHandler(&TestEchoBody{
			T:      t,
			H:      h.Dependency.UpdateServiceDependency,
			Body:   &handlers.UpdateDependencyReq{TargetServiceID: targetServiceID, TargetCol: "name", EnvKey: "UPDATED_NAME"},
			Params: params,
			IsAuth: true,
		})
		if err != nil {
			t.Fatal(err)
		}
		if rec.Code != http.StatusOK {
			printRaw(rec.Result().Body, t)
			t.Fatalf("expected status code %d, got %d", http.StatusOK, rec.Code)
		}
		body = rec.Result().Body
		defer body.Close()
		var updateRes types.Res[handlers.ServiceDependencyRes]
		if err := readAndUnmarshl(body, &updateRes); err != nil {
			t.Fatal(err)
		}
		if updateRes.Data.EnvKey != "UPDATED_NAME" {
			t.Fatalf("expected env_key UPDATED_NAME, got %s", updateRes.Data.EnvKey)
		}
		if updateRes.Data.TargetCol != "name" {
			t.Fatalf("expected target_col name, got %s", updateRes.Data.TargetCol)
		}
	})

	t.Run("delete dependency", func(t *testing.T) {
		// First get the existing dependency
		rec, err := TestEchoHandler(&TestEchoBody{
			T:      t,
			H:      h.Dependency.GetServiceDependencies,
			Query:  url.Values{"service_id": {sourceServiceID.String()}},
			IsAuth: true,
		})
		if err != nil {
			t.Fatal(err)
		}
		if rec.Code != http.StatusOK {
			t.Fatalf("expected status code %d, got %d", http.StatusOK, rec.Code)
		}
		body := rec.Result().Body
		defer body.Close()
		var listRes types.Res[handlers.ListDependenciesRes]
		if err := readAndUnmarshl(body, &listRes); err != nil {
			t.Fatal(err)
		}
		if len(listRes.Data.Dependencies) == 0 {
			t.Fatal("expected at least one dependency")
		}
		depID := listRes.Data.Dependencies[0].ID
		depCountBefore := len(listRes.Data.Dependencies)

		// Delete the dependency
		params := echo.PathValues{}
		params = append(params, echo.PathValue{Name: "id", Value: depID.String()})

		rec, err = TestEchoHandler(&TestEchoBody{
			T:      t,
			H:      h.Dependency.DeleteServiceDependency,
			Params: params,
			IsAuth: true,
		})
		if err != nil {
			t.Fatal(err)
		}
		if rec.Code != http.StatusOK {
			printRaw(rec.Result().Body, t)
			t.Fatalf("expected status code %d, got %d", http.StatusOK, rec.Code)
		}

		// Verify deletion
		rec, err = TestEchoHandler(&TestEchoBody{
			T:      t,
			H:      h.Dependency.GetServiceDependencies,
			Query:  url.Values{"service_id": {sourceServiceID.String()}},
			IsAuth: true,
		})
		if err != nil {
			t.Fatal(err)
		}
		if rec.Code != http.StatusOK {
			t.Fatalf("expected status code %d, got %d", http.StatusOK, rec.Code)
		}
		body = rec.Result().Body
		defer body.Close()
		if err := readAndUnmarshl(body, &listRes); err != nil {
			t.Fatal(err)
		}
		if len(listRes.Data.Dependencies) != depCountBefore-1 {
			t.Fatalf("expected %d dependencies after delete, got %d", depCountBefore-1, len(listRes.Data.Dependencies))
		}
	})

	t.Run("cleanup psql service", func(t *testing.T) {
		deleteReq := &handlers.DeletePsqlServiceReq{ServiceId: psqlServiceID, KeepData: false}
		rec, err := TestEchoHandler(&TestEchoBody{T: t, H: h.Service.DeletePsqlService, IsAuth: true, Body: deleteReq})
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
	})

	t.Run("cleanup redis service", func(t *testing.T) {
		deleteReq := &handlers.DeleteRedisServiceReq{ServiceId: redisServiceID, KeepData: false}
		rec, err := TestEchoHandler(&TestEchoBody{T: t, H: h.Service.DeleteRedisService, IsAuth: true, Body: deleteReq})
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
	})

	fmt.Println("dependency tests completed")
}

func containsStr(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}

func TestDependencyGraph(t *testing.T) {
	server, h, err := GetDummyServerHandler()
	if err != nil {
		t.Fatal(err)
	}

	user := mockUserRejister(h, t, true)
	instanceID := user.InstanceID

	createAppServiceReq := &handlers.CreateAppServiceReq{
		InstanceID:   instanceID,
		Name:         "graph-source",
		GitProvider:  "github",
		Public:       false,
		Port:         80,
		BuildPath:    "/",
		WatchPath:    "/",
		Env:          []string{},
		BuildArgs:    []string{},
		BuildSecrets: []string{},
		DockerBuild:  &handlers.DockerBuildReq{},
	}

	// Setup github app for service creation
	{
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
	}

	// Setup github repo for service creation
	{
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
	}

	rec, err := TestEchoHandler(&TestEchoBody{T: t, H: h.Service.CreateAppService, Body: createAppServiceReq, IsAuth: true})
	if err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusOK {
		printRaw(rec.Result().Body, t)
		t.Fatalf("expected status code %d, got %d", http.StatusOK, rec.Code)
	}
	body := rec.Result().Body
	defer body.Close()
	var sourceRes types.Res[db.CreateAppServiceRow]
	if err := readAndUnmarshl(body, &sourceRes); err != nil {
		t.Fatal(err)
	}
	sourceServiceID := sourceRes.Data.ID

	// Create target app service
	req2 := *createAppServiceReq
	req2.Name = "graph-target"
	rec, err = TestEchoHandler(&TestEchoBody{T: t, H: h.Service.CreateAppService, Body: &req2, IsAuth: true})
	if err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusOK {
		printRaw(rec.Result().Body, t)
		t.Fatalf("expected status code %d, got %d", http.StatusOK, rec.Code)
	}
	body = rec.Result().Body
	defer body.Close()
	var targetRes types.Res[db.CreateAppServiceRow]
	if err := readAndUnmarshl(body, &targetRes); err != nil {
		t.Fatal(err)
	}
	targetServiceID := targetRes.Data.ID

	// Create psql service
	rec, err = TestEchoHandler(&TestEchoBody{
		T: t,
		H: h.Service.CreatePsqlService,
		Body: &handlers.CreatePsqlServiceReq{
			InstanceID: instanceID,
			Name:       "graph-psql",
			DbName:     "testdb",
			DbUser:     "testuser",
			DbPassword: "testpass",
			Image:      "postgres:15",
		},
		IsAuth: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusOK {
		printRaw(rec.Result().Body, t)
		t.Fatalf("expected status code %d, got %d", http.StatusOK, rec.Code)
	}
	body = rec.Result().Body
	defer body.Close()
	var psqlRes types.Res[db.CreatePsqlServiceRow]
	if err := readAndUnmarshl(body, &psqlRes); err != nil {
		t.Fatal(err)
	}
	psqlServiceID := psqlRes.Data.ID

	// Create dependency source -> target
	rec, err = TestEchoHandler(&TestEchoBody{
		T: t,
		H: h.Dependency.CreateServiceDependency,
		Body: &handlers.CreateDependencyReq{
			SourceServiceID: sourceServiceID,
			TargetServiceID: targetServiceID,
			TargetCol:       "internal_url",
			EnvKey:          "TARGET_URL",
		},
		IsAuth: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusCreated {
		printRaw(rec.Result().Body, t)
		t.Fatalf("expected status code %d, got %d", http.StatusCreated, rec.Code)
	}

	// Create dependency source -> psql
	rec, err = TestEchoHandler(&TestEchoBody{
		T: t,
		H: h.Dependency.CreateServiceDependency,
		Body: &handlers.CreateDependencyReq{
			SourceServiceID: sourceServiceID,
			TargetServiceID: psqlServiceID,
			TargetCol:       "db_password",
			EnvKey:          "DB_PASSWORD",
		},
		IsAuth: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusCreated {
		printRaw(rec.Result().Body, t)
		t.Fatalf("expected status code %d, got %d", http.StatusCreated, rec.Code)
	}

	// Test graph endpoint
	t.Run("returns correct nodes and edges", func(t *testing.T) {
		params := echo.PathValues{}
		params = append(params, echo.PathValue{Name: "id", Value: instanceID.String()})

		rec, err := TestEchoHandler(&TestEchoBody{
			T:      t,
			H:      h.Instance.GetDependencyGraph,
			Params: params,
			IsAuth: true,
		})
		if err != nil {
			t.Fatal(err)
		}
		if rec.Code != http.StatusOK {
			printRaw(rec.Result().Body, t)
			t.Fatalf("expected status code %d, got %d", http.StatusOK, rec.Code)
		}
		body := rec.Result().Body
		defer body.Close()
		var graphRes types.Res[handlers.DependencyGraphRes]
		if err := readAndUnmarshl(body, &graphRes); err != nil {
			t.Fatal(err)
		}

		graph := graphRes.Data

		// Check nodes: should include source, target, psql
		if len(graph.Nodes) < 3 {
			t.Fatalf("expected at least 3 nodes, got %d", len(graph.Nodes))
		}

		nodeIDs := make(map[string]bool)
		for _, n := range graph.Nodes {
			nodeIDs[n.ID.String()] = true
		}
		if !nodeIDs[sourceServiceID.String()] {
			t.Fatal("graph missing source service node")
		}
		if !nodeIDs[targetServiceID.String()] {
			t.Fatal("graph missing target service node")
		}
		if !nodeIDs[psqlServiceID.String()] {
			t.Fatal("graph missing psql service node")
		}

		// Check edges: should include 2 dependencies
		if len(graph.Edges) != 2 {
			t.Fatalf("expected 2 edges, got %d", len(graph.Edges))
		}

		// Check edge details
		foundTargetURL := false
		foundDBPassword := false
		for _, e := range graph.Edges {
			if e.Source == sourceServiceID && e.Target == targetServiceID && e.EnvKey == "TARGET_URL" {
				foundTargetURL = true
			}
			if e.Source == sourceServiceID && e.Target == psqlServiceID && e.EnvKey == "DB_PASSWORD" {
				foundDBPassword = true
			}
		}
		if !foundTargetURL {
			t.Fatal("graph missing source->target edge")
		}
		if !foundDBPassword {
			t.Fatal("graph missing source->psql edge")
		}
	})

	t.Run("rejects invalid instance id", func(t *testing.T) {
		params := echo.PathValues{}
		params = append(params, echo.PathValue{Name: "id", Value: "invalid-uuid"})

		rec, err := TestEchoHandler(&TestEchoBody{
			T:      t,
			H:      h.Instance.GetDependencyGraph,
			Params: params,
			IsAuth: true,
		})
		if err != nil {
			t.Fatal(err)
		}
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status code %d, got %d", http.StatusBadRequest, rec.Code)
		}
	})

	t.Run("cleanup incoming dependencies on psql delete", func(t *testing.T) {
		// Create a new dependency on the psql service first
		rec, err := TestEchoHandler(&TestEchoBody{
			T: t,
			H: h.Dependency.CreateServiceDependency,
			Body: &handlers.CreateDependencyReq{
				SourceServiceID: sourceServiceID,
				TargetServiceID: psqlServiceID,
				TargetCol:       "db_name",
				EnvKey:          "DB_NAME_TEST",
			},
			IsAuth: true,
		})
		if err != nil {
			t.Fatal(err)
		}
		if rec.Code != http.StatusCreated {
			printRaw(rec.Result().Body, t)
			t.Fatalf("expected status code %d, got %d", http.StatusCreated, rec.Code)
		}

		// Verify dependency exists
		rec, err = TestEchoHandler(&TestEchoBody{
			T:      t,
			H:      h.Dependency.GetServiceDependencies,
			Query:  url.Values{"service_id": {sourceServiceID.String()}},
			IsAuth: true,
		})
		if err != nil {
			t.Fatal(err)
		}
		body := rec.Result().Body
		defer body.Close()
		var listRes types.Res[handlers.ListDependenciesRes]
		if err := readAndUnmarshl(body, &listRes); err != nil {
			t.Fatal(err)
		}
		found := false
		for _, d := range listRes.Data.Dependencies {
			if d.EnvKey == "DB_NAME_TEST" {
				found = true
				break
			}
		}
		if !found {
			t.Fatal("dependency on psql should exist before deletion")
		}

		// Delete the psql service
		rec, err = TestEchoHandler(&TestEchoBody{
			T: t,
			H: h.Service.DeletePsqlService,
			Body: &handlers.DeletePsqlServiceReq{
				ServiceId: psqlServiceID,
				KeepData:  false,
			},
			IsAuth: true,
		})
		if err != nil {
			t.Fatal(err)
		}
		if rec.Code != http.StatusOK {
			printRaw(rec.Result().Body, t)
			t.Fatalf("expected status code %d, got %d", http.StatusOK, rec.Code)
		}

		// Verify dependency is gone
		rec, err = TestEchoHandler(&TestEchoBody{
			T:      t,
			H:      h.Dependency.GetServiceDependencies,
			Query:  url.Values{"service_id": {sourceServiceID.String()}},
			IsAuth: true,
		})
		if err != nil {
			t.Fatal(err)
		}
		body = rec.Result().Body
		defer body.Close()
		if err := readAndUnmarshl(body, &listRes); err != nil {
			t.Fatal(err)
		}
		for _, d := range listRes.Data.Dependencies {
			if d.EnvKey == "DB_NAME_TEST" {
				t.Fatal("dependency should have been cleaned up after psql deletion")
			}
		}
	})

	t.Run("cleanup incoming dependencies on app target delete", func(t *testing.T) {
		// Delete the target app service — but first create a dependency on it
		rec, err := TestEchoHandler(&TestEchoBody{
			T: t,
			H: h.Dependency.CreateServiceDependency,
			Body: &handlers.CreateDependencyReq{
				SourceServiceID: sourceServiceID,
				TargetServiceID: targetServiceID,
				TargetCol:       "name",
				EnvKey:          "TARGET_NAME",
			},
			IsAuth: true,
		})
		if err != nil {
			t.Fatal(err)
		}
		if rec.Code != http.StatusCreated {
			printRaw(rec.Result().Body, t)
			t.Fatalf("expected status code %d, got %d", http.StatusCreated, rec.Code)
		}

		// We can't easily delete app service via handler because it expects deployments.
		// Instead, test via direct DB: delete the app service and check cascade.
		q := server.DB.Queries
		if err := q.DeleteAppService(context.Background(), targetServiceID); err != nil {
			t.Fatalf("failed to delete target app service: %v", err)
		}

		// Verify dependency is gone
		rec, err = TestEchoHandler(&TestEchoBody{
			T:      t,
			H:      h.Dependency.GetServiceDependencies,
			Query:  url.Values{"service_id": {sourceServiceID.String()}},
			IsAuth: true,
		})
		if err != nil {
			t.Fatal(err)
		}
		body := rec.Result().Body
		defer body.Close()
		var listRes types.Res[handlers.ListDependenciesRes]
		if err := readAndUnmarshl(body, &listRes); err != nil {
			t.Fatal(err)
		}
		for _, d := range listRes.Data.Dependencies {
			if d.TargetServiceID == targetServiceID {
				t.Fatal("incoming dependency on deleted app target should have been cleaned up")
			}
		}
	})
}
