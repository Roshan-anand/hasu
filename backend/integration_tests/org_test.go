package testing

import (
	"net/http"
	"testing"

	"github.com/Roshan-anand/godploy/internal/db"
	"github.com/Roshan-anand/godploy/internal/handlers"
	"github.com/Roshan-anand/godploy/internal/lib/types"
	"github.com/google/uuid"
)

func TestOrgOperations(t *testing.T) {
	_, h, err := GetDummyServerHandler()
	if err != nil {
		t.Fatal(err)
	}

	user := mockUserRejister(h, t)
	if err != nil {
		t.Fatal(err)
	}

	var secOrg uuid.UUID

	orgReqBody := &handlers.OrgReq{
		OrgID: user.OrgId,
	}

	createOrgBody := &handlers.CreateOrgReq{
		Name: "origami",
	}

	t.Run("Delete the only org", func(t *testing.T) {
		rec, err := TestEchoHandler(t, h.Org.DeleteOrg, orgReqBody, true)
		if err != nil {
			t.Fatal(err)
		}

		if rec.Code != http.StatusConflict {
			t.Fatalf("expected status code %d, got %d", http.StatusConflict, rec.Code)
		}
	})

	t.Run("Create a new org", func(t *testing.T) {
		rec, err := TestEchoHandler(t, h.Org.CreateOrg, createOrgBody, true)
		if err != nil {
			t.Fatal(err)
		}
		body := rec.Result().Body
		defer body.Close()

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status code %d, got %d", http.StatusOK, rec.Code)
		}

		var res types.Res[db.CreateOrgRow]
		if err := readAndUnmarshl(body, &res); err != nil {
			t.Fatal(err)
		}
		secOrg = res.Data.ID
	})

	t.Run("Create a duplicate org", func(t *testing.T) {
		rec, err := TestEchoHandler(t, h.Org.CreateOrg, createOrgBody, true)
		if err != nil {
			t.Fatal(err)
		}
		body := rec.Result().Body
		defer body.Close()

		var res types.Res[any]
		if err := readAndUnmarshl(body, &res); err != nil {
			t.Fatal(err)
		}

		if rec.Code != http.StatusConflict {
			t.Fatalf("expected status code %d, got %d \n msg: %s", http.StatusConflict, rec.Code, res.Message)
		}
	})

	t.Run("get all orgs", func(t *testing.T) {
		rec, err := TestEchoHandler(t, h.Org.GetAllOrgs, nil, true)
		if err != nil {
			t.Fatal(err)
		}
		body := rec.Result().Body
		defer body.Close()

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status code %d, got %d", http.StatusOK, rec.Code)
		}

		var res types.Res[[]db.GetAllOrgRow]
		if err := readAndUnmarshl(body, &res); err != nil {
			t.Fatal(err)
		}

		if len(res.Data) != 2 {
			t.Fatalf("expected 2 orgs, got %d", len(res.Data))
		}
	})

	t.Run("switch org", func(t *testing.T) {
		orgReqBody.OrgID = secOrg
		rec, err := TestEchoHandler(t, h.Org.SwitchOrg, orgReqBody, true)
		if err != nil {
			t.Fatal(err)
		}

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status code %d, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("DELETE /org : returns 200 for deleting org (not the only one)", func(t *testing.T) {
		orgReqBody.OrgID = user.OrgId
		rec, err := TestEchoHandler(t, h.Org.DeleteOrg, orgReqBody, true)
		if err != nil {
			t.Fatal(err)
		}

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status code %d, got %d", http.StatusOK, rec.Code)
		}
	})
}
