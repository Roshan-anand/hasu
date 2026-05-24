package testing

import (
	"bytes"
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
	"github.com/Roshan-anand/godploy/internal/handlers"
	"github.com/Roshan-anand/godploy/internal/lib/auth"
	"github.com/Roshan-anand/godploy/internal/lib/types"
	"github.com/Roshan-anand/godploy/internal/routes"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/echotest"
)

// initialize a mock server for testing with config values suitable for testing
func mockServer() (*echo.Echo, *config.Server, error) {

	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load config: %w", err)
	}

	// update config to include testing data
	cfg.JwtSecret = "test_secret"
	cfg.AppEnv = types.TestMode

	sqliteTempPath, badgerTempPath, err := getTempDir()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get temp dir: %w", err)
	}
	cfg.SqliteDir = sqliteTempPath
	cfg.BadgerDir = badgerTempPath

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

// get temp dir for testing
func getTempDir() (string, string, error) {
	p, err := os.MkdirTemp("", "godploy_test_*")
	if err != nil {
		return "", "", err
	}

	sqliteDir := fmt.Sprintf("%s/sqlite", p)
	if err := os.Mkdir(sqliteDir, os.FileMode(0755)); err != nil {
		return "", "", err
	}

	badgerDir := fmt.Sprintf("%s/badger", p)
	if err := os.Mkdir(badgerDir, os.FileMode(0755)); err != nil {
		return "", "", err
	}

	return sqliteDir, badgerDir, nil
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

	sqliteTempPath, badgerTempPath, err := getTempDir()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get temp dir: %w", err)
	}
	cfg.SqliteDir = sqliteTempPath
	cfg.BadgerDir = badgerTempPath

	// create server instance
	server, err := config.NewServer(cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize server: %w", err)
	}

	h := handlers.NewHandeler(server)

	return server, h, nil
}

func TestEchoHandler(t *testing.T, h echo.HandlerFunc, body any, isAuth bool, query url.Values) (*httptest.ResponseRecorder, error) {

	config := echotest.ContextConfig{
		Headers: map[string][]string{
			echo.HeaderContentType: {echo.MIMEApplicationJSON},
		},
	}

	if query != nil {
		config.QueryValues = query
	}

	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("error marshaling body : %v", err)
		}

		config.JSONBody = b
	}

	eCtx, rec := config.ToContextRecorder(t)

	if isAuth {
		authUser := auth.AuthUser{
			Email: "test@email.com",
			Name:  "sample",
			Role:  types.AdminRole,
		}

		eCtx.Set("user_email", authUser)
	}

	if err := h(eCtx); err != nil {
		return nil, err
	}

	return rec, nil
}

var registerBody = &handlers.RegisterReq{
	Email:    "test@email.com",
	Name:     "sample",
	Password: "testtest",
	OrgName:  "red",
}

// mock a new logined user
func mockUserRejister(h *handlers.Handler, t *testing.T) handlers.AuthRes {
	rec, err := TestEchoHandler(t, h.Auth.AppRegiter, registerBody, false, nil)
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

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status code %d, got %d", http.StatusUnauthorized, rec.Code)
	}

	return userData.Data
}
