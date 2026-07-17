package testing

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Roshan-anand/godploy/internal/handlers"
)

// testing middleware by sending req from dummy server
func TestUserLogin(t *testing.T) {

	registerReq := handlers.RegisterReq{
		Name:     "test",
		Email:    "test@test.com",
		Password: "testtest",
		OrgName:  "test_org",
	}

	// initialize mock server
	e, srv, err := mockServer()
	if err != nil {
		t.Fatal("err config server :", err)
	}

	// start test server
	ts := httptest.NewServer(e)
	// url, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatal("err parsing url:", err)
	}
	t.Cleanup(ts.Close)

	// create  new global client
	h, err := getNewClient()
	if err != nil {
		t.Fatal("err creating http client:", err)
	}

	t.Run("get org without login", func(t *testing.T) {
		r, err := h.Get(ts.URL + "/api/org")
		if err != nil {
			t.Fatal("err making request:", err)
		}

		if r.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected status code %d, got %d", http.StatusUnauthorized, r.StatusCode)
		}
	})

	t.Run("register user", func(t *testing.T) {
		r, err := h.Post(ts.URL+"/api/auth/register", "application/json", reqBody(registerReq))
		if err != nil {
			t.Fatal("err making request:", err)
		}

		if r.StatusCode != http.StatusOK {
			t.Fatalf("expected status code %d, got %d", http.StatusOK, r.StatusCode)
		}

		if !hasCookie(r.Cookies(), srv.Config) {
			t.Fatal("expected cookies not found in response")
		}
	})

	t.Run("get org after login", func(t *testing.T) {
		r, err := h.Get(ts.URL + "/api/org")
		if err != nil {
			t.Fatal("err making request:", err)
		}

		if r.StatusCode != http.StatusOK {
			t.Fatalf("expected status code %d, got %d", http.StatusOK, r.StatusCode)
		}
	})
}
