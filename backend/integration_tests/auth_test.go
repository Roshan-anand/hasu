package testing
 import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Roshan-anand/godploy/internal/handlers"
)

func TestUserLogin(t *testing.T) {
	// route paths
	rUser := "/api/auth/user"
	rLogin := "/api/auth/login"
	rRegister := "/api/auth/register"

	loginReq := handlers.LoginReq{
		Email:    "test@test.com",
		Password: "testtest",
	}

	registerReq := handlers.RegisterReq{
		Name:     "test",
		Email:    "test@test.com",
		Password: "testtest",
		OrgName:  "test_org",
	}

	// initialize mock server
	e, srv, err := mockConfigServer()
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

	t.Run("first time authenticating retusn 403 as no admin", func(t *testing.T) {
		r, err := h.Get(ts.URL + rUser)
		if err != nil {
			t.Fatal("err making request:", err)
		}

		if r.StatusCode != http.StatusForbidden {
			t.Fatalf("expected status code %d, got %d", http.StatusForbidden, r.StatusCode)
		}
	})

	t.Run("/register : returns 200 for valid register", func(t *testing.T) {
		r, err := h.Post(ts.URL+rRegister, "application/json", reqBody(registerReq))
		if err != nil {
			t.Fatal("err making request:", err)
		}

		if r.StatusCode != http.StatusOK {
			t.Fatalf("expected status code %d, got %d", http.StatusUnauthorized, r.StatusCode)
		}

		if !hasCookie(r.Cookies(), srv.Config) {
			t.Fatal("expected cookies not found in response")
		}
	})

	t.Run("/register : admin already login should return 400", func(t *testing.T) {
		r, err := h.Post(ts.URL+rRegister, "application/json", reqBody(registerReq))
		if err != nil {
			t.Fatal("err making request:", err)
		}

		if r.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected status code %d, got %d", http.StatusUnauthorized, r.StatusCode)
		}
	})

	t.Run("/user : returns 200 ", func(t *testing.T) {
		r, err := h.Get(ts.URL + rUser)
		if err != nil {
			t.Fatal("err making request:", err)
		}

		if r.StatusCode != http.StatusOK {
			t.Fatalf("expected status code %d, got %d", http.StatusOK, r.StatusCode)
		}
	})

	t.Run("/login : returns 200 for valid login", func(t *testing.T) {
		r, err := h.Post(ts.URL+rLogin, "application/json", reqBody(loginReq))
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

	t.Run("/login : returns 401 for invalid creadential", func(t *testing.T) {
		loginReq.Password = "qrong_pssword"
		r, err := h.Post(ts.URL+rLogin, "application/json", reqBody(loginReq))
		if err != nil {
			t.Fatal("err making request:", err)
		}

		if r.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected status code %d, got %d", http.StatusUnauthorized, r.StatusCode)
		}

		loginReq.Email = "no@email.com"
		r, err = h.Post(ts.URL+rLogin, "application/json", reqBody(loginReq))
		if err != nil {
			t.Fatal("err making request:", err)
		}

		if r.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected status code %d, got %d", http.StatusUnauthorized, r.StatusCode)
		}
	})
}
