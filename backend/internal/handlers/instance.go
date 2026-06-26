package handlers

import (
	"context"
	"net/http"

	"github.com/Roshan-anand/godploy/internal/config"
	"github.com/Roshan-anand/godploy/internal/db"
	"github.com/Roshan-anand/godploy/internal/lib/types"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

type InstanceHandler struct {
	Server   *config.Server
	Validate *validator.Validate
	qCtx     context.Context
}

type GetAllInstanceRes struct {
	Instances []db.GetAllInstanceRow `json:"instances"`
	ProjectID uuid.UUID              `json:"project_id"`
}

type RenameInstanceReq struct {
	InstanceID uuid.UUID `json:"instance_id" validate:"required"`
	ProjectID  uuid.UUID `json:"project_id" validate:"required"`
	Name       string    `json:"name" validate:"required,min=3"`
}

type GraphNode struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	ServiceType string    `json:"service_type,omitempty"`
}

type GraphEdge struct {
	Source    uuid.UUID `json:"source"`
	Target    uuid.UUID `json:"target"`
	TargetCol string    `json:"target_col"`
	EnvKey    string    `json:"env_key"`
}

type DependencyGraphRes struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

func InitInstanceHandlers(s *config.Server) *InstanceHandler {
	return &InstanceHandler{
		Server:   s,
		Validate: validator.New(),
		qCtx:     context.Background(),
	}
}

// get all organizations accessible to the authenticated user
//
// route: GET /api/instance?project=&org_id=
func (h *InstanceHandler) GetAllInstance(c *echo.Context) error {
	q := h.Server.DB.Queries

	project := c.QueryParam("project")
	if project == "" {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{
			Message: "invalid project",
		})
	}

	orgID, err := uuid.Parse(c.QueryParam("org_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{
			Message: "invalid org_id",
		})
	}

	// get all instances for the project
	instances, err := q.GetAllInstance(h.qCtx, db.GetAllInstanceParams{
		OrganizationID: orgID,
		Project:        project,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{
			Message: "internal server error",
		})
	}

	projectID, err := q.GetProjectIDByName(h.qCtx, db.GetProjectIDByNameParams{
		OrganizationID: orgID,
		Name:           project,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{
			Message: "internal server error",
		})
	}

	return c.JSON(http.StatusOK, types.Res[GetAllInstanceRes]{
		Message: "",
		Data: GetAllInstanceRes{
			Instances: instances,
			ProjectID: projectID,
		},
	})
}

// rename a project instance
//
// route: PUT /api/instance/rename
func (h *InstanceHandler) RenameInstance(c *echo.Context) error {
	b := new(RenameInstanceReq)

	if Res := BindAndValidate(b, c, h.Validate); Res != nil {
		return c.JSON(http.StatusBadRequest, Res)
	}

	q := h.Server.DB.Queries

	instance, err := q.RenameInstance(h.qCtx, db.RenameInstanceParams{
		Name: b.Name,
		ID:   b.InstanceID,
	})
	if err != nil {
		if h.Server.DB.IsUniqueConstraintError(err) {
			return c.JSON(http.StatusConflict, types.Res[struct{}]{Message: "Instance with this name already exists in the project"})
		}
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to rename instance"})
	}

	return c.JSON(http.StatusOK, types.Res[db.RenameInstanceRow]{Message: "", Data: instance})
}

// get all services in an instance and their dependency edges
//
// route: GET /api/instance/:id/dependency-graph
func (h *InstanceHandler) GetDependencyGraph(c *echo.Context) error {
	q := h.Server.DB.Queries

	instanceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "invalid instance id"})
	}

	nodes, err := q.GetDependencyGraphNodes(h.qCtx, db.GetDependencyGraphNodesParams{
		InstanceID:   instanceID,
		InstanceID_2: instanceID,
		InstanceID_3: instanceID,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "failed to fetch graph nodes"})
	}

	edges, err := q.GetDependencyGraphEdges(h.qCtx, instanceID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "failed to fetch graph edges"})
	}

	res := DependencyGraphRes{
		Nodes: make([]GraphNode, len(nodes)),
		Edges: make([]GraphEdge, len(edges)),
	}

	for i, n := range nodes {
		res.Nodes[i] = GraphNode{
			ID:          n.ID,
			Name:        n.Name,
			Type:        n.ServiceType,
			ServiceType: n.ServiceType,
		}
	}

	for i, e := range edges {
		res.Edges[i] = GraphEdge{
			Source:    e.SourceServiceID,
			Target:    e.TargetServiceID,
			TargetCol: e.TargetCol,
			EnvKey:    e.EnvKey,
		}
	}

	return c.JSON(http.StatusOK, types.Res[DependencyGraphRes]{Data: res})
}
