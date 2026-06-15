package deployjob

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/Roshan-anand/godploy/internal/db"
	"github.com/Roshan-anand/godploy/internal/jobs/logbroker"
	"github.com/Roshan-anand/godploy/internal/lib/docker"
	"github.com/go-playground/validator/v10"
	"golang.org/x/sync/errgroup"
)

type JobType string

const (
	DeployJob   JobType = "deploy"
	RebuildJob  JobType = "rebuild"
	ReDeployJob JobType = "redeploy"
)

type RepoType string

const (
	RepoPR     RepoType = "pr"
	RepoBranch RepoType = "branch"
)

type DeploymentJob struct {
	ctx  context.Context
	Body *DeploymentServiceParams
}

type workerData struct {
	cancel context.CancelFunc
	// id     int
}

type DeploymentService struct {
	mu           sync.Mutex
	v            *validator.Validate
	q            *db.Queries
	qCtx         context.Context
	docker       *docker.DockerClient
	log          *logbroker.LogBrokerService
	codeStoreDir string
	once         sync.Once
	eg           *errgroup.Group
	egCtx        context.Context
	cancel       context.CancelFunc
	jobs         chan DeploymentJob
	shut         chan struct{}
	workerID     atomic.Int32
	workers      map[int32]*workerData
}

// initializes a new deployment service
func NewDeploymentService(q *db.Queries, docker *docker.DockerClient, log *logbroker.LogBrokerService) *DeploymentService {
	jobs := make(chan DeploymentJob, 100)
	shut := make(chan struct{})
	workers := make(map[int32]*workerData)

	return &DeploymentService{
		qCtx:    context.Background(),
		v:       validator.New(),
		q:       q,
		docker:  docker,
		log:     log,
		eg:      new(errgroup.Group),
		jobs:    jobs,
		shut:    shut,
		workers: workers,
	}
}

// starts the deployment workers to listen for jobs
func (d *DeploymentService) Start(ctx context.Context, codeStorePath string) error {
	d.codeStoreDir = codeStorePath
	// TODO :gets the worker coutn from global settings
	workers := 3

	d.egCtx, d.cancel = context.WithCancel(ctx)
	d.eg, _ = errgroup.WithContext(d.egCtx)

	for range workers {
		d.addWorker()
	}

	return nil
}

// stops the deployment workers gracefully
func (d *DeploymentService) Stop(ctx context.Context) error {
	d.once.Do(func() {
		close(d.shut) // stops new submissions
		close(d.jobs) // stops the wokers
	})

	done := make(chan error, 1)
	go func() {
		done <- d.eg.Wait()
	}()

	select {
	case <-ctx.Done():
		d.cancel()
		return ctx.Err()

	case err := <-done:
		return err
	}
}

// adds a new worker to the deployment service and starts it
func (d *DeploymentService) addWorker() {
	d.mu.Lock()
	defer d.mu.Unlock()

	// generate  a random id
	id := d.workerID.Add(1)

	ctx, cancle := context.WithCancel(context.Background())

	// add woker info
	d.workers[id] = &workerData{
		cancel: cancle,
	}

	d.eg.Go(func() error {
		defer d.removeWorker(id)
		return d.worker(ctx)
	})
}

// removes the given worker from the deployment service data
func (d *DeploymentService) removeWorker(id int32) {
	d.mu.Lock()
	defer d.mu.Unlock()

	delete(d.workers, id)
}

// starts a new worker lokking for deployment jobs
func (d *DeploymentService) worker(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			fmt.Println("Worker  exiting")
			return ctx.Err()

		case j, ok := <-d.jobs:
			if !ok {
				fmt.Printf("Worker exiting, no more jobs\n")
				return nil
			}

			fmt.Println("Processing job:", j.Body.JobType)
			if j.Body.JobType == ReDeployJob {
				d.redeploy(&reDeployData{
					deploymentID: j.Body.DeploymentID,
					swarmService: j.Body.SwarmService,
					isPublic:     j.Body.IsPublic,
					env:          j.Body.Env,
					imgName:      j.Body.ImgName,
				})
			} else {
				d.runDeploymentPipeline(j.ctx, j.Body)
			}
		}
	}
}

// submits a new job to a deployment worker
func (d *DeploymentService) Submit(ctx context.Context, body *DeploymentServiceParams) error {
	select {
	case <-d.egCtx.Done():
		return d.egCtx.Err()

	case <-d.shut:
		return fmt.Errorf("deployment service is shutting down, cannot accept new jobs")

	case <-ctx.Done():
		return ctx.Err()

	case d.jobs <- DeploymentJob{
		ctx:  ctx,
		Body: body,
	}:
		return nil
	}
}

// starts a new worker to the deployment service
func (d *DeploymentService) Incriment() {
	// TODO :update the db also
	d.addWorker()
}

// stops a worker from the deployment service
func (d *DeploymentService) Decriment() error {
	// TODO :update the db also

	d.mu.Lock()

	var cancle context.CancelFunc

	// cancle a random worker
	for _, w := range d.workers {
		cancle = w.cancel
		return nil
	}

	d.mu.Unlock()

	if cancle != nil {
		cancle()
		return nil
	}

	return fmt.Errorf("no workers to remove")
}
