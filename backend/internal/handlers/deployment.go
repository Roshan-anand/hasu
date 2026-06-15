package handlers

import (
	"context"
	"log"
	"net/http"

	"github.com/Roshan-anand/godploy/internal/config"
	"github.com/Roshan-anand/godploy/internal/db"
	deployjob "github.com/Roshan-anand/godploy/internal/jobs/deployment"
	"github.com/Roshan-anand/godploy/internal/jobs/logbroker"
	"github.com/Roshan-anand/godploy/internal/lib/security"
	"github.com/Roshan-anand/godploy/internal/lib/sse"
	"github.com/Roshan-anand/godploy/internal/lib/types"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

type DeploymentHandler struct {
	Server   *config.Server
	Validate *validator.Validate
	qCtx     context.Context
}

func InitDeploymentHandlers(s *config.Server) *DeploymentHandler {
	return &DeploymentHandler{
		Server:   s,
		Validate: validator.New(),
		qCtx:     context.Background(),
	}
}

// get all service deployment jobs
//
// route: GET /api/service/deployment?service_id
func (h *DeploymentHandler) GetServiceDeployments(c *echo.Context) error {
	q := h.Server.DB.Queries

	// TODO : inlcude org_id to get all deployments of the org / service based on the query params.

	serviceID, err := uuid.Parse(c.QueryParam("service_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "invalid service_id"})
	}

	deployemnts, err := q.GetDeploymentsByServiceID(h.qCtx, serviceID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "failed to get deployments"})
	}

	return c.JSON(http.StatusOK, types.Res[[]db.GetDeploymentsByServiceIDRow]{
		Message: "",
		Data:    deployemnts,
	})
}

// delete service deployment by deployment id
//
// route: DELETE /api/service/deployment
func (h *DeploymentHandler) DeleteServiceDeployment(c *echo.Context) error {
	b := new(DeploymentReq)
	q := h.Server.DB.Queries

	if Res := BindAndValidate(b, c, h.Validate); Res != nil {
		return c.JSON(http.StatusBadRequest, Res)
	}

	dyp, err := q.GetDeploymentImgByID(h.qCtx, b.DeploymentID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to get deployment"})
	}

	if err := q.DeleteDeploymentByID(h.qCtx, b.DeploymentID); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "failed to delete deployment"})
	}

	h.Server.Docker.RemoveImages([]string{dyp.Image.String})
	h.Server.BadgerDB.DeleteAllLogsByDeploymentID([]uuid.UUID{b.DeploymentID})

	return c.JSON(http.StatusOK, types.Res[struct{}]{Message: "deployment deleted successfully"})
}

// subscribe to service deployment logs event
//
// route: GET /api/service/deployment/logs?deployment_id
func (h *DeploymentHandler) SubscribeServiceDeploymentLogs(c *echo.Context) error {
	q := h.Server.DB.Queries
	dID, err := uuid.Parse(c.QueryParam("deployment_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "invalid deployment_id"})
	}

	w := c.Response()
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	sse := sse.NewSSE(w)

	status, err := q.GetDeploymentStatus(h.qCtx, dID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "failed to get deployment status"})
	}

	userID := security.GeneratePrimaryKey()

	// if deployment is not in queued or building stage, then stream logs from badgerDB
	if status != types.DeploymentQueued && status != types.DeploymentBuilding {
		if err := h.Server.BadgerDB.StreamAllLogsByDeploymentID(dID, sse); err != nil {
			return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "failed to stream logs"})
		}
		return nil
	}

	l := h.Server.Services.LogBroker

	// subscribe to log broker queue to get real-time logs of the deployment
	l.Subscribe(userID, &logbroker.Subscriber{
		SSE:          sse,
		DeploymentID: dID,
	})

	for {
		select {
		case <-c.Request().Context().Done():
			log.Printf("SSE client disconnected, ip: %v", c.RealIP())
			l.Unsubscribe(userID)
			return nil
		}
	}
}

// rebuild app service
//
// route: POST /api/service/app/rebuild
func (h *DeploymentHandler) RebuildAppService(c *echo.Context) error {
	b := new(ServiceReq)
	q := h.Server.DB.Queries

	if Res := BindAndValidate(b, c, h.Validate); Res != nil {
		return c.JSON(http.StatusBadRequest, Res)
	}

	s, err := q.GetAppServiceForRebuild(h.qCtx, b.ServiceId)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to get branch"})
	}

	var newStatus types.DeploymentStatus
	switch s.DeploymentStatus {
	case types.DeploymentQueued, types.DeploymentBuilding:
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "Deployment is in progress, cannot rebuild now"})
	case types.DeploymentReady:
		newStatus = types.DeploymentInactive
	default:
		newStatus = types.DeploymentPruned
	}

	// start a new db transaction
	tx, err := h.Server.DB.Pool.BeginTx(context.Background(), nil)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to create service"})
	}
	tq := q.WithTx(tx)

	// update the previous deployment is_latest to false
	if err := tq.DownGradeDeployment(h.qCtx, db.DownGradeDeploymentParams{
		DeploymentID: s.DeploymentID,
		Status:       newStatus,
	}); err != nil {
		tx.Rollback()
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to create service"})
	}

	ghData, err := GetGitHubDeployData(tq, s.GhAppID, s.GhRepoID, s.Branch)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "failed to fetch github data"})
	}

	// create a new deployment for the app service
	dID, err := tq.CreateDeployment(h.qCtx, db.CreateDeploymentParams{
		ID:         security.GeneratePrimaryKey(),
		ServiceID:  s.ID,
		CommitHash: ghData.CommitHash,
		CommitMsg:  ghData.CommitMsg,
		IsCurrent:  true,
	})
	if err != nil {
		tx.Rollback()
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to create deployment"})
	}

	// used as unique image and service name
	unique := generateServiceAndImgName(s.Name, s.Branch)

	envStr, err := UnmarshalServiceEnv(&ServiceEnvByte{
		Env:          s.Env,
		BuildArgs:    s.BuildArgs,
		BuildSecrets: s.BuildSecrets,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to get branch domain"})
	}

	// end the db transaction and commit
	if err := tx.Commit(); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to create service"})
	}

	// push a new deployment job to the queue
	h.Server.Services.Deployment.Submit(context.Background(), &deployjob.DeploymentServiceParams{
		JobType:           deployjob.RebuildJob,
		DeploymentID:      dID,
		Token:             ghData.Token,
		Url:               s.GhRepoUrl,
		RepoType:          deployjob.RepoBranch,
		Branch:            s.Branch,
		SwarmService:      s.SwarmService,
		BuildPath:         s.BuildPath,
		DockerFilePath:    s.DockerFilepath,
		DockerContextPath: s.DockerContextpath,
		DockerBuildStage:  s.DockerBuildstage,
		ImgName:           unique.ImgName,
		Env:               envStr.Env,
		BuildArgs:         envStr.BuildArgs,
		BuildSecrets:      envStr.BuildSecrets,
	})

	return c.JSON(http.StatusOK, types.Res[string]{Message: "Successfully updated env", Data: s.Name})
}

// rollback app service to previous deployment
//
// route: POST /api/service/app/rollback
func (h *DeploymentHandler) RollbackAppService(c *echo.Context) error {
	b := new(RoolbackServiceReq)
	q := h.Server.DB.Queries

	if Res := BindAndValidate(b, c, h.Validate); Res != nil {
		return c.JSON(http.StatusBadRequest, Res)
	}

	// get all deployments
	deployments, err := q.GetDeploymentsByServiceID(h.qCtx, b.ServiceID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to get deployments"})
	}

	for i, d := range deployments {
		// check if there is a old deployment after an active deployment to rollback.
		if d.IsCurrent && i+1 < len(deployments) {

			if d.Status == types.DeploymentQueued || d.Status == types.DeploymentBuilding {
				return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "Deployment is in progress, cannot rollback now"})
			}

			tx, err := h.Server.DB.Pool.BeginTx(context.Background(), nil)
			if err != nil {
				return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to start rollback"})
			}
			tq := q.WithTx(tx)

			newDyp := deployments[i+1]

			// down grade the current deployment
			if err := tq.DownGradeDeployment(h.qCtx, db.DownGradeDeploymentParams{
				Status:       types.DeploymentInactive,
				DeploymentID: d.ID,
			}); err != nil {
				tx.Rollback()
				return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to start rollback"})
			}

			//upgrade the new deployment
			if err := tq.UpgradeDeployment(h.qCtx, db.UpgradeDeploymentParams{
				DeploymentID: newDyp.ID,
				Status:       types.DeploymentBuilding,
			}); err != nil {
				tx.Rollback()
				return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to start rollback"})
			}

			// check if deployment image exists
			if newDyp.Status != types.DeploymentPruned && newDyp.Image.Valid {

				h.Server.Services.Deployment.Submit(context.Background(), &deployjob.DeploymentServiceParams{
					JobType:      deployjob.ReDeployJob,
					DeploymentID: newDyp.ID,
					ImgName:      newDyp.Image.String,
					SwarmService: newDyp.SwarmService,
				})

			} else {
				// TODO : pull build deploy
			}

			if err := tx.Commit(); err != nil {
				return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to start rollback"})
			}

			return c.JSON(http.StatusOK, types.Res[struct{}]{Message: "successfully started rollback"})
		} else if d.IsCurrent {
			break
		}
	}

	return c.JSON(http.StatusNotImplemented, types.Res[struct{}]{Message: "old deployment not found to rollback"})
}
