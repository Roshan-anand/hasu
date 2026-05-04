package testing

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Roshan-anand/godploy/internal/db"
	"github.com/Roshan-anand/godploy/internal/handlers"
)

func TestOrgOperations(t *testing.T) {
	e, srv, err := mockConfigServer()
	if err != nil {
		t.Fatal("err config server :", err)
	}

	ts := httptest.NewServer(e)
	t.Cleanup(ts.Close)

	h, err := getNewClient()
	if err != nil {
		t.Fatal("err creating http client:", err)
	}

	user, err := mockUserRejister(ts.URL, h, srv.Config)
	if err != nil {
		t.Fatal(err)
	}
	CurrentOrgID := user.OrgId

	rOrg := "/api/org"

	t.Run("DELETE /org : returns 400 when trying to delete only org", func(t *testing.T) {
		deleteOrgReq := handlers.OrgReq{OrgID: CurrentOrgID}
		deleteReq, err := http.NewRequest(http.MethodDelete, ts.URL+rOrg, reqBody(deleteOrgReq))
		if err != nil {
			t.Fatal("err creating request:", err)
		}
		deleteReq.Header.Set("Content-Type", "application/json")

		r, err := h.Do(deleteReq)
		if err != nil {
			t.Fatal("err making request:", err)
		}
		defer r.Body.Close()

		if r.StatusCode != http.StatusConflict {
			t.Fatalf("expected status code %d, got %d", http.StatusConflict, r.StatusCode)
		}
	})

	t.Run("POST /org : returns 200 for creating new org", func(t *testing.T) {
		createOrgReq := handlers.CreateOrgReq{Name: "new_org"}
		r, err := h.Post(ts.URL+rOrg, "application/json", reqBody(createOrgReq))
		if err != nil {
			t.Fatal("err making request:", err)
		}
		defer r.Body.Close()

		if r.StatusCode != http.StatusOK {
			t.Fatalf("expected status code %d, got %d", http.StatusOK, r.StatusCode)
		}

		var res db.CreateOrgRow
		if err := readAndUnmarshl(r.Body, &res); err != nil {
			t.Fatal(err)
		}
		CurrentOrgID = res.ID
	})

	t.Run("GET /org : returns all orgs for the user", func(t *testing.T) {
		r, err := h.Get(ts.URL + rOrg)
		if err != nil {
			t.Fatal("err making request:", err)
		}
		defer r.Body.Close()

		if r.StatusCode != http.StatusOK {
			t.Fatalf("expected status code %d, got %d", http.StatusOK, r.StatusCode)
		}

		var orgs []map[string]interface{}
		if err := readAndUnmarshl(r.Body, &orgs); err != nil {
			t.Fatal(err)
		}

		if len(orgs) != 2 {
			t.Fatalf("expected 2 orgs, got %d", len(orgs))
		}
	})

	t.Run("PUT /org/switch : returns 200 for switching org", func(t *testing.T) {
		switchReq := handlers.OrgReq{OrgID: CurrentOrgID}

		r, err := h.Post(ts.URL+rOrg+"/switch", "application/json", reqBody(switchReq))
		if err != nil {
			t.Fatal("err making request:", err)
		}
		defer r.Body.Close()

		if r.StatusCode != http.StatusOK {
			t.Fatalf("expected status code %d, got %d", http.StatusOK, r.StatusCode)
		}
	})

	// t.Run("DELETE /org : returns 200 for deleting org (not the only one)", func(t *testing.T) {
	// 	deleteOrgReq := handlers.OrgReq{
	// 		OrgID: CurrentOrgID,
	// 	}

	// 	deleteReq, err := http.NewRequest(http.MethodDelete, ts.URL+rOrg, reqBody(deleteOrgReq))
	// 	if err != nil {
	// 		t.Fatal("err creating request:", err)
	// 	}
	// 	deleteReq.Header.Set("Content-Type", "application/json")

	// 	r, err := h.Do(deleteReq)
	// 	if err != nil {
	// 		t.Fatal("err making request:", err)
	// 	}
	// 	defer r.Body.Close()

	// 	if r.StatusCode != http.StatusOK {
	// 		t.Fatalf("expected status code %d, got %d", http.StatusOK, r.StatusCode)
	// 	}
	// })
}
