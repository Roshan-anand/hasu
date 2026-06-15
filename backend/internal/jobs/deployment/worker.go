package deployjob

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/Roshan-anand/godploy/internal/db"
	"github.com/Roshan-anand/godploy/internal/jobs/logbroker"
	"github.com/Roshan-anand/godploy/internal/lib/types"
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
}

type reDeployData struct {
	deploymentID uuid.UUID `validate:"required"`
	swarmService string    `validate:"required"`
	isPublic     bool      `validate:"required"`
	env          []string  `validate:"required"`
	imgName      string    `validate:"required"`
}

type DeploymentServiceParams struct {
	JobType           JobType   `validate:"required,oneof=deploy rebuild redeploy"`
	DeploymentID      uuid.UUID `validate:"required"`
	Token             string    `validate:"required"`
	Url               string    `validate:"required"`
	RepoType          RepoType  `validate:"required,oneof=pr branch"`
	Branch            string    `validate:"required"`
	SwarmService      string    `validate:"required"`
	BuildPath         string    `validate:"required"`
	DockerFilePath    string    `validate:"required"`
	DockerContextPath string    `validate:"required"`
	DockerBuildStage  string    `validate:"required"`
	ImgName           string    `validate:"required"`
	Env               []string  `validate:"required"`
	BuildArgs         []string  `validate:"required"`
	BuildSecrets      []string  `validate:"required"`
	IsPublic          bool      `validate:"required"`
	InstanceID        uuid.UUID `validate:"omnitempty"`
}

// starts the deployment work pipeline
func (d *DeploymentService) runDeploymentPipeline(ctx context.Context, data *DeploymentServiceParams) {
	dataLog := &logbroker.PubData{
		ID: data.DeploymentID,
	}
	errLog := &logbroker.EndLogData{
		DeploymentID: data.DeploymentID,
		Status:       types.DeploymentError,
	}

	if err := d.v.Struct(data); err != nil {
		log.Printf("PullWorker: error validating data: %v\n", err)
		return
	}

	dataLog.Msg = getTitle("Pulling code from" + data.Url)
	d.log.PublishLog(dataLog)

	// update the deployment status to building
	if err := d.q.UpdateDeploymentStatus(d.qCtx, db.UpdateDeploymentStatusParams{
		Status: types.DeploymentBuilding,
		ID:     data.DeploymentID,
	}); err != nil {
		fmt.Printf("PullWorker: error updating deployment status: %v\n", err)
	}

	// clone the repo and get the code path
	cmd, outputPath := data.getCloneRepoCmd(d.codeStoreDir)
	if err := runWorkerCmd(d.log, data.DeploymentID, cmd, "pull"); err != nil {
		fmt.Printf("PullWorker: error running command: %v\n", err)
		d.log.EndLogs(errLog)
		return // TODO : trigger retry logic
	}

	fmt.Println("finished pulling :", data.Url)
	dataLog.Msg = getTitle("Building the image " + data.ImgName)
	d.log.PublishLog(dataLog)

	// generate a new docker build cmd
	buildCmd := data.getDockerBuildCmd(outputPath)

	if err := runWorkerCmd(d.log, data.DeploymentID, buildCmd, "build"); err != nil {
		fmt.Printf("BuildWorker: error running command: %v\n", err)
		d.log.EndLogs(errLog)
		return // TODO : trigger retry logic
	}

	// update the deployment with the built image name
	if err := d.q.SetDeploymentImageName(d.qCtx, db.SetDeploymentImageNameParams{
		ID: data.DeploymentID,
		Image: sql.NullString{
			Valid:  true,
			String: data.ImgName,
		},
	}); err != nil {
		fmt.Printf("BuildWorker: error updating deployment image name: %v\n", err)
		d.log.EndLogs(errLog)
		return // TODO : trigger retry logic
	}

	// remove the code folder
	go os.RemoveAll(outputPath)

	fmt.Println("finished building :", data.ImgName)

	switch data.JobType {
	case DeployJob:
		network, err := d.getServiceNetwork(data.InstanceID)
		if err != nil {
			errLog.Message = err.Error()
			d.log.EndLogs(errLog)
			return // TODO : trigger retry logic
		}
		d.deploy(data.getDeployData(network))

	case ReDeployJob:
		d.redeploy(data.getReDeployData())

	default:
		fmt.Printf("BuildWorker: unknown job type: %v\n", data.JobType)
	}
}

// starts the deploy pipeline for the given data
func (d *DeploymentService) deploy(data *deployData) {
	if err := d.v.Struct(data); err != nil {
		log.Printf("PullWorker: error validating data: %v\n", err)
		return
	}

	d.log.PublishLog(&logbroker.PubData{
		ID:  data.deploymentID,
		Msg: getTitle("Deploying  the service " + data.swarmService),
	})

	// get the service spec
	spec := data.getBaseSpec()

	_, err := d.docker.Client.ServiceCreate(context.Background(), *spec, swarm.ServiceCreateOptions{})
	if err != nil {
		fmt.Printf("DeployWorker: error creating service: %v\n", err)
		d.log.EndLogs(&logbroker.EndLogData{
			DeploymentID: data.deploymentID,
			Status:       types.DeploymentError,
			Message:      err.Error(),
		})

		return // TODO : trigger retry logic
	}

	fmt.Println("finished deploying :", data.swarmService)
	d.log.EndLogs(&logbroker.EndLogData{
		DeploymentID: data.deploymentID,
		Status:       types.DeploymentReady,
		Message:      getTitle("successfully deployed"),
	})
}

// starts the redeploy pipeline for the given data
func (d *DeploymentService) redeploy(data *reDeployData) {
	if err := d.v.Struct(data); err != nil {
		log.Printf("PullWorker: error validating data: %v\n", err)
		return
	}

	d.log.PublishLog(&logbroker.PubData{
		ID:  data.deploymentID,
		Msg: "Redeploying  the service " + data.swarmService,
	})

	// get the swarm service spec
	res, _, err := d.docker.Client.ServiceInspectWithRaw(context.Background(), data.swarmService, swarm.ServiceInspectOptions{})
	if err != nil {
		fmt.Printf("DeployWorker: error inspecting service: %v\n", err)
		d.log.EndLogs(&logbroker.EndLogData{
			DeploymentID: data.deploymentID,
			Status:       types.DeploymentError,
			Message:      err.Error(),
		})

		return // TODO : trigger retry logic
	}
	version := res.Version
	spec := res.Spec

	// update the image and env
	spec.TaskTemplate.ContainerSpec.Image = data.imgName
	if len(data.env) > 0 {
		spec.TaskTemplate.ContainerSpec.Env = data.env
	}

	// update the service with the new spec
	if _, err := d.docker.Client.ServiceUpdate(context.Background(), data.swarmService, version, spec, swarm.ServiceUpdateOptions{}); err != nil {
		fmt.Printf("DeployWorker: error updating service: %v\n", err)
		d.log.EndLogs(&logbroker.EndLogData{
			DeploymentID: data.deploymentID,
			Status:       types.DeploymentError,
			Message:      err.Error(),
		})

		return // TODO : trigger retry logic
	}

	// end the logs
	d.log.EndLogs(&logbroker.EndLogData{
		DeploymentID: data.deploymentID,
		Status:       types.DeploymentReady,
		Message:      getTitle("successfully redeployed"),
	})
}
