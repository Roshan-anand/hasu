package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	"github.com/Roshan-anand/godploy/internal/db"
	deploymentqueue "github.com/Roshan-anand/godploy/internal/jobs/deployment/queue"
	"github.com/Roshan-anand/godploy/internal/lib/gh"
	"github.com/Roshan-anand/godploy/internal/lib/security"
	"github.com/Roshan-anand/godploy/internal/lib/types"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

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

	// start a new db transaction
	tx, err := h.Server.DB.Pool.BeginTx(context.Background(), nil)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to create service"})
	}
	q = q.WithTx(tx)

	var newStatus types.DeploymentStatus
	if service.DeploymentStatus == types.DeploymentReady {
		newStatus = types.DeploymentInactive
	} else {
		newStatus = types.DeploymentPruned
	}

	// update the previous deployment is_latest to false
	if err := q.DownGradeDeployment(h.qCtx, db.DownGradeDeploymentParams{
		DeploymentID: service.DeploymentID,
		Status:       newStatus,
	}); err != nil {
		tx.Rollback()
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to create service"})
	}

	// TODO : get commit msg from client side
	// create a new deployment for the app service
	dID, err := q.CreateDeployment(h.qCtx, db.CreateDeploymentParams{
		ID:        security.GeneratePrimaryKey(),
		BranchID:  service.BranchID,
		CommitMsg: "s",
		IsCurrent: true,
	})
	if err != nil {
		tx.Rollback()
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to create deployment"})
	}

	// get the github app details
	ghApp, err := q.GetGhAppByAppId(h.qCtx, service.GhAppID)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "invalid github app"})
		}
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "failed to verify github app"})
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
