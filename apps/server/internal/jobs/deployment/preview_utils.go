package deployjob

import (
	"context"
	"fmt"

	"github.com/Roshan-anand/godploy/internal/db"
	"github.com/Roshan-anand/godploy/internal/lib/types"
	"github.com/google/uuid"
)

type PreviewCleanupResources struct {
	SwarmServices map[string]struct{}
	Volumes       []string
	Images        []string
	DeploymentIDs []uuid.UUID
}

// DeletePreview marks the preview as deleting and removes all associated resources asynchronously.
func (d *DeploymentService) DeletePreview(ctx context.Context, previewID uuid.UUID) error {
	q := d.db.Queries

	if err := q.UpdateInstanceStatus(ctx, db.UpdateInstanceStatusParams{ID: previewID, Status: types.InstanceDeleting}); err != nil {
		return fmt.Errorf("failed to mark preview as deleting: %w", err)
	}

	go d.cleanupPreview(ctx, previewID)
	return nil
}

// curates swarm service, volume, image and deployment from all services list
func collectCleanupResources(services []db.GetAllServiceForCleanupRow) *PreviewCleanupResources {
	swarmServices := make(map[string]struct{})
	var volumes []string
	var images []string
	var dIDs []uuid.UUID

	for _, svc := range services {
		if svc.SwarmService != "" {
			swarmServices[svc.SwarmService] = struct{}{}
		}
		if svc.Type != types.AppServiceType && svc.Volume != "" {
			volumes = append(volumes, svc.Volume)
		}
		if svc.Type == types.AppServiceType {
			if svc.Image.Valid {
				images = append(images, svc.Image.String)
			}
			dIDs = append(dIDs, svc.DeploymentID)
		}
	}

	return &PreviewCleanupResources{
		SwarmServices: swarmServices,
		Volumes:       volumes,
		Images:        images,
		DeploymentIDs: dIDs,
	}
}

// cleanupPreview removes Docker resources and DB records for a preview instance.
func (d *DeploymentService) cleanupPreview(ctx context.Context, previewID uuid.UUID) {
	q := d.db.Queries

	network, err := q.GetInstanceNetwork(ctx, previewID)
	if err != nil {
		return
	}

	services, err := q.GetAllServiceForCleanup(ctx, previewID)
	if err != nil {
		q.DeleteInstance(ctx, previewID)
		d.docker.RemoveNetworks([]string{network})
		return
	}

	// curate all service resources for cleanup
	r := collectCleanupResources(services)

	// remove all swam service
	if len(r.SwarmServices) > 0 {
		d.docker.RemoveServices(r.SwarmServices)
	}

	// remove all images
	if err := d.docker.RemoveImages(r.Images); err != nil {
		fmt.Printf("failed to remove images for preview %s: %v\n", previewID, err)
	}

	// remove all volumes
	if err := d.docker.RemoveVolumes(r.Volumes); err != nil {
		fmt.Printf("failed to remove volumes for preview %s: %v\n", previewID, err)
	}

	// remove all logs for the preview
	if err := d.badger.DeleteAllLogsByDeploymentID(r.DeploymentIDs); err != nil {
		fmt.Printf("failed to delete logs for preview %s: %v\n", previewID, err)
	}

	if network != "" {
		d.docker.RemoveNetworks([]string{network})
	}

	if err := q.DeleteInstance(ctx, previewID); err != nil {
		fmt.Printf("failed to delete preview instance %s from DB: %v\n", previewID, err)
	}
}

// ListPreviews returns all preview instances for a project.
func (d *DeploymentService) ListPreviews(ctx context.Context, projectID uuid.UUID) ([]db.GetPreviewInstancesByProjectRow, error) {
	return d.db.Queries.GetPreviewInstancesByProject(ctx, projectID)
}
