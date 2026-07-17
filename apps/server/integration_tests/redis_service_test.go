package testing

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/Roshan-anand/godploy/internal/db"
	"github.com/Roshan-anand/godploy/internal/handlers"
	"github.com/Roshan-anand/godploy/internal/lib/types"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

func TestRedisService(t *testing.T) {

	_, h, err := GetDummyServerHandler()
	if err != nil {
		t.Fatal(err)
	}

	user := mockUserRejister(h, t, true)
	if err != nil {
		t.Fatal(err)
	}

	createRedisServiceReq := &handlers.CreateRedisServiceReq{
		InstanceID: user.InstanceID,
		Name:       "newredis",
		Password:   "testpass",
		Image:      "redis:7-alpine",
	}

	updateRedisServiceReq := &handlers.UpdateRedisServiceReq{
		Password: "updated_pass",
	}

	var redisServiceID uuid.UUID
	var orphanVolume string

	t.Run("create redis service and get id", func(t *testing.T) {
		rec, err := TestEchoHandler(&TestEchoBody{T: t, H: h.Service.CreateRedisService, IsAuth: true, Body: createRedisServiceReq})
		if err != nil {
			t.Fatal(err)
		}
		body := rec.Result().Body
		defer body.Close()

		if rec.Code != http.StatusOK {
			printRaw(body, t)
			t.Fatalf("expected status code %d, got %d", http.StatusOK, rec.Code)
		}

		var res types.Res[db.CreateRedisServiceRow]
		if err := readAndUnmarshl(body, &res); err != nil {
			t.Fatal(err)
		}

		if res.Data.Name == "" {
			t.Fatal("expected non-empty service name")
		}
		redisServiceID = res.Data.ID
	})

	t.Run("create duplicate redis service", func(t *testing.T) {
		rec, err := TestEchoHandler(&TestEchoBody{T: t, H: h.Service.CreateRedisService, IsAuth: true, Body: createRedisServiceReq})
		if err != nil {
			t.Fatal(err)
		}

		if rec.Code != http.StatusConflict {
			t.Fatalf("expected status code %d, got %d", http.StatusConflict, rec.Code)
		}
	})

	t.Run("create redis service with invalid image", func(t *testing.T) {
		createRedisServiceReq.Image = "nginx:alpine"
		rec, err := TestEchoHandler(&TestEchoBody{T: t, H: h.Service.CreateRedisService, IsAuth: true, Body: createRedisServiceReq})
		if err != nil {
			t.Fatal(err)
		}

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status code %d, got %d", http.StatusBadRequest, rec.Code)
		}
	})

	t.Run("get the redis service details", func(t *testing.T) {
		params := echo.PathValues{}
		params = append(params, echo.PathValue{Name: "id", Value: redisServiceID.String()})

		rec, err := TestEchoHandler(&TestEchoBody{T: t, H: h.Service.GetRedisServiceById, IsAuth: true, Params: params})
		if err != nil {
			t.Fatal(err)
		}
		body := rec.Result().Body
		defer body.Close()

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status code %d, got %d", http.StatusOK, rec.Code)
		}

		var res types.Res[db.RedisService]
		if err := readAndUnmarshl(body, &res); err != nil {
			t.Fatal(err)
		}

		if res.Data.ID != redisServiceID {
			t.Fatalf("expected service id %s, got %s", redisServiceID, res.Data.ID)
		}
	})

	t.Run("update redis service details", func(t *testing.T) {
		updateRedisServiceReq.ServiceID = redisServiceID
		rec, err := TestEchoHandler(&TestEchoBody{T: t, H: h.Service.UpdateRedisServiceDetails, IsAuth: true, Body: updateRedisServiceReq})
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
		params = append(params, echo.PathValue{Name: "id", Value: redisServiceID.String()})

		rec, err = TestEchoHandler(&TestEchoBody{T: t, H: h.Service.GetRedisServiceById, IsAuth: true, Params: params})
		if err != nil {
			t.Fatal(err)
		}
		body = rec.Result().Body
		defer body.Close()

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status code %d, got %d", http.StatusOK, rec.Code)
		}

		var res types.Res[db.RedisService]
		if err := readAndUnmarshl(body, &res); err != nil {
			t.Fatal(err)
		}

		if res.Data.Password != updateRedisServiceReq.Password {
			t.Fatalf("expected password %s, got %s", updateRedisServiceReq.Password, res.Data.Password)
		}
	})

	t.Run("redeploy redis service", func(t *testing.T) {
		redeployReq := &handlers.ServiceReq{ServiceId: redisServiceID}
		rec, err := TestEchoHandler(&TestEchoBody{T: t, H: h.Service.RedeployRedisService, IsAuth: true, Body: redeployReq})
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

	t.Run("delete redis service and dont keep the data", func(t *testing.T) {
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

	t.Run("create redis service for keep data flow", func(t *testing.T) {
		keepDataReq := &handlers.CreateRedisServiceReq{
			InstanceID: user.InstanceID,
			Name:       "newredis-keep",
			Password:   "keeppass",
			Image:      "redis:7-alpine",
		}

		rec, err := TestEchoHandler(&TestEchoBody{T: t, H: h.Service.CreateRedisService, IsAuth: true, Body: keepDataReq})
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

		var res types.Res[db.CreateRedisServiceRow]
		if err := readAndUnmarshl(body, &res); err != nil {
			t.Fatal(err)
		}

		redisServiceID = res.Data.ID
	})

	t.Run("delete redis service and keep data", func(t *testing.T) {
		params := echo.PathValues{}
		params = append(params, echo.PathValue{Name: "id", Value: redisServiceID.String()})

		rec, err := TestEchoHandler(&TestEchoBody{T: t, H: h.Service.GetRedisServiceById, IsAuth: true, Params: params})
		if err != nil {
			t.Fatal(err)
		}
		body := rec.Result().Body
		defer body.Close()

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status code %d, got %d", http.StatusOK, rec.Code)
		}

		var getRes types.Res[db.RedisService]
		if err := readAndUnmarshl(body, &getRes); err != nil {
			t.Fatal(err)
		}
		orphanVolume = getRes.Data.Volume

		deleteReq := &handlers.DeleteRedisServiceReq{ServiceId: redisServiceID, KeepData: true}
		rec, err = TestEchoHandler(&TestEchoBody{T: t, H: h.Service.DeleteRedisService, IsAuth: true, Body: deleteReq})
		if err != nil {
			t.Fatal(err)
		}
		body = rec.Result().Body
		defer body.Close()

		if rec.Code != http.StatusOK {
			printRaw(body, t)
			t.Fatalf("expected status code %d, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("get orphan volume after keeping redis data", func(t *testing.T) {
		query := url.Values{}
		query.Add("org_id", user.OrgId.String())
		rec, err := TestEchoHandler(&TestEchoBody{T: t, H: h.Service.GetAllVolume, IsAuth: true, Query: query})
		if err != nil {
			t.Fatal(err)
		}
		body := rec.Result().Body
		defer body.Close()

		if rec.Code != http.StatusOK {
			printRaw(body, t)
			t.Fatalf("expected status code %d, got %d", http.StatusOK, rec.Code)
		}

		var res types.Res[[]db.OrphanVolume]
		if err := readAndUnmarshl(body, &res); err != nil {
			t.Fatal(err)
		}

		found := false
		for _, v := range res.Data {
			if v.Volume == orphanVolume {
				found = true
				break
			}
		}

		if !found {
			t.Fatalf("expected orphan volume %s to be present", orphanVolume)
		}
	})

	t.Run("delete orphan volume", func(t *testing.T) {
		deleteReq := &handlers.DeleteVolumeReq{Volumes: []string{orphanVolume}}
		rec, err := TestEchoHandler(&TestEchoBody{T: t, H: h.Service.DeleteVolume, IsAuth: true, Body: deleteReq})
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
