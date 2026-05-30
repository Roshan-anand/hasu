package testing

import (
	"net/http"
	"testing"

	"github.com/Roshan-anand/godploy/internal/db"
	"github.com/Roshan-anand/godploy/internal/handlers"
	"github.com/Roshan-anand/godploy/internal/lib/types"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

func TestPsqlService(t *testing.T) {
	_, h, err := GetDummyServerHandler()
	if err != nil {
		t.Fatal(err)
	}

	user := mockUserRejister(h, t, true)
	if err != nil {
		t.Fatal(err)
	}

	createPsqlServiceReq := &handlers.CreatePsqlServiceReq{
		ProjectID:  user.ProjectID,
		Name:       "newpsql",
		DbName:     "testdb",
		DbUser:     "testuser",
		DbPassword: "testpass",
		Image:      "postgres:16-alpine",
	}

	updatePsqlServiceReq := &handlers.UpdatePsqlServiceReq{
		DbName:     "updated_db",
		DbUser:     "updated_user",
		DbPassword: "updated_pass",
	}

	var psqlServiceID uuid.UUID

	t.Run("create psql service and get id", func(t *testing.T) {
		rec, err := TestEchoHandler(&TestEchoBody{T: t, H: h.Service.CreatePsqlService, IsAuth: true, Body: createPsqlServiceReq})
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

		if res.Data == uuid.Nil {
			t.Fatal("expected non-empty service id")
		}
		psqlServiceID = res.Data
	})

	t.Run("create duplicate psql service", func(t *testing.T) {
		rec, err := TestEchoHandler(&TestEchoBody{T: t, H: h.Service.CreatePsqlService, IsAuth: true, Body: createPsqlServiceReq})
		if err != nil {
			t.Fatal(err)
		}

		if rec.Code != http.StatusConflict {
			t.Fatalf("expected status code %d, got %d", http.StatusConflict, rec.Code)
		}
	})

	t.Run("create psql service with invalid image", func(t *testing.T) {
		createPsqlServiceReq.Image = "nginx:alpine"
		rec, err := TestEchoHandler(&TestEchoBody{T: t, H: h.Service.CreatePsqlService, IsAuth: true, Body: createPsqlServiceReq})
		if err != nil {
			t.Fatal(err)
		}

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status code %d, got %d", http.StatusBadRequest, rec.Code)
		}
	})

	t.Run("get the psql service details", func(t *testing.T) {
		params := echo.PathValues{}
		params = append(params, echo.PathValue{Name: "id", Value: psqlServiceID.String()})

		rec, err := TestEchoHandler(&TestEchoBody{T: t, H: h.Service.GetPsqlServiceById, IsAuth: true, Params: params})
		if err != nil {
			t.Fatal(err)
		}
		body := rec.Result().Body
		defer body.Close()

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status code %d, got %d", http.StatusOK, rec.Code)
		}

		var res types.Res[db.PsqlService]
		if err := readAndUnmarshl(body, &res); err != nil {
			t.Fatal(err)
		}

		if res.Data.ID != psqlServiceID {
			t.Fatalf("expected service id %s, got %s", psqlServiceID, res.Data.ID)
		}
	})

	t.Run("update psql service details", func(t *testing.T) {
		updatePsqlServiceReq.ServiceID = psqlServiceID
		rec, err := TestEchoHandler(&TestEchoBody{T: t, H: h.Service.UpdatePsqlServiceDetails, IsAuth: true, Body: updatePsqlServiceReq})
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

		params := echo.PathValues{}
		params = append(params, echo.PathValue{Name: "id", Value: psqlServiceID.String()})

		rec, err = TestEchoHandler(&TestEchoBody{T: t, H: h.Service.GetPsqlServiceById, IsAuth: true, Params: params})
		if err != nil {
			t.Fatal(err)
		}
		body = rec.Result().Body
		defer body.Close()

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status code %d, got %d", http.StatusOK, rec.Code)
		}

		var res types.Res[db.PsqlService]
		if err := readAndUnmarshl(body, &res); err != nil {
			t.Fatal(err)
		}

		if res.Data.DbName != updatePsqlServiceReq.DbName {
			t.Fatalf("expected db name %s, got %s", updatePsqlServiceReq.DbName, res.Data.DbName)
		}
		if res.Data.DbUser != updatePsqlServiceReq.DbUser {
			t.Fatalf("expected db user %s, got %s", updatePsqlServiceReq.DbUser, res.Data.DbUser)
		}
		if res.Data.DbPassword != updatePsqlServiceReq.DbPassword {
			t.Fatalf("expected db password %s, got %s", updatePsqlServiceReq.DbPassword, res.Data.DbPassword)
		}
	})

	t.Run("redeploy psql service", func(t *testing.T) {
		redeployReq := &handlers.ServiceReq{ServiceId: psqlServiceID}
		rec, err := TestEchoHandler(&TestEchoBody{T: t, H: h.Service.RedeployPsqlService, IsAuth: true, Body: redeployReq})
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

	t.Run("delete psql service", func(t *testing.T) {
		deleteReq := &handlers.ServiceReq{ServiceId: psqlServiceID}
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
}
