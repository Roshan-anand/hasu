package deployjob

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/Roshan-anand/godploy/internal/jobs/logbroker"
	"github.com/Roshan-anand/godploy/internal/lib/database"
	"github.com/Roshan-anand/godploy/internal/lib/docker"
	"github.com/go-playground/validator/v10"
	"golang.org/x/sync/errgroup"
)

// job is a generic wrapper used by the submit[T] dispatcher.
type job[T any] struct {
	ctx  context.Context
	Body *T
}

type workerData struct {
	cancel context.CancelFunc
}

type DeploymentService struct {
	mu           sync.Mutex
	v            *validator.Validate
	db           *database.DataBase
	qCtx         context.Context
	docker       *docker.DockerClient
	log          *logbroker.LogBrokerService
	codeStoreDir string
	once         sync.Once
	eg           *errgroup.Group
	egCtx        context.Context
	cancel       context.CancelFunc
	deployJobs   chan job[DeploymentServiceParams]
	rebuildJobs  chan job[RebuildServiceParams]
	redeployJobs chan job[ReDeployData]
	shut         chan struct{}
	workerID     atomic.Int32
	workers      map[int32]*workerData
}

// NewDeploymentService initializes a new deployment service.
func NewDeploymentService(db *database.DataBase, docker *docker.DockerClient, log *logbroker.LogBrokerService) *DeploymentService {
	return &DeploymentService{
		qCtx:         context.Background(),
		v:            validator.New(),
		db:           db,
		docker:       docker,
		log:          log,
		eg:           new(errgroup.Group),
		deployJobs:   make(chan job[DeploymentServiceParams], 100),
		rebuildJobs:  make(chan job[RebuildServiceParams], 100),
		redeployJobs: make(chan job[ReDeployData], 100),
		shut:         make(chan struct{}),
		workers:      make(map[int32]*workerData),
	}
}

// Start starts the deployment workers to listen for jobs.
func (d *DeploymentService) Start(ctx context.Context, codeStorePath string) error {
	d.codeStoreDir = codeStorePath
	// TODO: gets the worker count from global settings
	workers := 3

	d.egCtx, d.cancel = context.WithCancel(ctx)
	d.eg, _ = errgroup.WithContext(d.egCtx)

	for range workers {
		d.addWorker()
	}

	return nil
}

// Stop stops the deployment workers gracefully.
func (d *DeploymentService) Stop(ctx context.Context) error {
	d.once.Do(func() {
		close(d.shut) // stops new submissions
		close(d.deployJobs)
		close(d.rebuildJobs)
		close(d.redeployJobs)
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

// addWorker adds a new worker to the deployment service and starts it.
func (d *DeploymentService) addWorker() {
	d.mu.Lock()
	defer d.mu.Unlock()

	id := d.workerID.Add(1)
	ctx, cancel := context.WithCancel(context.Background())

	d.workers[id] = &workerData{
		cancel: cancel,
	}

	d.eg.Go(func() error {
		defer d.removeWorker(id)
		return d.worker(ctx)
	})
}

// removeWorker removes the given worker from the deployment service data.
func (d *DeploymentService) removeWorker(id int32) {
	d.mu.Lock()
	defer d.mu.Unlock()

	delete(d.workers, id)
}

// worker listens for jobs on all three channels and dispatches them to the appropriate pipeline.
func (d *DeploymentService) worker(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			fmt.Println("Worker exiting")
			return ctx.Err()

		case j, ok := <-d.deployJobs:
			if !ok {
				fmt.Printf("Worker exiting, no more jobs\n")
				return nil
			}
			fmt.Println("Processing deploy job for deployment ID:", j.Body.DeploymentID)
			d.runDeploymentPipeline(j.ctx, j.Body)

		case j, ok := <-d.rebuildJobs:
			if !ok {
				fmt.Printf("Worker exiting, no more jobs\n")
				return nil
			}
			fmt.Println("Processing rebuild job for service ID:", j.Body.ServiceID)
			d.runRebuildPipeline(j.ctx, j.Body)

		case j, ok := <-d.redeployJobs:
			if !ok {
				fmt.Printf("Worker exiting, no more jobs\n")
				return nil
			}
			fmt.Println("Processing redeploy job for deployment ID:", j.Body.DeploymentID)
			d.redeploy(j.Body)
		}
	}
}

// submit is a generic dispatcher that validates the body and sends it on the given channel.
// It is a standalone function (not a method) because Go does not allow type parameters on methods.
func submit[T any](d *DeploymentService, ctx context.Context, body *T) error {
	if err := d.v.Struct(body); err != nil {
		return fmt.Errorf("validation error: %w", err)
	}

	select {
	case <-d.egCtx.Done():
		return d.egCtx.Err()

	case <-d.shut:
		return fmt.Errorf("deployment service is shutting down, cannot accept new jobs")

	case <-ctx.Done():
		return ctx.Err()

	default:
		switch v := any(body).(type) {
		case *DeploymentServiceParams:
			d.deployJobs <- job[DeploymentServiceParams]{ctx: ctx, Body: v}

		case *RebuildServiceParams:
			d.rebuildJobs <- job[RebuildServiceParams]{ctx: ctx, Body: v}

		case *ReDeployData:
			d.redeployJobs <- job[ReDeployData]{ctx: ctx, Body: v}

		default:
			return fmt.Errorf("unsupported job type: %T", body)
		}

		return nil
	}
}

// AssignDeploy submits a new deploy job.
func (d *DeploymentService) AssignDeploy(ctx context.Context, body *DeploymentServiceParams) error {
	return submit(d, ctx, body)
}

// AssignRebuild submits a new rebuild job.
func (d *DeploymentService) AssignRebuild(ctx context.Context, body *RebuildServiceParams) error {
	return submit(d, ctx, body)
}

// AssignRedeploy submits a new redeploy job.
func (d *DeploymentService) AssignRedeploy(ctx context.Context, body *ReDeployData) error {
	return submit(d, ctx, body)
}

// Incriment adds a new worker to the pool.
func (d *DeploymentService) Incriment() {
	// TODO :update the db also
	d.addWorker()
}

// Decriment stops a random worker from the pool.
func (d *DeploymentService) Decriment() error {
	// TODO: update the db also

	d.mu.Lock()

	var cancel context.CancelFunc

	// cancel a random worker
	for _, w := range d.workers {
		cancel = w.cancel
		break
	}

	d.mu.Unlock()

	if cancel != nil {
		cancel()
		return nil
	}

	return fmt.Errorf("no workers to remove")
}
