package deployjob

import (
	"context"
	"fmt"
	"log"
	"path"

	"github.com/Roshan-anand/godploy/internal/jobs/logbroker"
	"github.com/Roshan-anand/godploy/internal/lib/docker"
	ghservice "github.com/Roshan-anand/godploy/internal/lib/gh"
	"github.com/Roshan-anand/godploy/internal/lib/types"
	"github.com/Roshan-anand/godploy/internal/lib/utils"
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

type ReDeployData struct {
	DeploymentID uuid.UUID `validate:"required"`
	SwarmService string    `validate:"required"`
	Env          []string  `validate:"required"`
	ImgName      string    `validate:"required"`
}

type DeploymentServiceParams struct {
	DeploymentID      uuid.UUID `validate:"required"`
	InstanceID        uuid.UUID `validate:"required"`
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
	BuildArgs         []string `validate:"required"`
	BuildSecrets      []string `validate:"required"`
	IsPublic          bool     `validate:"required"`
}

type RebuildServiceParams struct {
	ServiceID  uuid.UUID `validate:"required"`
	CommitHash string    `validate:"required"`
	CommitMsg  string    `validate:"required"`
}

// starts the deployment work pipeline
func (d *DeploymentService) runDeploymentPipeline(ctx context.Context, data *DeploymentServiceParams) {
	errLog := &logbroker.EndLogData{
		DeploymentID: data.DeploymentID,
		Status:       types.DeploymentError,
	}

	// create the deployment utils
	outputPath := path.Join(d.codeStoreDir, data.SwarmService)
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
		BuildArgs:         data.BuildArgs,
		BuildSecrets:      data.BuildSecrets,
	})
	if err != nil {
		fmt.Println("PullWorker: error creating deployment utils:", err)
		return
	}

	// trigger pull code
	if err := utils.pullCode(d); err != nil {
		fmt.Println("PullWorker: error pulling code:", err)
		errLog.Message = errorMsg(err.Error())
		d.log.EndLogs(errLog)
		return // TODO : trigger retry logic
	}

	// trigger build image
	if err := utils.buildImg(d); err != nil {
		errLog.Message = errorMsg(err.Error())
		d.log.EndLogs(errLog)
		return // TODO : trigger retry logic
	}

	network, err := d.getServiceNetwork(data.InstanceID)
	if err != nil {
		errLog.Message = errorMsg(err.Error())
		d.log.EndLogs(errLog)
		return // TODO : trigger retry logic
	}

	d.deploy(data.getDeployData(network))
}

// starts the rebuild pipeline for the given data
func (d *DeploymentService) runRebuildPipeline(ctx context.Context, data *RebuildServiceParams) {
	errLog := &logbroker.EndLogData{
		Status: types.DeploymentError,
	}

	q := d.db.Queries

	s, err := q.GetAppServiceForRebuild(d.qCtx, data.ServiceID)
	if err != nil {
		fmt.Println("RebuildWorker: error getting app service for rebuild:", err)
		return // TODO : trigger retry logic
	}

	// update the deployment status to building
	dID, err := data.createNewDeploymentData(d, &s)
	errLog.DeploymentID = dID
	if err != nil {
		fmt.Println("RebuildWorker: error creating new deployment data for rebuild:", err)
		d.log.EndLogs(errLog)
		return // TODO : trigger retry logic
	}

	// used as unique image
	// note : we do not use a new service name
	unique := docker.GenerateServiceAndImgName(s.Name, s.Branch)

	envStr, err := utils.UnmarshalServiceEnv(&utils.ServiceEnvByte{
		Env:          s.Env,
		BuildArgs:    s.BuildArgs,
		BuildSecrets: s.BuildSecrets,
	})
	if err != nil {
		fmt.Println("RebuildWorker: Error unmarshaling service env:", err)
		d.log.EndLogs(errLog)
		return // TODO : trigger retry logic
	}

	// create new github client
	gh, err := ghservice.New(q, s.GhAppID)
	if err != nil {
		fmt.Println("Error creating github client:", err)
		d.log.EndLogs(errLog)
		return // TODO : trigger retry logic
	}

	// create the deployment utils
	outputPath := path.Join(d.codeStoreDir, s.SwarmService)
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
		BuildArgs:         envStr.BuildArgs,
		BuildSecrets:      envStr.BuildSecrets,
	})
	if err != nil {
		fmt.Println("PullWorker: error creating deployment utils:", err)
		return
	}

	// trigger pull code
	if err := utils.pullCode(d); err != nil {
		errLog.Message = errorMsg(err.Error())
		d.log.EndLogs(errLog)
		return // TODO : trigger retry logic
	}

	// trigger build image
	if err := utils.buildImg(d); err != nil {
		errLog.Message = errorMsg(err.Error())
		d.log.EndLogs(errLog)
		return // TODO : trigger retry logic
	}

	fmt.Println("finished building :", unique.ImgName)
	d.redeploy(&ReDeployData{
		DeploymentID: dID,
		SwarmService: s.SwarmService,
		Env:          envStr.Env,
		ImgName:      unique.ImgName,
	})
}

// starts the deploy pipeline for the given data
func (d *DeploymentService) deploy(data *deployData) {
	if err := d.v.Struct(data); err != nil {
		log.Printf("PullWorker: error validating data: %v\n", err)
		return
	}

	d.log.PublishLog(&logbroker.PubData{
		ID:  data.deploymentID,
		Msg: infoMsg("Deploying  the service " + data.swarmService),
	})

	// get the service spec
	spec := data.getBaseSpec()

	_, err := d.docker.Client.ServiceCreate(context.Background(), *spec, swarm.ServiceCreateOptions{})
	if err != nil {
		fmt.Printf("DeployWorker: error creating service: %v\n", err)
		d.log.EndLogs(&logbroker.EndLogData{
			DeploymentID: data.deploymentID,
			Status:       types.DeploymentError,
			Message:      errorMsg(err.Error()),
		})

		return // TODO : trigger retry logic
	}

	fmt.Println("finished deploying :", data.swarmService)
	d.log.EndLogs(&logbroker.EndLogData{
		DeploymentID: data.deploymentID,
		Status:       types.DeploymentReady,
		Message:      successMsg("successfully deployed : " + data.swarmService),
	})
}

// starts the redeploy pipeline for the given data
func (d *DeploymentService) redeploy(data *ReDeployData) {
	if err := d.v.Struct(data); err != nil {
		log.Printf("PullWorker: error validating data: %v\n", err)
		return
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

		return // TODO : trigger retry logic
	}
	version := res.Version
	spec := res.Spec

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

		return // TODO : trigger retry logic
	}

	// end the logs
	d.log.EndLogs(&logbroker.EndLogData{
		DeploymentID: data.DeploymentID,
		Status:       types.DeploymentReady,
		Message:      successMsg("successfully redeployed"),
	})
}
