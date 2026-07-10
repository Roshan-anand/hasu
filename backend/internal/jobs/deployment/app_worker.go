package deployjob

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"

	"github.com/Roshan-anand/godploy/internal/db"
	"github.com/Roshan-anand/godploy/internal/jobs/logbroker"
	"github.com/Roshan-anand/godploy/internal/lib/docker"
	ghservice "github.com/Roshan-anand/godploy/internal/lib/gh"
	"github.com/Roshan-anand/godploy/internal/lib/types"
	"github.com/Roshan-anand/godploy/internal/lib/utils"
	"github.com/containerd/errdefs"
	"github.com/docker/docker/api/types/swarm"
	"github.com/google/uuid"
)

type deployData struct {
	deploymentID uuid.UUID `validate:"required"`
	swarmService string    `validate:"required"`
	networkName  string    `validate:"required"`
	isPublic     bool      `validate:"required"`
	env          []string  `validate:"required"`
	imgName      string    `validate:"required"`
	domain       string
	port         int32
}

type ReDeployData struct {
	DeploymentID uuid.UUID `validate:"required"`
	ServiceID    uuid.UUID `validate:"required"`
	SwarmService string    `validate:"required"`
	Env          []string
	ImgName      string `validate:"required"`
}

type CloneDeployData struct {
	InstanceID   uuid.UUID `validate:"required"`
	ServiceID    uuid.UUID `validate:"required"`
	SwarmService string    `validate:"required"`
	NetworkName  string    `validate:"required"`
	ImgName      string    `validate:"required"`
	Env          []string  `validate:"required"`
	Domain       string    `validate:"required"`
	IsPublic     bool
	Port         int32
}

type DeploymentServiceParams struct {
	DeploymentID      uuid.UUID `validate:"required"`
	InstanceID        uuid.UUID `validate:"required"`
	ServiceID         uuid.UUID `validate:"required"`
	Token             string    `validate:"required"`
	Url               string    `validate:"required"`
	Branch            string    `validate:"required"`
	SwarmService      string    `validate:"required"`
	BuildPath         string    `validate:"required"`
	DockerFilePath    string
	DockerContextPath string
	DockerBuildStage  string
	ImgName           string   `validate:"required"`
	Env               []string `validate:"required"`
	BuildSecrets      []string `validate:"required"`
	IsPublic          bool
	GitProvider       types.GitProvider
	JobID             int64
}

type RebuildServiceParams struct {
	ServiceID  uuid.UUID `validate:"required"`
	CommitHash string    `validate:"required"`
	CommitMsg  string    `validate:"required"`
	// Source identifies how the rebuild was triggered: "manual" or "webhook".
	// It is used only for logging/test assertions; it does not change logic.
	Source string
	// JobID is the monotonic identity token assigned by the rebuild registry.
	// Workers use it for identity checks when cleaning up registry entries.
	JobID int64
}

// starts the deployment work pipeline
func (d *DeploymentService) runDeploymentPipeline(ctx context.Context, data *DeploymentServiceParams) error {
	// If this deploy work was canceled while queued, exit without creating a Deployment row.
	if err := ctx.Err(); err != nil {
		fmt.Println("RebuildWorker: rebuild canceled before starting work:", err)
		return fmt.Errorf("rebuild:ctx_canceled: %w", err)
	}

	// Record the candidate Deployment ID in the registry so explicit cancel by
	// Deployment identifier can locate the active rebuild in later phases.
	d.SetRebuildDeploymentID(data.ServiceID, data.JobID, data.DeploymentID)

	errLog := &logbroker.EndLogData{
		DeploymentID: data.DeploymentID,
		Status:       types.DeploymentError,
	}

	// resolve and merge dependency env values
	data.Env = MergeDependencyEnv(d.db.Queries, data.ServiceID, data.Env)

	// create a unique output path for the code
	outputPath := getOutputPath(d.codeStoreDir, data.SwarmService)

	// create the deployment utils
	utils, err := d.newDeploymentServiceUtils(&DeploymentServiceUtils{
		DeploymentID:      data.DeploymentID,
		Token:             data.Token,
		Url:               data.Url,
		Branch:            data.Branch,
		OutputPath:        outputPath,
		BuildPath:         data.BuildPath,
		DockerFilePath:    data.DockerFilePath,
		DockerContextPath: data.DockerContextPath,
		DockerBuildStage:  data.DockerBuildStage,
		ImgName:           data.ImgName,
		Env:               data.Env,
		BuildSecrets:      data.BuildSecrets,
		GitProvider:       data.GitProvider,
	})
	if err != nil {
		fmt.Println("PullWorker: error creating deployment utils:", err)
		return fmt.Errorf("deploy:create_utils: %w", err)
	}

	// trigger pull code
	if err := utils.pullCode(ctx, d); err != nil {
		if ctx.Err() != nil {
			d.MarkDeploymentCanceled(data.DeploymentID, "canceled")
			return fmt.Errorf("deploy:pull_code_canceled: %w", err)
		}
		errLog.Message = errorMsg(err.Error())
		d.log.EndLogs(errLog)
		return fmt.Errorf("deploy:pull_code: %w", err) // TODO : trigger retry logic
	}

	if err := ctx.Err(); err != nil {
		d.MarkDeploymentCanceled(data.DeploymentID, "canceled")
		return fmt.Errorf("deploy:ctx_canceled_after_pull: %w", err)
	}

	// trigger build image
	if err := utils.buildImg(ctx, d); err != nil {
		if ctx.Err() != nil {
			d.MarkDeploymentCanceled(data.DeploymentID, "canceled")
			return fmt.Errorf("deploy:build_img_canceled: %w", err)
		}
		errLog.Message = errorMsg(err.Error())
		d.log.EndLogs(errLog)
		return fmt.Errorf("deploy:build_img: %w", err) // TODO : trigger retry logic
	}

	if err := ctx.Err(); err != nil {
		d.MarkDeploymentCanceled(data.DeploymentID, "canceled")
		return fmt.Errorf("deploy:ctx_canceled_after_build: %w", err)
	}

	network, err := d.getServiceNetwork(data.InstanceID)
	if err != nil {
		errLog.Message = errorMsg(err.Error())
		d.log.EndLogs(errLog)
		return fmt.Errorf("deploy:get_network: %w", err) // TODO : trigger retry logic
	}

	if err := d.deploy(ctx, data.getDeployData(network)); err != nil {
		return fmt.Errorf("deploy:deploy_service: %w", err)
	}

	return nil
}

// MarkDeploymentCanceled finalizes the log stream for a canceled rebuild and
// updates the Deployment status to canceled. It does not touch any Current
// Deployment.
func (d *DeploymentService) MarkDeploymentCanceled(deploymentID uuid.UUID, reason string) {
	if err := d.log.EndLogs(&logbroker.EndLogData{
		DeploymentID: deploymentID,
		Status:       types.DeploymentCanceled,
		Message:      warningMsg("rebuild canceled: " + reason),
	}); err != nil {
		fmt.Printf("RebuildWorker: error ending canceled logs: %v\n", err)
	}
}

// RunRebuildPipeline starts the rebuild pipeline for the given data.
// It is exported so integration tests can exercise cancellation behavior
// directly without invoking the full async queue.
func (d *DeploymentService) RunRebuildPipeline(ctx context.Context, data *RebuildServiceParams) error {
	// If this rebuild was canceled while queued, exit without creating a
	// Deployment row and without touching the previous Current Deployment.
	if err := ctx.Err(); err != nil {
		fmt.Println("RebuildWorker: rebuild canceled before starting work:", err)
		return fmt.Errorf("rebuild:ctx_canceled: %w", err)
	}

	errLog := &logbroker.EndLogData{
		Status: types.DeploymentError,
	}

	q := d.db.Queries

	s, err := q.GetAppServiceForRebuild(d.qCtx, data.ServiceID)
	if err != nil {
		fmt.Println("RebuildWorker: error getting app service for rebuild:", err)
		return fmt.Errorf("rebuild:get_service: %w", err) // TODO : trigger retry logic
	}

	// create a new Deployment row for the rebuild candidate.
	dID, err := data.createNewDeploymentData(d, &s)
	errLog.DeploymentID = dID
	if err != nil {
		fmt.Println("RebuildWorker: error creating new deployment data for rebuild:", err)
		d.log.EndLogs(errLog)
		return fmt.Errorf("rebuild:create_deployment_data: %w", err) // TODO : trigger retry logic
	}

	// Record the candidate Deployment ID in the registry so explicit cancel by
	// Deployment identifier can locate the active rebuild in later phases.
	d.SetRebuildDeploymentID(data.ServiceID, data.JobID, dID)

	// If cancellation arrived right after the candidate row was created, mark
	// it canceled and stop before doing any pull/build work.
	if err := ctx.Err(); err != nil {
		d.MarkDeploymentCanceled(dID, "replaced by newer rebuild")
		return fmt.Errorf("rebuild:ctx_canceled_after_candidate: %w", err)
	}

	// used as unique image
	// note : we do not use a new service name
	unique := docker.GenerateServiceAndImgName(s.Name, s.Branch)

	envStr, err := utils.UnmarshalServiceEnv(&utils.ServiceEnvByte{
		Env:          s.Env,
		BuildSecrets: s.BuildSecrets,
	})
	if err != nil {
		fmt.Println("RebuildWorker: Error unmarshaling service env:", err)
		d.log.EndLogs(errLog)
		return fmt.Errorf("rebuild:unmarshal_env: %w", err) // TODO : trigger retry logic
	}

	// resolve and merge dependency env values
	envStr.Env = MergeDependencyEnv(d.db.Queries, data.ServiceID, envStr.Env)

	// If cancellation arrived before we talk to GitHub, mark canceled and stop.
	if err := ctx.Err(); err != nil {
		d.MarkDeploymentCanceled(dID, "replaced by newer rebuild")
		return fmt.Errorf("rebuild:ctx_canceled_before_gh: %w", err)
	}

	// create new github client
	gh, err := ghservice.New(q, s.GhAppID)
	if err != nil {
		fmt.Println("Error creating github client:", err)
		d.log.EndLogs(errLog)
		return fmt.Errorf("rebuild:create_gh: %w", err) // TODO : trigger retry logic
	}

	// create a unique output path for the code
	outputPath := getOutputPath(d.codeStoreDir, s.SwarmService)

	// create the deployment utils
	utils, err := d.newDeploymentServiceUtils(&DeploymentServiceUtils{
		DeploymentID:      dID,
		Token:             gh.Token,
		Url:               s.GhRepoUrl,
		Branch:            s.Branch,
		OutputPath:        outputPath,
		BuildPath:         s.BuildPath,
		DockerFilePath:    s.DockerFilepath,
		DockerContextPath: s.DockerContextpath,
		DockerBuildStage:  s.DockerBuildstage,
		ImgName:           unique.ImgName,
		Env:               envStr.Env,
		BuildSecrets:      envStr.BuildSecrets,
	})
	if err != nil {
		fmt.Println("PullWorker: error creating deployment utils:", err)
		return fmt.Errorf("rebuild:create_utils: %w", err)
	}

	// trigger pull code
	if err := utils.pullCode(ctx, d); err != nil {
		if ctx.Err() != nil {
			d.MarkDeploymentCanceled(dID, "replaced by newer rebuild")
			return fmt.Errorf("rebuild:pull_code_canceled: %w", err)
		}
		errLog.Message = errorMsg(err.Error())
		d.log.EndLogs(errLog)
		return fmt.Errorf("rebuild:pull_code: %w", err) // TODO : trigger retry logic
	}

	// If cancellation arrived after a successful pull, mark canceled and stop.
	if err := ctx.Err(); err != nil {
		d.MarkDeploymentCanceled(dID, "replaced by newer rebuild")
		return fmt.Errorf("rebuild:ctx_canceled_after_pull: %w", err)
	}

	// trigger build image
	if err := utils.buildImg(ctx, d); err != nil {
		if ctx.Err() != nil {
			d.MarkDeploymentCanceled(dID, "replaced by newer rebuild")
			return fmt.Errorf("rebuild:build_img_canceled: %w", err)
		}
		errLog.Message = errorMsg(err.Error())
		d.log.EndLogs(errLog)
		return fmt.Errorf("rebuild:build_img: %w", err) // TODO : trigger retry logic
	}

	fmt.Println("finished building :", unique.ImgName)

	// If cancellation arrived after a successful build, mark canceled and stop
	// before touching the Docker swarm service.
	if err := ctx.Err(); err != nil {
		d.MarkDeploymentCanceled(dID, "replaced by newer rebuild")
		return fmt.Errorf("rebuild:ctx_canceled_after_build: %w", err)
	}

	network, err := d.getServiceNetwork(s.InstanceID)
	if err != nil {
		fmt.Printf("RebuildWorker: error getting service network: %v\n", err)
		d.log.EndLogs(errLog)
		return fmt.Errorf("rebuild:get_network: %w", err) // TODO : trigger retry logic
	}

	domain := ""
	if s.Domain.Valid {
		domain = s.Domain.String
	}

	if err := d.applyRebuildDeployment(ctx, s.ID, &deployData{
		deploymentID: dID,
		swarmService: s.SwarmService,
		networkName:  network,
		isPublic:     s.IsPublic,
		env:          envStr.Env,
		imgName:      unique.ImgName,
		domain:       domain,
		port:         s.Port,
	}); err != nil {
		return fmt.Errorf("rebuild:apply_deployment: %w", err)
	}

	return nil
}

// starts the deploy pipeline for the given data
func (d *DeploymentService) deploy(ctx context.Context, data *deployData) error {
	if err := d.v.Struct(data); err != nil {
		log.Printf("PullWorker: error validating data: %v\n", err)
		return fmt.Errorf("deploy:validate: %w", err)
	}

	d.log.PublishLog(&logbroker.PubData{
		ID:  data.deploymentID,
		Msg: infoMsg("Deploying  the service " + data.swarmService),
	})

	// get the service spec
	spec := data.getBaseSpec()

	_, err := d.docker.Client.ServiceCreate(ctx, *spec, swarm.ServiceCreateOptions{})
	if err != nil {
		if ctx.Err() != nil {
			d.MarkDeploymentCanceled(data.deploymentID, "replaced by newer rebuild")
			return fmt.Errorf("apply:ctx_canceled_inspect: %w", err)
		}

		fmt.Printf("DeployWorker: error creating service: %v\n", err)
		d.log.EndLogs(&logbroker.EndLogData{
			DeploymentID: data.deploymentID,
			Status:       types.DeploymentError,
			Message:      errorMsg(err.Error()),
		})

		return fmt.Errorf("deploy:create_service: %w", err) // TODO : trigger retry logic
	}

	fmt.Println("finished deploying :", data.swarmService)
	d.log.EndLogs(&logbroker.EndLogData{
		DeploymentID: data.deploymentID,
		Status:       types.DeploymentReady,
		Message:      successMsg("successfully deployed : " + data.swarmService),
	})

	return nil
}

// Redeploy starts the redeploy pipeline for the given data.
// It is exported so integration tests can exercise Docker-apply failures directly.
func (d *DeploymentService) Redeploy(ctx context.Context, data *ReDeployData) error {
	if err := d.v.Struct(data); err != nil {
		log.Printf("PullWorker: error validating data: %v\n", err)
		return fmt.Errorf("redeploy:validate: %w", err)
	}

	d.log.PublishLog(&logbroker.PubData{
		ID:  data.DeploymentID,
		Msg: infoMsg("Redeploying  the service " + data.SwarmService),
	})

	// get the swarm service spec
	res, _, err := d.docker.Client.ServiceInspectWithRaw(context.Background(), data.SwarmService, swarm.ServiceInspectOptions{})
	if err != nil {
		fmt.Printf("DeployWorker: error inspecting service: %v\n", err)
		d.log.EndLogs(&logbroker.EndLogData{
			DeploymentID: data.DeploymentID,
			Status:       types.DeploymentError,
			Message:      errorMsg(err.Error()),
		})

		return fmt.Errorf("redeploy:inspect_service: %w", err) // TODO : trigger retry logic
	}
	version := res.Version
	spec := res.Spec

	// Resolve dependencies at apply time so redeploys use current values.
	data.Env = MergeDependencyEnv(d.db.Queries, data.ServiceID, data.Env)

	// update the image and env
	spec.TaskTemplate.ContainerSpec.Image = data.ImgName
	if len(data.Env) > 0 {
		spec.TaskTemplate.ContainerSpec.Env = data.Env
	}

	// update the service with the new spec
	if _, err := d.docker.Client.ServiceUpdate(context.Background(), data.SwarmService, version, spec, swarm.ServiceUpdateOptions{}); err != nil {
		fmt.Printf("DeployWorker: error updating service: %v\n", err)
		d.log.EndLogs(&logbroker.EndLogData{
			DeploymentID: data.DeploymentID,
			Status:       types.DeploymentError,
			Message:      errorMsg(err.Error()),
		})

		return fmt.Errorf("redeploy:update_service: %w", err) // TODO : trigger retry logic
	}

	// For rebuilds, promote the candidate to Current and downgrade the previous
	// Current Deployment in a single transaction. For plain redeploys there is
	// no candidate to promote, so just end the logs.
	if data.ServiceID != uuid.Nil {
		if err := d.PromoteDeploymentToCurrent(ctx, data.ServiceID, data.DeploymentID); err != nil {
			fmt.Printf("DeployWorker: error promoting deployment: %v\n", err)
			d.log.EndLogs(&logbroker.EndLogData{
				DeploymentID: data.DeploymentID,
				Status:       types.DeploymentError,
				Message:      errorMsg(err.Error()),
			})
			return fmt.Errorf("redeploy:promote: %w", err) // TODO : trigger retry logic
		}
	}

	// end the logs
	d.log.EndLogs(&logbroker.EndLogData{
		DeploymentID: data.DeploymentID,
		Status:       types.DeploymentReady,
		Message:      successMsg("successfully redeployed"),
	})

	return nil
}

// applyRebuildDeployment applies the rebuilt image to the swarm service. It
// first inspects the service; if it exists the existing spec is reused and only
// the image (and env) is replaced. If the service is missing, a fresh spec is
// built from the saved Service configuration and the candidate image and a new
// service is created. The candidate is promoted to Current only after Docker
// accepts the create/update.
func (d *DeploymentService) applyRebuildDeployment(ctx context.Context, serviceID uuid.UUID, data *deployData) error {
	if err := d.v.Struct(data); err != nil {
		log.Printf("RebuildWorker: error validating deploy data: %v\n", err)
		d.log.EndLogs(&logbroker.EndLogData{
			DeploymentID: data.deploymentID,
			Status:       types.DeploymentError,
			Message:      errorMsg(err.Error()),
		})
		return fmt.Errorf("apply:validate: %w", err)
	}

	d.log.PublishLog(&logbroker.PubData{
		ID:  data.deploymentID,
		Msg: infoMsg("Applying rebuilt image to service " + data.swarmService),
	})

	var (
		spec    swarm.ServiceSpec
		version swarm.Version
		create  bool
	)

	res, _, err := d.docker.Client.ServiceInspectWithRaw(ctx, data.swarmService, swarm.ServiceInspectOptions{})
	if err != nil {
		if ctx.Err() != nil {
			d.MarkDeploymentCanceled(data.deploymentID, "replaced by newer rebuild")
			return fmt.Errorf("apply:ctx_canceled_inspect: %w", err)
		}
		if errdefs.IsNotFound(err) {
			create = true
			spec = *data.getBaseSpec()
		} else {
			fmt.Printf("RebuildWorker: error inspecting service: %v\n", err)
			d.log.EndLogs(&logbroker.EndLogData{
				DeploymentID: data.deploymentID,
				Status:       types.DeploymentError,
				Message:      errorMsg(err.Error()),
			})
			return fmt.Errorf("apply:inspect_service: %w", err) // TODO : trigger retry logic
		}
	} else {
		version = res.Version
		spec = res.Spec
		spec.TaskTemplate.ContainerSpec.Image = data.imgName
		if len(data.env) > 0 {
			spec.TaskTemplate.ContainerSpec.Env = data.env
		}
	}

	if err := ctx.Err(); err != nil {
		d.MarkDeploymentCanceled(data.deploymentID, "replaced by newer rebuild")
		return fmt.Errorf("apply:ctx_canceled_before_apply: %w", err)
	}

	if create {
		_, err = d.docker.Client.ServiceCreate(ctx, spec, swarm.ServiceCreateOptions{})
	} else {
		_, err = d.docker.Client.ServiceUpdate(ctx, data.swarmService, version, spec, swarm.ServiceUpdateOptions{})
	}
	if err != nil {
		if ctx.Err() != nil {
			d.MarkDeploymentCanceled(data.deploymentID, "replaced by newer rebuild")
			return fmt.Errorf("apply:ctx_canceled_apply: %w", err)
		}
		fmt.Printf("RebuildWorker: error applying service: %v\n", err)
		d.log.EndLogs(&logbroker.EndLogData{
			DeploymentID: data.deploymentID,
			Status:       types.DeploymentError,
			Message:      errorMsg(err.Error()),
		})
		return fmt.Errorf("apply:apply_service: %w", err) // TODO : trigger retry logic
	}

	if err := d.PromoteDeploymentToCurrent(ctx, serviceID, data.deploymentID); err != nil {
		fmt.Printf("RebuildWorker: error promoting deployment: %v\n", err)
		d.log.EndLogs(&logbroker.EndLogData{
			DeploymentID: data.deploymentID,
			Status:       types.DeploymentError,
			Message:      errorMsg(err.Error()),
		})
		return fmt.Errorf("apply:promote: %w", err) // TODO : trigger retry logic
	}

	d.log.EndLogs(&logbroker.EndLogData{
		DeploymentID: data.deploymentID,
		Status:       types.DeploymentReady,
		Message:      successMsg("successfully rebuilt and applied : " + data.swarmService),
	})

	return nil
}

// PromoteDeploymentToCurrent promotes the candidate deployment to Current and
// downgrades the previous Current Deployment in a single database transaction.
// If no previous Current Deployment exists, it skips the downgrade.
// The caller's ctx is used for the transaction so it can be interrupted during shutdown.
func (d *DeploymentService) PromoteDeploymentToCurrent(ctx context.Context, serviceID, candidateID uuid.UUID) error {
	tx, err := d.db.Pool.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("start promotion transaction: %w", err)
	}
	defer tx.Rollback()

	tq := d.db.Queries.WithTx(tx)

	prev, err := tq.GetCurrentDeploymentByServiceId(ctx, serviceID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("lookup current deployment: %w", err)
	}

	// downgrade previous current deployment if it exists and is not the candidate itself
	if err == nil && prev.ID != candidateID {
		if err := tq.DownGradeDeployment(ctx, db.DownGradeDeploymentParams{
			DeploymentID: prev.ID,
			Status:       types.DeploymentInactive,
		}); err != nil {
			return fmt.Errorf("downgrade previous current deployment: %w", err)
		}
	}

	// promote candidate to current / ready
	if err := tq.UpgradeDeployment(ctx, db.UpgradeDeploymentParams{
		DeploymentID: candidateID,
		Status:       types.DeploymentReady,
	}); err != nil {
		return fmt.Errorf("upgrade candidate deployment: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit promotion transaction: %w", err)
	}

	return nil
}

// runCloneDeployPipeline creates a new swarm service for a preview instance
// using an existing production image and merged environment variables.
// It skips the build step entirely and applies a preview-specific Traefik Host rule.
func (d *DeploymentService) runCloneDeployPipeline(ctx context.Context, data *CloneDeployData) error {
	if err := d.v.Struct(data); err != nil {
		log.Printf("CloneDeploy: validation error: %v\n", err)
		return fmt.Errorf("clone_deploy:validate: %w", err)
	}

	d.log.PublishLog(&logbroker.PubData{
		Msg: infoMsg("Clone deploying preview service " + data.SwarmService),
	})

	// resolve and merge dependency env values
	env := MergeDependencyEnv(d.db.Queries, data.ServiceID, data.Env)

	// create the network if it doesn't exist
	if err := d.docker.CreateNetwork(data.NetworkName); err != nil {
		fmt.Printf("CloneDeploy: error creating network: %v\n", err)
		d.log.EndLogs(&logbroker.EndLogData{
			Status:  types.DeploymentError,
			Message: errorMsg(err.Error()),
		})
		return fmt.Errorf("clone_deploy:create_network: %w", err)
	}

	spec := (&deployData{
		swarmService: data.SwarmService,
		networkName:  data.NetworkName,
		isPublic:     data.IsPublic,
		env:          env,
		imgName:      data.ImgName,
		domain:       data.Domain,
		port:         data.Port,
	}).getBaseSpec()

	_, err := d.docker.Client.ServiceCreate(ctx, *spec, swarm.ServiceCreateOptions{})
	if err != nil {
		fmt.Printf("CloneDeploy: error creating service: %v\n", err)
		d.log.EndLogs(&logbroker.EndLogData{
			Status:  types.DeploymentError,
			Message: errorMsg(err.Error()),
		})
		return fmt.Errorf("clone_deploy:create_service: %w", err)
	}

	fmt.Println("finished clone deploying:", data.SwarmService)
	d.log.PublishLog(&logbroker.PubData{
		Msg: successMsg("successfully clone deployed : " + data.SwarmService),
	})

	return nil
}
