package deploymentqueue

import "github.com/google/uuid"

//	!IMP : this is just a placeholder
//
// TODO : fill struct with actual data required for the job
type PullJobData struct {
	DeploymentID uuid.UUID
	Token        string
	Url          string
	Branch       string
	StorePath    string
	BuildPath    string
}

type BuildJobData struct {
	DeploymentID uuid.UUID
	BuildPath    string
}

type DeployJobData struct {
	DeploymentID uuid.UUID
}

type JobQueue struct {
	PullQueue   chan *PullJobData
	BuildQueue  chan *BuildJobData
	DeployQueue chan *DeployJobData
}

// initializes the job queues
func InitDeploymentQueue() *JobQueue {
	pull := make(chan *PullJobData, 10)
	build := make(chan *BuildJobData, 10)
	deploy := make(chan *DeployJobData, 10)

	return &JobQueue{
		PullQueue:   pull,
		BuildQueue:  build,
		DeployQueue: deploy,
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

// closes all the queue channels
func (j *JobQueue) Close() {
	close(j.PullQueue)
	close(j.BuildQueue)
	close(j.DeployQueue)
}
