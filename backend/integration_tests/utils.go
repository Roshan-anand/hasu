package testing

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/Roshan-anand/godploy/internal/config"
	"github.com/Roshan-anand/godploy/internal/db"
	"github.com/Roshan-anand/godploy/internal/handlers"
	"github.com/Roshan-anand/godploy/internal/lib/auth"
	"github.com/Roshan-anand/godploy/internal/lib/types"
	"github.com/Roshan-anand/godploy/internal/routes"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/echotest"
)

type TestEchoBody struct {
	T      *testing.T
	H      echo.HandlerFunc
	Body   any
	Query  url.Values
	Params echo.PathValues
	IsAuth bool
}

type MockUser struct {
	Name       string
	Email      string
	OrgId      uuid.UUID
	OrgName    string
	ProjectID  uuid.UUID
	InstanceID uuid.UUID
}

var registerBody = &handlers.RegisterReq{
	Email:    "test@email.com",
	Name:     "sample",
	Password: "testtest",
	OrgName:  "red",
}

// initialize a mock server for testing with config values suitable for testing
func mockServer() (*echo.Echo, *config.Server, error) {

	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load config: %w", err)
	}

	// update config to include testing data
	cfg.JwtSecret = "test_secret"
	cfg.AppEnv = types.TestMode

	tmpDir, err := getTempDir()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get temp dir: %w", err)
	}
	cfg.SqliteDir = tmpDir.sqliteDir
	cfg.BadgerDir = tmpDir.badgerDir
	cfg.CodeStoreDir = tmpDir.codeStoreDir

	// create server instance
	server, err := config.NewServer(cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize server: %w", err)
	}

	// setup all routes
	r, err := routes.SetupRoutes(server)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to setup routes: %w", err)
	}

	// server.SetupHttp(r) // setup http server with routes

	return r, server, nil
}

// get a new http client with cookie jar
func getNewClient() (*http.Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %w", err)
	}
	// create  new global client
	h := http.DefaultClient
	h.Jar = jar
	return h, nil
}

// reads the reader and unmarshal it
func readAndUnmarshl(body io.ReadCloser, v any) error {
	b, err := io.ReadAll(body)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(b, v); err != nil {
		return err
	}

	return nil
}

// reads the reader and unmarshal it
func readOnly(body io.ReadCloser) (string, error) {
	b, err := io.ReadAll(body)
	if err != nil {
		return "", err
	}

	return string(b), nil
}

func printRaw(body io.ReadCloser, t *testing.T) {
	msg, err := readOnly(body)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(msg)
}

// check if cookies exists
func hasCookie(c []*http.Cookie, cfg *config.Config) bool {
	for _, cookie := range c {
		switch cookie.Name {
		case cfg.SessionDataName, cfg.SessionTokenName:
		default:
			return false
		}
	}
	return true
}

// get a random free port
func getFreePort() (int, error) {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}
	defer l.Close()

	return l.Addr().(*net.TCPAddr).Port, nil
}

type TestDir struct {
	sqliteDir    string
	badgerDir    string
	codeStoreDir string
}

// get temp dir for testing
func getTempDir() (*TestDir, error) {
	p, err := os.MkdirTemp("", "godploy_test_*")
	if err != nil {
		return nil, err
	}

	sqliteDir := fmt.Sprintf("%s/sqlite", p)
	if err := os.Mkdir(sqliteDir, os.FileMode(0755)); err != nil {
		return nil, err
	}

	badgerDir := fmt.Sprintf("%s/badger", p)
	if err := os.Mkdir(badgerDir, os.FileMode(0755)); err != nil {
		return nil, err
	}

	codeStoreDir := fmt.Sprintf("%s/code", p)
	if err := os.Mkdir(codeStoreDir, os.FileMode(0755)); err != nil {
		return nil, err
	}

	return &TestDir{
		sqliteDir:    sqliteDir,
		badgerDir:    badgerDir,
		codeStoreDir: codeStoreDir,
	}, nil
}

func reqBody(data any) io.Reader {
	jsonData, _ := json.Marshal(data)
	return bytes.NewReader(jsonData)
}

func GetDummyServerHandler() (*config.Server, *handlers.Handler, error) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load config: %w", err)
	}

	// update config to include testing data
	cfg.JwtSecret = "test_secret"

	tmpDir, err := getTempDir()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get temp dir: %w", err)
	}
	cfg.SqliteDir = tmpDir.sqliteDir
	cfg.BadgerDir = tmpDir.badgerDir
	cfg.CodeStoreDir = tmpDir.codeStoreDir

	// create server instance
	server, err := config.NewServer(cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize server: %w", err)
	}

	server.Services.Deployment.Start(context.Background(), cfg.CodeStoreDir)
	server.Services.LogBroker.Start(context.Background())

	h := handlers.NewHandeler(server)

	return server, h, nil
}

func TestEchoHandler(te *TestEchoBody) (*httptest.ResponseRecorder, error) {

	config := echotest.ContextConfig{
		Headers: map[string][]string{
			echo.HeaderContentType: {echo.MIMEApplicationJSON},
		},
	}

	if te.Query != nil {
		config.QueryValues = te.Query
	}

	if te.Params != nil {
		config.PathValues = te.Params
	}

	if te.Body != nil {
		b, err := json.Marshal(te.Body)
		if err != nil {
			return nil, fmt.Errorf("error marshaling body : %v", err)
		}

		config.JSONBody = b
	}

	eCtx, rec := config.ToContextRecorder(te.T)

	if te.IsAuth {
		authUser := auth.AuthUser{
			Email: "test@email.com",
			Name:  "sample",
			Role:  types.AdminRole,
		}

		eCtx.Set("user_email", authUser)
	}

	if err := te.H(eCtx); err != nil {
		return nil, err
	}

	return rec, nil
}

// mock a new logined user
func mockUserRejister(h *handlers.Handler, t *testing.T, project bool) *MockUser {
	err := godotenv.Load("../../.env")
	if err != nil {
		t.Fatal(err)
	}

	mockUser := new(MockUser)

	rec, err := TestEchoHandler(&TestEchoBody{
		T:      t,
		H:      h.Auth.AppRegiter,
		Body:   registerBody,
		Query:  nil,
		IsAuth: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	body := rec.Result().Body
	defer body.Close()

	var userData types.Res[handlers.AuthRes]

	data, err := io.ReadAll(body)
	if err := json.Unmarshal(data, &userData); err != nil {
		t.Fatal(err)
	}

	mockUser.Name = userData.Data.Name
	mockUser.Email = userData.Data.Email
	mockUser.OrgId = userData.Data.OrgId
	mockUser.OrgName = userData.Data.OrgName

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status code %d, got %d", http.StatusUnauthorized, rec.Code)
	}

	// set a sample github app
	rec, err = TestEchoHandler(&TestEchoBody{
		T:      t,
		H:      h.Health.SetGhApp,
		Body:   &handlers.GhAppReq{Name: os.Getenv("GH_APP_NAME"), AppID: os.Getenv("GH_APP_ID"), OrgID: mockUser.OrgId, InstallationID: os.Getenv("GH_INSTALLATION_ID"), PemKey: os.Getenv("GH_PEM_KEY"), WebhookSecret: os.Getenv("WEBHOOK_SECRET")},
		Query:  nil,
		IsAuth: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status code %d, got %d", http.StatusUnauthorized, rec.Code)
	}

	// mock a sample project
	if project {
		rec, err = TestEchoHandler(&TestEchoBody{
			T:      t,
			H:      h.Project.CreateProject,
			Body:   &handlers.CreateProjectReq{Name: "newbe"},
			Query:  nil,
			IsAuth: true,
		})
		if err != nil {
			t.Fatal(err)
		}
		body = rec.Result().Body
		defer body.Close()

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status code %d, got %d", http.StatusOK, rec.Code)
		}

		var res types.Res[db.CreateProjectRow]
		if err := readAndUnmarshl(body, &res); err != nil {
			t.Fatal(err)
		}

		mockUser.ProjectID = res.Data.ID

		// get the instance
		query := url.Values{}
		query.Add("project", res.Data.Name)
		query.Add("org_id", mockUser.OrgId.String())
		rec, err = TestEchoHandler(&TestEchoBody{
			T:      t,
			H:      h.Instance.GetAllInstance,
			Body:   &handlers.CreateProjectReq{Name: "newbe"},
			Query:  query,
			IsAuth: true,
		})
		if err != nil {
			t.Fatal(err)
		}
		body = rec.Result().Body
		defer body.Close()

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status code %d, got %d", http.StatusOK, rec.Code)
		}

		var instanceRes types.Res[handlers.GetAllInstanceRes]
		if err := readAndUnmarshl(body, &instanceRes); err != nil {
			t.Fatal(err)
		}

		if len(instanceRes.Data.Instances) == 0 {
			t.Fatal("expected at least one instance, got 0")
		}

		mockUser.InstanceID = instanceRes.Data.Instances[0].ID
	}

	// mockUser.InstanceID = res.Data.InstanceID
	return mockUser
}
