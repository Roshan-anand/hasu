package testing

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/Roshan-anand/godploy/internal/db"
	"github.com/Roshan-anand/godploy/internal/handlers"
	"github.com/Roshan-anand/godploy/internal/lib/types"
	"github.com/google/uuid"
)

// AI: Exercise create/get/delete flows via the Echo handler harness to validate the PSQL service lifecycle.
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
		query := url.Values{}
		query.Add("service_id", psqlServiceID.String())
		rec, err := TestEchoHandler(&TestEchoBody{T: t, H: h.Service.GetPsqlServiceById, IsAuth: true, Query: query})
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
