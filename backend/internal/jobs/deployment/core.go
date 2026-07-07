package deployjob

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/Roshan-anand/godploy/internal/db"
	"github.com/Roshan-anand/godploy/internal/jobs/logbroker"
	"github.com/Roshan-anand/godploy/internal/lib/database"
	"github.com/Roshan-anand/godploy/internal/lib/docker"
	"github.com/Roshan-anand/godploy/internal/lib/types"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

// Sentinel errors returned by CancelDeployment.
// Handlers use errors.Is to map each error to the correct HTTP status code.
var (
	ErrCancelFinishedDeployment = errors.New("cannot cancel finished deployment")
	ErrCancelNotOwnedByRebuild  = errors.New("deployment is not owned by an active rebuild")
	ErrCancelInvalidStatus      = errors.New("cannot cancel deployment")
)

// job is a generic wrapper used by the submit[T] dispatcher.
// done is only used in integration tests to synchronously receive the pipeline result.
// Production callers (handlers, utils etc.) must pass nil — the worker discards
// the result when done is nil.
type job[T any] struct {
	ctx  context.Context
	done chan error
	Body *T
}

type workerData struct {
	cancel context.CancelFunc
}

// rebuildEntry tracks one active rebuild job in the in-memory registry.
type rebuildEntry struct {
	cancel       context.CancelFunc
	deploymentID *uuid.UUID // nil until the candidate Deployment row is created
	jobID        int64      // monotonic identity token for stale-cleanup checks
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

	// rebuildMu protects rebuilds, the in-memory registry of active rebuild work
	// keyed by Service ID. It is independent from mu so worker lifecycle changes
	// do not block rebuild coordination.
	rebuildMu    sync.RWMutex
	rebuilds     map[uuid.UUID]*rebuildEntry
	rebuildJobID atomic.Int64
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
			if err := d.runDeploymentPipeline(j.ctx, j.Body); j.done != nil {
				j.done <- err
			}

		case j, ok := <-d.rebuildJobs:
			if !ok {
				fmt.Printf("Worker exiting, no more jobs\n")
				return nil
			}
			fmt.Println("Processing rebuild job for service ID:", j.Body.ServiceID)
			if err := d.RunRebuildPipeline(j.ctx, j.Body); j.done != nil {
				j.done <- err
			}
			// Remove this rebuild from the registry only if it is still the
			// active entry for the Service. This prevents a stale worker from
			// wiping out a newer active rebuild when it finishes.
			d.CleanupRebuild(j.Body.ServiceID, j.Body.JobID)

		case j, ok := <-d.redeployJobs:
			if !ok {
				fmt.Printf("Worker exiting, no more jobs\n")
				return nil
			}
			fmt.Println("Processing redeploy job for deployment ID:", j.Body.DeploymentID)
			if err := d.Redeploy(j.Body); j.done != nil {
				j.done <- err
			}

		case j, ok := <-d.cloneDeployJobs:
			if !ok {
				fmt.Printf("Worker exiting, no more jobs\n")
				return nil
			}
			fmt.Println("Processing clone deploy job for service ID:", j.Body.ServiceID)
			if err := d.runCloneDeployPipeline(j.ctx, j.Body); j.done != nil {
				j.done <- err
			}

		case j, ok := <-d.previewJobs:
			if !ok {
				fmt.Printf("Worker exiting, no more jobs\n")
				return nil
			}
			fmt.Println("Processing create preview job for project ID:", j.Body.ProjectID)
			if err := d.CreatePreviewFromPR(j.ctx, *j.Body); j.done != nil {
				j.done <- err
			}
		}
	}
}

// trySend races the send on ch against all cancellation signals
// so a blocked send never hangs the caller.
// done is forwarded to the job struct; pass nil for production use.
func trySend[T any](d *DeploymentService, ctx context.Context, ch chan<- job[T], body *T, done chan error) error {
	select {
	case <-d.egCtx.Done():
		return d.egCtx.Err()
	case <-d.shut:
		return fmt.Errorf("deployment service is shutting down, cannot accept new jobs")
	case <-ctx.Done():
		return ctx.Err()
	case ch <- job[T]{ctx: ctx, done: done, Body: body}:
		return nil
	}
}

// submit is a generic dispatcher that validates the body and dispatches it to
// the correct job channel via trySend. It is a standalone function (not a
// method) because Go does not allow type parameters on methods.
// done is forwarded to trySend; pass nil for production use.
func submit[T any](d *DeploymentService, ctx context.Context, body *T, done chan error) error {
	if err := d.v.Struct(body); err != nil {
		return fmt.Errorf("validation error: %w", err)
	}

	switch v := any(body).(type) {
	case *DeploymentServiceParams:
		return trySend(d, ctx, d.deployJobs, v, done)
	case *RebuildServiceParams:
		return trySend(d, ctx, d.rebuildJobs, v, done)
	case *ReDeployData:
		return trySend(d, ctx, d.redeployJobs, v, done)
	case *CloneDeployData:
		return trySend(d, ctx, d.cloneDeployJobs, v, done)
	case *CreatePreviewJobParams:
		return trySend(d, ctx, d.previewJobs, v, done)
	default:
		return fmt.Errorf("unsupported job type: %T", body)
	}
}

// AssignDeploy submits a new deploy job.
//
// deploy : uses the fresh app_service details to perform pull, build and deploy for a new deployment.
// It uses the latest commit hash and message from the repo.
//
// done is only needed in integration tests to synchronously receive the
// pipeline result. Production callers (handlers, utils) must pass nil.
func (d *DeploymentService) AssignDeploy(ctx context.Context, body *DeploymentServiceParams, done chan error) error {
	return submit(d, ctx, body, done)
}

// AssignRebuild submits a new rebuild job.
//
// rebuild : uses the exixting app_service details to perform
// pull, build and deploy for a new deployment. It uses the latest commit hash and message from the repo.
// It returns the registry job ID, the context the worker will run under, a
// cancel function for that context, and any error encountered while queuing.
//
// done is only needed in integration tests to synchronously receive the
// pipeline result. Production callers (handlers, utils) must pass nil.
func (d *DeploymentService) AssignRebuild(ctx context.Context, body *RebuildServiceParams, done chan error) (jobID int64, rebuildCtx context.Context, cancel context.CancelFunc, err error) {
	// Register this rebuild in the newest-wins registry keyed by Service ID.
	// If an active rebuild already exists for the same Service, RegisterRebuild
	// cancels it and returns true; cross-Service rebuilds remain isolated.
	// The rebuild job runs with the registry context so replacement by a newer
	// rebuild cancels its work.
	jobID, rebuildCtx, cancel, _ = d.RegisterRebuild(d.egCtx, body.ServiceID, nil)
	body.JobID = jobID

	if err = submit(d, rebuildCtx, body, done); err != nil {
		cancel()
		d.CleanupRebuild(body.ServiceID, jobID)
		return
	}
	return jobID, rebuildCtx, cancel, nil
}

// AssignRedeploy submits a new redeploy job.
//
// redeploy : uses the existing swarm service and updates with give envs and image
//
// done is only needed in integration tests to synchronously receive the
// pipeline result. Production callers (handlers, utils) must pass nil.
func (d *DeploymentService) AssignRedeploy(ctx context.Context, body *ReDeployData, done chan error) error {
	return submit(d, ctx, body, done)
}

// AssignCloneDeploy submits a clone deploy job for preview instances.
//
// done is only needed in integration tests to synchronously receive the
// pipeline result. Production callers (handlers, utils) must pass nil.
func (d *DeploymentService) AssignCloneDeploy(ctx context.Context, body *CloneDeployData, done chan error) error {
	return submit(d, ctx, body, done)
}

// AssignCreatePreview submits a preview creation job.
//
// preview : for given PR/branch, it clones the existing production instance and performs
//
// newdeploy for service realated to PR
//
// redeploy for service not related to PR
//
// done is only needed in integration tests to synchronously receive the
// pipeline result. Production callers (handlers, utils) must pass nil.
func (d *DeploymentService) AssignCreatePreview(ctx context.Context, body *CreatePreviewJobParams, done chan error) error {
	return submit(d, ctx, body, done)
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

// RegisterRebuild records a new active rebuild for serviceID. If an entry
// already exists for the same Service, it is replaced (newest-wins): the
// previous cancel function is invoked and true is returned. Rebuilds for
// different Services are isolated because the registry is keyed by Service ID.
// The returned context is derived from parentCtx so service shutdown also
// cancels the rebuild.
func (d *DeploymentService) RegisterRebuild(parentCtx context.Context, serviceID uuid.UUID, deploymentID *uuid.UUID) (jobID int64, ctx context.Context, cancel context.CancelFunc, replaced bool) {
	d.rebuildMu.Lock()
	defer d.rebuildMu.Unlock()

	if d.rebuilds == nil {
		d.rebuilds = make(map[uuid.UUID]*rebuildEntry)
	}

	jobID = d.rebuildJobID.Add(1)

	var prevCancel context.CancelFunc
	if prev, ok := d.rebuilds[serviceID]; ok {
		replaced = true
		prevCancel = prev.cancel
	}

	ctx, c := context.WithCancel(parentCtx)
	d.rebuilds[serviceID] = &rebuildEntry{
		cancel:       c,
		deploymentID: deploymentID,
		jobID:        jobID,
	}
	cancel = c

	// Cancel the previous job outside the critical section so any callback
	// that re-enters the registry cannot deadlock.
	if prevCancel != nil {
		prevCancel()
	}

	return jobID, ctx, cancel, replaced
}

// CancelRebuild cancels the active rebuild for serviceID only if the registry
// entry still matches jobID. It returns true when an active entry was found and
// canceled. The entry is removed so newer rebuilds are not affected.
func (d *DeploymentService) CancelRebuild(serviceID uuid.UUID, jobID int64) bool {
	d.rebuildMu.Lock()
	defer d.rebuildMu.Unlock()

	entry, ok := d.rebuilds[serviceID]
	if !ok || entry.jobID != jobID {
		return false
	}

	delete(d.rebuilds, serviceID)
	if entry.cancel != nil {
		entry.cancel()
	}
	return true
}

// CleanupRebuild removes the registry entry for serviceID only if it still
// matches jobID. This prevents a stale worker from wiping out a newer active
// rebuild when it finishes. It returns true if the entry was removed.
func (d *DeploymentService) CleanupRebuild(serviceID uuid.UUID, jobID int64) bool {
	d.rebuildMu.Lock()
	defer d.rebuildMu.Unlock()

	entry, ok := d.rebuilds[serviceID]
	if !ok || entry.jobID != jobID {
		return false
	}

	delete(d.rebuilds, serviceID)
	return true
}

// SetRebuildDeploymentID updates the registry entry for serviceID with the
// candidate Deployment ID after the row is created. It only updates the entry
// if it still matches jobID, and returns true when the update succeeded.
func (d *DeploymentService) SetRebuildDeploymentID(serviceID uuid.UUID, jobID int64, deploymentID uuid.UUID) bool {
	d.rebuildMu.Lock()
	defer d.rebuildMu.Unlock()

	entry, ok := d.rebuilds[serviceID]
	if !ok || entry.jobID != jobID {
		return false
	}

	entry.deploymentID = &deploymentID
	d.rebuilds[serviceID] = entry
	return true
}

// GetRebuildDeploymentID returns the candidate Deployment ID recorded for the
// active rebuild, or nil if the entry does not exist or does not match jobID.
// It is exported for integration tests that need to observe the moment the
// candidate row is created without polling the database.
func (d *DeploymentService) GetRebuildDeploymentID(serviceID uuid.UUID, jobID int64) *uuid.UUID {
	d.rebuildMu.RLock()
	defer d.rebuildMu.RUnlock()

	entry, ok := d.rebuilds[serviceID]
	if !ok || entry.jobID != jobID {
		return nil
	}
	return entry.deploymentID
}

// DrainRebuildRegistry clears all active rebuild entries and resets the job
// sequence. It is used on startup to ensure a restarted server does not keep
// stale cancel functions from a previous process. Cancel functions are invoked
// outside the lock to avoid deadlocks with callbacks that re-enter the registry.
func (d *DeploymentService) DrainRebuildRegistry() {
	d.rebuildMu.Lock()
	toCancel := make([]context.CancelFunc, 0, len(d.rebuilds))
	for _, entry := range d.rebuilds {
		if entry.cancel != nil {
			toCancel = append(toCancel, entry.cancel)
		}
	}
	d.rebuilds = make(map[uuid.UUID]*rebuildEntry)
	d.rebuildJobID.Store(0)
	d.rebuildMu.Unlock()

	for _, cancel := range toCancel {
		cancel()
	}
}

// CleanupInterruptedDeployments runs on server startup. It marks any deployments
// that were queued or building when the server stopped as error, finalizes
// their log streams, and drains the in-memory rebuild registry. The function is
// idempotent: repeated calls are safe and become no-ops once interrupted work
// has been cleaned up.
func (d *DeploymentService) CleanupInterruptedDeployments(ctx context.Context) error {
	const msg = "interrupted by server restart"

	statuses := []types.DeploymentStatus{types.DeploymentQueued, types.DeploymentBuilding}
	var interrupted []db.Deployment
	for _, status := range statuses {
		deps, err := d.db.Queries.GetDeploymentsByStatus(ctx, status)
		if err != nil {
			return fmt.Errorf("list %s deployments: %w", status, err)
		}
		interrupted = append(interrupted, deps...)
	}

	for _, dep := range interrupted {
		if err := d.db.Queries.UpdateDeploymentStatus(ctx, db.UpdateDeploymentStatusParams{
			ID:     dep.ID,
			Status: types.DeploymentError,
		}); err != nil {
			fmt.Printf("CleanupInterruptedDeployments: failed to mark deployment %s error: %v\n", dep.ID, err)
			continue
		}

		if err := d.log.EndLogs(&logbroker.EndLogData{
			DeploymentID: dep.ID,
			Status:       types.DeploymentError,
			Message:      errorMsg(msg),
		}); err != nil {
			fmt.Printf("CleanupInterruptedDeployments: failed to finalize logs for deployment %s: %v\n", dep.ID, err)
		}
	}

	d.DrainRebuildRegistry()
	return nil
}

// CancelActiveRebuild cancels the active rebuild for serviceID, if any.
// It is exported for integration tests; production code should prefer
// cancelRebuild when the job identity is known.
func (d *DeploymentService) CancelActiveRebuild(serviceID uuid.UUID) bool {
	d.rebuildMu.Lock()
	defer d.rebuildMu.Unlock()

	entry, ok := d.rebuilds[serviceID]
	if !ok {
		return false
	}

	delete(d.rebuilds, serviceID)
	if entry.cancel != nil {
		entry.cancel()
	}
	return true
}

// ActiveRebuildCount returns the number of active rebuilds in the registry.
// It is exported for integration tests.
func (d *DeploymentService) ActiveRebuildCount() int {
	d.rebuildMu.RLock()
	defer d.rebuildMu.RUnlock()

	return len(d.rebuilds)
}

// HasActiveRebuild reports whether an active rebuild exists for serviceID.
// Handlers use this to reject redeploy and rollback actions while a rebuild
// is in progress without exposing the internal registry structure.
func (d *DeploymentService) HasActiveRebuild(serviceID uuid.UUID) bool {
	d.rebuildMu.RLock()
	defer d.rebuildMu.RUnlock()

	_, ok := d.rebuilds[serviceID]
	return ok
}

// CancelDeployment cancels the Deployment identified by deploymentID.
//
// It is idempotent: an already canceled Deployment returns nil. Finished
// Deployments (ready, error, inactive, pruned, paused) are rejected. For
// queued or building Deployments, the method triggers the active rebuild's
// cancel function when the Deployment is the one tracked by the registry,
// updates the Deployment status to canceled, and finalizes its log stream.
func (d *DeploymentService) CancelDeployment(deploymentID uuid.UUID) error {
	ctx := context.Background()

	deployment, err := d.db.Queries.GetDeployment(ctx, deploymentID)
	if err != nil {
		return fmt.Errorf("deployment not found: %w", err)
	}

	switch deployment.Status {
	case types.DeploymentCanceled:
		return nil
	case types.DeploymentReady, types.DeploymentError, types.DeploymentInactive, types.DeploymentPruned, types.DeploymentPaused:
		return ErrCancelFinishedDeployment
	case types.DeploymentQueued, types.DeploymentBuilding:
		// ok
	default:
		return fmt.Errorf("%w with status %q", ErrCancelInvalidStatus, deployment.Status)
	}

	// Find the active rebuild entry that owns this Deployment and trigger its
	// cancel. We only call cancel when the registry still points to the same
	// candidate; otherwise the Deployment is not owned by an active rebuild.
	var cancel context.CancelFunc
	d.rebuildMu.RLock()
	entry, ok := d.rebuilds[deployment.ServiceID]
	if ok && entry.deploymentID != nil && *entry.deploymentID == deploymentID {
		cancel = entry.cancel
	}
	d.rebuildMu.RUnlock()

	if cancel == nil {
		return ErrCancelNotOwnedByRebuild
	}

	// Trigger the worker's context cancellation. The worker will also mark the
	// Deployment canceled, but we update synchronously so the HTTP response
	// reflects the final state even for queued jobs that may not have started.
	cancel()

	if err := d.db.Queries.UpdateDeploymentStatus(ctx, db.UpdateDeploymentStatusParams{
		ID:     deploymentID,
		Status: types.DeploymentCanceled,
	}); err != nil {
		return fmt.Errorf("failed to update deployment status: %w", err)
	}

	return nil
}
