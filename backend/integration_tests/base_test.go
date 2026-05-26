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

func TestOrgOperations(t *testing.T) {
	_, h, err := GetDummyServerHandler()
	if err != nil {
		t.Fatal(err)
	}

	user := mockUserRejister(h, t, false)
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
		rec, err := TestEchoHandler(&TestEchoBody{T: t, H: h.Org.DeleteOrg, Body: orgReqBody, IsAuth: true})
		if err != nil {
			t.Fatal(err)
		}

		if rec.Code != http.StatusConflict {
			t.Fatalf("expected status code %d, got %d", http.StatusConflict, rec.Code)
		}
	})

	t.Run("Create a new org", func(t *testing.T) {
		rec, err := TestEchoHandler(&TestEchoBody{T: t, H: h.Org.CreateOrg, Body: createOrgBody, IsAuth: true})
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
		rec, err := TestEchoHandler(&TestEchoBody{T: t, H: h.Org.CreateOrg, Body: createOrgBody, IsAuth: true})
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
		rec, err := TestEchoHandler(&TestEchoBody{T: t, H: h.Org.GetAllOrgs, IsAuth: true})
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
		rec, err := TestEchoHandler(&TestEchoBody{T: t, H: h.Org.SwitchOrg, Body: orgReqBody, IsAuth: true})
		if err != nil {
			t.Fatal(err)
		}

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status code %d, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("DELETE /org : returns 200 for deleting org (not the only one)", func(t *testing.T) {
		orgReqBody.OrgID = user.OrgId
		rec, err := TestEchoHandler(&TestEchoBody{T: t, H: h.Org.DeleteOrg, Body: orgReqBody, IsAuth: true})
		if err != nil {
			t.Fatal(err)
		}

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status code %d, got %d", http.StatusOK, rec.Code)
		}
	})
}

func TestProjectOperations(t *testing.T) {
	_, h, err := GetDummyServerHandler()
	if err != nil {
		t.Fatal(err)
	}

	user := mockUserRejister(h, t, false)
	if err != nil {
		t.Fatal(err)
	}

	projectReqBody := &handlers.ProjectReq{}

	createProjectBody := &handlers.CreateProjectReq{
		Name: "newbe",
	}

	t.Run("Create a new project", func(t *testing.T) {
		rec, err := TestEchoHandler(&TestEchoBody{T: t, H: h.Project.CreateProject, Body: createProjectBody, IsAuth: true})
		if err != nil {
			t.Fatal(err)
		}

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status code %d, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("Create a duplicate project", func(t *testing.T) {
		rec, err := TestEchoHandler(&TestEchoBody{T: t, H: h.Project.CreateProject, Body: createProjectBody, IsAuth: true})
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

	t.Run("Create second project", func(t *testing.T) {
		createProjectBody.Name = "veteran"
		rec, err := TestEchoHandler(&TestEchoBody{T: t, H: h.Project.CreateProject, Body: createProjectBody, IsAuth: true})
		if err != nil {
			t.Fatal(err)
		}
		body := rec.Result().Body
		defer body.Close()

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status code %d, got %d", http.StatusOK, rec.Code)
		}

		var res types.Res[db.CreateProjectRow]
		if err := readAndUnmarshl(body, &res); err != nil {
			t.Fatal(err)
		}
		projectReqBody.ProjectID = res.Data.ID
	})

	t.Run("get all project", func(t *testing.T) {
		query := url.Values{}
		query.Add("org_id", user.OrgId.String())
		rec, err := TestEchoHandler(&TestEchoBody{T: t, H: h.Project.GetAllProject, IsAuth: true, Query: query})
		if err != nil {
			t.Fatal(err)
		}
		body := rec.Result().Body
		defer body.Close()

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status code %d, got %d", http.StatusOK, rec.Code)
		}

		var res types.Res[[]db.Project]
		if err := readAndUnmarshl(body, &res); err != nil {
			t.Fatal(err)
		}

		if len(res.Data) != 2 {
			t.Fatalf("expected 2 projects, got %d", len(res.Data))
		}
	})

	t.Run("delete the project", func(t *testing.T) {
		rec, err := TestEchoHandler(&TestEchoBody{T: t, H: h.Project.DeleteProject, Body: projectReqBody, IsAuth: true})
		if err != nil {
			t.Fatal(err)
		}

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status code %d, got %d", http.StatusOK, rec.Code)
		}
	})
}
