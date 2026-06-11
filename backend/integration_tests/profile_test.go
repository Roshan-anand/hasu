package testing

import (
	"net/http"
	"testing"

	"github.com/Roshan-anand/godploy/internal/handlers"
	"github.com/Roshan-anand/godploy/internal/lib/types"
)

func TestProfileOperations(t *testing.T) {
	_, h, err := GetDummyServerHandler()
	if err != nil {
		t.Fatal(err)
	}

	user := mockUserRejister(h, t, false)

	t.Run("GET /api/profile returns user profile", func(t *testing.T) {
		rec, err := TestEchoHandler(&TestEchoBody{T: t, H: h.Profile.GetProfile, IsAuth: true})
		if err != nil {
			t.Fatal(err)
		}

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status code %d, got %d", http.StatusOK, rec.Code)
		}

		var res types.Res[handlers.ProfileRes]
		if err := readAndUnmarshl(rec.Result().Body, &res); err != nil {
			t.Fatal(err)
		}

		if res.Data.Name != user.Name {
			t.Fatalf("expected name %s, got %s", user.Name, res.Data.Name)
		}
		if res.Data.Email != user.Email {
			t.Fatalf("expected email %s, got %s", user.Email, res.Data.Email)
		}
	})

	t.Run("PUT /api/profile updates name and avatar", func(t *testing.T) {
		updateBody := &handlers.UpdateProfileReq{
			Name:   "Updated Name",
			Email:  "test@email.com",
			Avatar: "godploy-color.png",
		}

		rec, err := TestEchoHandler(&TestEchoBody{T: t, H: h.Profile.UpdateProfile, Body: updateBody, IsAuth: true})
		if err != nil {
			t.Fatal(err)
		}

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status code %d, got %d", http.StatusOK, rec.Code)
		}

		// verify the update persisted
		rec, err = TestEchoHandler(&TestEchoBody{T: t, H: h.Profile.GetProfile, IsAuth: true})
		if err != nil {
			t.Fatal(err)
		}

		var res types.Res[handlers.ProfileRes]
		if err := readAndUnmarshl(rec.Result().Body, &res); err != nil {
			t.Fatal(err)
		}

		if res.Data.Name != "Updated Name" {
			t.Fatalf("expected name 'Updated Name', got '%s'", res.Data.Name)
		}
		if res.Data.Avatar != "godploy-color.png" {
			t.Fatalf("expected avatar 'godploy-color.png', got '%s'", res.Data.Avatar)
		}
	})

	t.Run("PUT /api/profile/password changes password with valid old password", func(t *testing.T) {
		passwordBody := &handlers.ChangePasswordReq{
			OldPassword: "testtest",
			NewPassword: "newpass123",
		}

		rec, err := TestEchoHandler(&TestEchoBody{T: t, H: h.Profile.ChangePassword, Body: passwordBody, IsAuth: true})
		if err != nil {
			t.Fatal(err)
		}

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status code %d, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("PUT /api/profile/password rejects invalid old password", func(t *testing.T) {
		passwordBody := &handlers.ChangePasswordReq{
			OldPassword: "wrongpass",
			NewPassword: "anothernew456",
		}

		rec, err := TestEchoHandler(&TestEchoBody{T: t, H: h.Profile.ChangePassword, Body: passwordBody, IsAuth: true})
		if err != nil {
			t.Fatal(err)
		}

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected status code %d, got %d", http.StatusUnauthorized, rec.Code)
		}
	})

	t.Run("GET /api/profile requires authentication", func(t *testing.T) {
		rec, err := TestEchoHandler(&TestEchoBody{T: t, H: h.Profile.GetProfile, IsAuth: false})
		if err != nil {
			t.Fatal(err)
		}

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected status code %d, got %d", http.StatusUnauthorized, rec.Code)
		}
	})
}
