package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/Roshan-anand/godploy/internal/db"
	deploymentqueue "github.com/Roshan-anand/godploy/internal/jobs/deployment/queue"
	logbrokerqueue "github.com/Roshan-anand/godploy/internal/jobs/logbroker/queue"
	"github.com/Roshan-anand/godploy/internal/lib/gh"
	"github.com/Roshan-anand/godploy/internal/lib/security"
	"github.com/Roshan-anand/godploy/internal/lib/sse"
	"github.com/Roshan-anand/godploy/internal/lib/types"
	"github.com/google/go-github/v84/github"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

// get all service deployment jobs
//
// route: GET /api/service/deployment?service_id
func (h *ServiceHandler) GetServiceDeployments(c *echo.Context) error {
	q := h.Server.DB.Queries

	// TODO : inlcude org_id to get all deployments of the org / service based on the query params.

	serviceID, err := uuid.Parse(c.QueryParam("service_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "invalid service_id"})
	}

	deployemnts, err := q.GetDeploymentsByServiceID(h.qCtx, serviceID)
	if err != nil {
		fmt.Printf("error getting deployments for service_id: %v, error: %v\n", serviceID, err)
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
func (h *ServiceHandler) DeleteServiceDeployment(c *echo.Context) error {
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

	h.Server.Docker.RemoveImages([]string{dyp.ImageName.String})
	h.Server.BadgerDB.DeleteAllLogsByDeploymentID([]uuid.UUID{b.DeploymentID})

	return c.JSON(http.StatusOK, types.Res[struct{}]{Message: "deployment deleted successfully"})
}

// subscribe to service deployment logs event
//
// route: GET /api/service/deployment/logs?deployment_id
func (h *ServiceHandler) SubscribeServiceDeploymentLogs(c *echo.Context) error {
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

	l := h.Server.LogBrokerQ

	// subscribe to log broker queue to get real-time logs of the deployment
	l.SubscribeLogs(userID, &logbrokerqueue.Subscriber{
		SSE:          sse,
		DeploymentID: dID,
	})

	for {
		select {
		case <-c.Request().Context().Done():
			log.Printf("SSE client disconnected, ip: %v", c.RealIP())
			l.UnsubscribeLogs(userID)
			return nil
		}
	}
}

// rebuild app service
//
// route: POST /api/service/app/rebuild
func (h *ServiceHandler) RebuildAppService(c *echo.Context) error {
	b := new(RebuildServiceReq)
	q := h.Server.DB.Queries

	if Res := BindAndValidate(b, c, h.Validate); Res != nil {
		return c.JSON(http.StatusBadRequest, Res)
	}

	service, err := q.GetAppServiceByBranchId(h.qCtx, b.BranchID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to get branch"})
	}

	var newStatus types.DeploymentStatus
	switch service.DeploymentStatus {
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
	q = q.WithTx(tx)

	// update the previous deployment is_latest to false
	if err := q.DownGradeDeployment(h.qCtx, db.DownGradeDeploymentParams{
		DeploymentID: service.DeploymentID,
		Status:       newStatus,
	}); err != nil {
		tx.Rollback()
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to create service"})
	}

	// get the github app details
	ghApp, err := q.GetGhAppByAppId(h.qCtx, service.GhAppID)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "invalid github app"})
		}
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "failed to verify github app"})
	}

	// get teh gh client
	ghClient, err := gh.CreateGithubClient(context.Background(), ghApp.AppID, ghApp.InstallationID.Int64, ghApp.PemKey)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "failed to create github client"})
	}

	// get the github repository details
	repo, _, err := ghClient.Repositories.GetByID(context.Background(), service.GhRepoID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "failed to fetch repository info from github"})
	}

	owner := repo.GetOwner().GetLogin()
	repoShortName := repo.GetName()

	// get the latest commit info of the default branch
	commits, _, err := ghClient.Repositories.ListCommits(context.Background(), owner, repoShortName, &github.CommitsListOptions{
		SHA: service.BranchName,
		ListOptions: github.ListOptions{
			PerPage: 1,
			Page:    1,
		},
	})
	if err != nil || len(commits) == 0 {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "failed to fetch latest commit info from github"})
	}

	latestCommitHash := commits[0].GetSHA()
	latestCommitMsg := commits[0].GetCommit().GetMessage()

	// TODO : get commit msg from client side
	// create a new deployment for the app service
	dID, err := q.CreateDeployment(h.qCtx, db.CreateDeploymentParams{
		ID:         security.GeneratePrimaryKey(),
		BranchID:   service.BranchID,
		CommitMsg:  latestCommitMsg,
		CommitHash: latestCommitHash,
		IsCurrent:  true,
	})
	if err != nil {
		tx.Rollback()
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to create deployment"})
	}

	// get gh token
	token, err := gh.GetGhToken(ghApp.AppID, ghApp.InstallationID.Int64, ghApp.PemKey)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to get github token"})
	}

	// used as uniquecontainer name and code storing path
	imgName := strings.ToLower(fmt.Sprintf("%s-%s-dyp_%s", service.Name, service.BranchName, security.GenerateRandomID(6)))

	envStr, err := UnmarshalServiceEnv(&ServiceEnvByte{
		Env:          service.Env,
		BuildArgs:    service.BuildArgs,
		BuildSecrets: service.BuildSecrets,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to get branch domain"})
	}

	// end the db transaction and commit
	if err := tx.Commit(); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to create service"})
	}

	// push a new deployment job to the queue
	h.Server.DeploymentQ.EnqueuePullJob(&deploymentqueue.PullJobData{
		Type:              deploymentqueue.RebuildJob,
		DeploymentID:      dID,
		Token:             token,
		Url:               service.GhRepoUrl,
		Branch:            service.BranchName,
		SwarmServiceName:  service.SwarmServiceName,
		BuildPath:         service.BuildPath,
		DockerFilePath:    service.DockerFilepath,
		DockerContextPath: service.DockerContextpath,
		DockerBuildStage:  service.DockerBuildstage,
		ImgName:           imgName,
		Env:               envStr.Env,
		BuildArgs:         envStr.BuildArgs,
		BuildSecrets:      envStr.BuildSecrets,
	})

	return c.JSON(http.StatusOK, types.Res[uuid.UUID]{Message: "Successfully updated env", Data: service.ServiceID})
}

// rollback app service to previous deployment
//
// route: POST /api/service/app/rollback
func (h *ServiceHandler) RollbackAppService(c *echo.Context) error {
	b := new(RoolbackServiceReq)
	q := h.Server.DB.Queries

	if Res := BindAndValidate(b, c, h.Validate); Res != nil {
		return c.JSON(http.StatusBadRequest, Res)
	}

	// get all deployments
	deployments, err := q.GetDeploymentsByBranchID(h.qCtx, b.BranchID)
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
			q = q.WithTx(tx)

			newDyp := deployments[i+1]

			// down grade the current deployment
			if err := q.DownGradeDeployment(h.qCtx, db.DownGradeDeploymentParams{
				Status:       types.DeploymentInactive,
				DeploymentID: d.ID,
			}); err != nil {
				tx.Rollback()
				return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to start rollback"})
			}

			//upgrade the new deployment
			if err := q.UpgradeDeployment(h.qCtx, db.UpgradeDeploymentParams{
				DeploymentID: newDyp.ID,
				Status:       types.DeploymentBuilding,
			}); err != nil {
				tx.Rollback()
				return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to start rollback"})
			}

			// check if deployment image exists
			if newDyp.Status != types.DeploymentPruned && newDyp.ImageName.Valid {

				h.Server.DeploymentQ.EnqueueRedeployJob(&deploymentqueue.RedeployJobData{
					DeploymentID:     newDyp.ID,
					ImgName:          newDyp.ImageName.String,
					SwarmServiceName: newDyp.SwarmServiceName,
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
