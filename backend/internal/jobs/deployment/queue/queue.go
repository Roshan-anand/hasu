package deploymentqueue

import "github.com/google/uuid"

type JobType string

const (
	DeployJob  JobType = "deploy"
	RebuildJob JobType = "rebuild"
)

type PullJobData struct {
	Type              JobType
	DeploymentID      uuid.UUID
	Token             string
	Url               string
	Branch            string
	SwarmServiceName  string
	BuildPath         string
	DockerFilePath    string
	DockerContextPath string
	DockerBuildStage  string
	ImgName           string
	Env               []string
	BuildArgs         []string
	BuildSecrets      []string
}

type BuildJobData struct {
	Type              JobType
	DeploymentID      uuid.UUID
	BuildPath         string
	SwarmServiceName  string
	StorePath         string
	DockerFilePath    string
	DockerContextPath string
	DockerBuildStage  string
	ImgName           string
	Env               []string
	BuildArgs         []string
	BuildSecrets      []string
}

type DeployJobData struct {
	DeploymentID     uuid.UUID
	SwarmServiceName string
	ImgName          string
	Env              []string
}

type RedeployJobData = DeployJobData

type JobQueue struct {
	PullQueue     chan *PullJobData
	BuildQueue    chan *BuildJobData
	DeployQueue   chan *DeployJobData
	RedeployQueue chan *RedeployJobData
}

// initializes the job queues
func InitDeploymentQueue() *JobQueue {
	pull := make(chan *PullJobData, 10)
	build := make(chan *BuildJobData, 10)
	deploy := make(chan *DeployJobData, 10)
	redeploy := make(chan *RedeployJobData, 10)

	return &JobQueue{
		PullQueue:     pull,
		BuildQueue:    build,
		DeployQueue:   deploy,
		RedeployQueue: redeploy,
	}
}

// push job to pull worker queue
func (j *JobQueue) EnqueuePullJob(data *PullJobData) {
	j.PullQueue <- data
}

// push job to build worker queue
func (j *JobQueue) EnqueueBuildJob(data *BuildJobData) {
	j.BuildQueue <- data
}

// push job to deploy worker queue
func (j *JobQueue) EnqueueDeployJob(data *DeployJobData) {
	j.DeployQueue <- data
}

// push job to redeploy worker queue
func (j *JobQueue) EnqueueRedeployJob(data *RedeployJobData) {
	j.RedeployQueue <- data
}

// closes all the queue channels
func (j *JobQueue) Close() {
	close(j.PullQueue)
	close(j.BuildQueue)
	close(j.DeployQueue)
	close(j.RedeployQueue)
}
