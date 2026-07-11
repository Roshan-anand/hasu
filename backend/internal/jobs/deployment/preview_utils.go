package deployjob

import (
	"context"
	"fmt"

	"github.com/Roshan-anand/godploy/internal/db"
	"github.com/Roshan-anand/godploy/internal/lib/types"
	"github.com/google/uuid"
)

// --- Direct-access preview lifecycle methods ---
// These run synchronously / fire-and-forget alongside the Assign* functions,
// not through the async worker loop.

// DeletePreview marks the preview as deleting and removes all associated resources asynchronously.
func (d *DeploymentService) DeletePreview(ctx context.Context, previewID uuid.UUID) error {
	q := d.db.Queries
	instance, err := q.GetPreviewInstanceByID(ctx, previewID)
	if err != nil {
		return fmt.Errorf("preview not found: %w", err)
	}

	if err := q.UpdateInstanceStatus(ctx, db.UpdateInstanceStatusParams{ID: previewID, Status: types.InstanceDeleting}); err != nil {
		return fmt.Errorf("failed to mark preview as deleting: %w", err)
	}

	go d.cleanupPreview(ctx, instance.ID, instance.Network)
	return nil
}

// collectCleanupResources gathers swarm service names and volumes from a list of services.
func collectCleanupResources(services []db.GetAllServiceRow) (map[string]struct{}, []string) {
	swarmServices := make(map[string]struct{})
	var volumes []string
	for _, svc := range services {
		if svc.SwarmService != "" {
			swarmServices[svc.SwarmService] = struct{}{}
		}
		if svc.Type != types.AppServiceType && svc.Volume != "" {
			volumes = append(volumes, svc.Volume)
		}
	}
	return swarmServices, volumes
}

// cleanupPreview removes Docker resources and DB records for a preview instance.
func (d *DeploymentService) cleanupPreview(ctx context.Context, previewID uuid.UUID, network string) {
	q := d.db.Queries
	services, err := q.GetAllService(ctx, previewID)
	if err != nil {
		q.DeleteInstance(ctx, previewID)
		d.docker.RemoveNetwork([]string{network})
		return
	}

	swarmServices, volumesToRemove := collectCleanupResources(services)

	if len(swarmServices) > 0 {
		d.docker.RemoveServices(swarmServices)
	}

	for _, vol := range volumesToRemove {
		if err := d.docker.RemoveVolume(vol); err != nil {
			continue
		}
	}

	if network != "" {
		d.docker.RemoveNetwork([]string{network})
	}

	if err := q.DeleteInstance(ctx, previewID); err != nil {
		return
	}
}

// ListPreviews returns all preview instances for a project.
func (d *DeploymentService) ListPreviews(ctx context.Context, projectID uuid.UUID) ([]db.GetPreviewInstancesByProjectRow, error) {
	return d.db.Queries.GetPreviewInstancesByProject(ctx, projectID)
}
