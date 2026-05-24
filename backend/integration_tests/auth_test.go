package testing

import (
	"net/http"
	"testing"

	"github.com/Roshan-anand/godploy/internal/handlers"
)

func TestSample(t *testing.T) {
	srv, h, err := GetDummyServerHandler()
	if err != nil {
		t.Fatal(err)
	}

	loginBody := &handlers.LoginReq{
		Email:    "test@email.com",
		Password: "testtest",
	}

	t.Run("first time authenticating retusn 403 as no admin", func(t *testing.T) {
		rec, err := TestEchoHandler(t, h.Auth.AuthUser, nil, false, nil)
		if err != nil {
			t.Fatal(err)
		}

		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected status code %d, got %d", http.StatusForbidden, rec.Code)
		}
	})

	t.Run("register user", func(t *testing.T) {
		rec, err := TestEchoHandler(t, h.Auth.AppRegiter, registerBody, false, nil)
		if err != nil {
			t.Fatal(err)
		}

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status code %d, got %d", http.StatusUnauthorized, rec.Code)
		}

		if !hasCookie(rec.Result().Cookies(), srv.Config) {
			t.Fatal("expected cookies not found in response")
		}
	})

	t.Run("register again as admin", func(t *testing.T) {
		rec, err := TestEchoHandler(t, h.Auth.AppRegiter, registerBody, false, nil)
		if err != nil {
			t.Fatal(err)
		}

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status code %d, got %d", http.StatusBadRequest, rec.Code)
		}
	})

	t.Run("get auth user", func(t *testing.T) {
		rec, err := TestEchoHandler(t, h.Auth.AuthUser, nil, true, nil)
		if err != nil {
			t.Fatal(err)
		}

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status code %d, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("login user", func(t *testing.T) {
		rec, err := TestEchoHandler(t, h.Auth.AppLogin, loginBody, false, nil)
		if err != nil {
			t.Fatal(err)
		}

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status code %d, got %d", http.StatusOK, rec.Code)
		}

		if !hasCookie(rec.Result().Cookies(), srv.Config) {
			t.Fatal("expected cookies not found in response")
		}
	})

	t.Run("login with invalid creadential", func(t *testing.T) {
		loginBody.Password = "wrong_pssword"
		rec, err := TestEchoHandler(t, h.Auth.AppLogin, loginBody, false, nil)
		if err != nil {
			t.Fatal(err)
		}

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected status code %d, got %d", http.StatusUnauthorized, rec.Code)
		}

		loginBody.Email = "no@email.com"
		rec, err = TestEchoHandler(t, h.Auth.AppLogin, loginBody, false, nil)
		if err != nil {
			t.Fatal(err)
		}

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected status code %d, got %d", http.StatusUnauthorized, rec.Code)
		}
	})
}
