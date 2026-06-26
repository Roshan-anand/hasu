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
	"github.com/google/uuid"
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

// CreatePreviewJobParams carries the input for preview creation jobs.
type CreatePreviewJobParams struct {
	ProjectID      uuid.UUID `validate:"required"`
	Name           string    `validate:"required"`
	PRNumber       int
	RepoID         int
	HeadBranch     string `validate:"required"`
	GitSourceType  string `validate:"required,oneof=pr branch"`
	GitSourceValue string `validate:"required"`
	EnvCopy        bool
}

type DeploymentService struct {
	mu              sync.Mutex
	v               *validator.Validate
	db              *database.DataBase
	qCtx            context.Context
	docker          *docker.DockerClient
	log             *logbroker.LogBrokerService
	codeStoreDir    string
	once            sync.Once
	eg              *errgroup.Group
	egCtx           context.Context
	cancel          context.CancelFunc
	deployJobs      chan job[DeploymentServiceParams]
	rebuildJobs     chan job[RebuildServiceParams]
	redeployJobs    chan job[ReDeployData]
	cloneDeployJobs chan job[CloneDeployData]
	previewJobs     chan job[CreatePreviewJobParams]
	shut            chan struct{}
	workerID        atomic.Int32
	workers         map[int32]*workerData
}

// NewDeploymentService initializes a new deployment service.
func NewDeploymentService(db *database.DataBase, docker *docker.DockerClient, log *logbroker.LogBrokerService) *DeploymentService {
	return &DeploymentService{
		qCtx:            context.Background(),
		v:               validator.New(),
		db:              db,
		docker:          docker,
		log:             log,
		eg:              new(errgroup.Group),
		deployJobs:      make(chan job[DeploymentServiceParams], 100),
		rebuildJobs:     make(chan job[RebuildServiceParams], 100),
		redeployJobs:    make(chan job[ReDeployData], 100),
		cloneDeployJobs: make(chan job[CloneDeployData], 100),
		previewJobs:     make(chan job[CreatePreviewJobParams], 100),
		shut:            make(chan struct{}),
		workers:         make(map[int32]*workerData),
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
		close(d.cloneDeployJobs)
		close(d.previewJobs)
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

		case j, ok := <-d.cloneDeployJobs:
			if !ok {
				fmt.Printf("Worker exiting, no more jobs\n")
				return nil
			}
			fmt.Println("Processing clone deploy job for service ID:", j.Body.ServiceID)
			d.runCloneDeployPipeline(j.ctx, j.Body)

		case j, ok := <-d.previewJobs:
			if !ok {
				fmt.Printf("Worker exiting, no more jobs\n")
				return nil
			}
			fmt.Println("Processing create preview job for project ID:", j.Body.ProjectID)
			d.CreatePreviewFromPR(j.ctx, *j.Body)
		}
	}
}

// submit is a generic dispatcher that validates the body and sends it on the given channel.
// It is a standalone function (not a method) because Go does not allow type parameters on methods.
// Type switch is done first to determine the target channel, then each case races the send
// against all cancellation signals so a blocked send never hangs the caller.
func submit[T any](d *DeploymentService, ctx context.Context, body *T) error {
	if err := d.v.Struct(body); err != nil {
		return fmt.Errorf("validation error: %w", err)
	}

	switch v := any(body).(type) {
	case *DeploymentServiceParams:
		select {
		case <-d.egCtx.Done():
			return d.egCtx.Err()
		case <-d.shut:
			return fmt.Errorf("deployment service is shutting down, cannot accept new jobs")
		case <-ctx.Done():
			return ctx.Err()
		case d.deployJobs <- job[DeploymentServiceParams]{ctx: ctx, Body: v}:
			return nil
		}

	case *RebuildServiceParams:
		select {
		case <-d.egCtx.Done():
			return d.egCtx.Err()
		case <-d.shut:
			return fmt.Errorf("deployment service is shutting down, cannot accept new jobs")
		case <-ctx.Done():
			return ctx.Err()
		case d.rebuildJobs <- job[RebuildServiceParams]{ctx: ctx, Body: v}:
			return nil
		}

	case *ReDeployData:
		select {
		case <-d.egCtx.Done():
			return d.egCtx.Err()
		case <-d.shut:
			return fmt.Errorf("deployment service is shutting down, cannot accept new jobs")
		case <-ctx.Done():
			return ctx.Err()
		case d.redeployJobs <- job[ReDeployData]{ctx: ctx, Body: v}:
			return nil
		}

	case *CloneDeployData:
		select {
		case <-d.egCtx.Done():
			return d.egCtx.Err()
		case <-d.shut:
			return fmt.Errorf("deployment service is shutting down, cannot accept new jobs")
		case <-ctx.Done():
			return ctx.Err()
		case d.cloneDeployJobs <- job[CloneDeployData]{ctx: ctx, Body: v}:
			return nil
		}

	case *CreatePreviewJobParams:
		select {
		case <-d.egCtx.Done():
			return d.egCtx.Err()
		case <-d.shut:
			return fmt.Errorf("deployment service is shutting down, cannot accept new jobs")
		case <-ctx.Done():
			return ctx.Err()
		case d.previewJobs <- job[CreatePreviewJobParams]{ctx: ctx, Body: v}:
			return nil
		}

	default:
		return fmt.Errorf("unsupported job type: %T", body)
	}
}

// AssignDeploy submits a new deploy job.
//
// deploy : uses the fresh app_service details to perform pull, build and deploy for a new deployment.
// It uses the latest commit hash and message from the repo.
func (d *DeploymentService) AssignDeploy(ctx context.Context, body *DeploymentServiceParams) error {
	return submit(d, ctx, body)
}

// AssignRebuild submits a new rebuild job.
//
// rebuild : uses the exixting app_service details to perform
// pull, build and deploy for a new deployment. It uses the latest commit hash and message from the repo.
func (d *DeploymentService) AssignRebuild(ctx context.Context, body *RebuildServiceParams) error {
	return submit(d, ctx, body)
}

// AssignRedeploy submits a new redeploy job.
//
// redeploy : uses the existing swarm service and updates with give envs and image
func (d *DeploymentService) AssignRedeploy(ctx context.Context, body *ReDeployData) error {
	return submit(d, ctx, body)
}

// AssignCloneDeploy submits a clone deploy job for preview instances.
func (d *DeploymentService) AssignCloneDeploy(ctx context.Context, body *CloneDeployData) error {
	return submit(d, ctx, body)
}

// AssignCreatePreview submits a preview creation job.
//
// preview : for given PR/branch, it clones the existing production instance and performs
//
// newdeploy for service realated to PR
//
// redeploy for service not related to PR
func (d *DeploymentService) AssignCreatePreview(ctx context.Context, body *CreatePreviewJobParams) error {
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
