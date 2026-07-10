package deployjob

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/Roshan-anand/godploy/internal/db"
	ghservice "github.com/Roshan-anand/godploy/internal/lib/gh"
	"github.com/Roshan-anand/godploy/internal/lib/security"
	"github.com/Roshan-anand/godploy/internal/lib/types"
	"github.com/Roshan-anand/godploy/internal/lib/utils"
	"github.com/docker/docker/api/types/volume"
	"github.com/google/uuid"
)

// appDeployItem captures what deploy action a cloned app service needs.
// Jobs are deferred until after dependencies are cloned so MergeDependencyEnv resolves correctly.
type appDeployItem struct {
	newID         uuid.UUID
	newSwarm      string
	previewDomain string
	isPRMatched   bool
	hasProdImage  bool
	prodImg       string
	dID           uuid.UUID
	branch        string
	svc           db.AppService
}

// CreatePreviewFromPR snapshots the production instance, clones all services,
// rewrites dependency IDs, generates preview domains, and queues deploys.
func (d *DeploymentService) CreatePreviewFromPR(ctx context.Context, input CreatePreviewJobParams) error {
	if err := d.v.Struct(input); err != nil {
		fmt.Println("PreviewWorker : validation error:", err)
		return fmt.Errorf("preview:validate: %w", err)
	}

	q := d.db.Queries

	// Get production instance for the project
	prod, err := q.GetProductionInstanceByProject(ctx, input.ProjectID)
	if err != nil {
		fmt.Println("PreviewWorker : no production instance found:", err)
		return fmt.Errorf("preview:get_prod_instance: %w", err)
	}

	fmt.Println("fetched prod instance data")

	// Create preview instance
	previewSlug := slugify(input.Name)
	previewID := security.GeneratePrimaryKey()
	previewNetwork := fmt.Sprintf("preview-%s", previewSlug)

	if err := d.docker.CreateNetwork(previewNetwork); err != nil {
		fmt.Println("PreviewWorker : failed to create preview network:", err)
		return fmt.Errorf("preview:create_network: %w", err)
	}

	if err := q.CreatePreviewInstance(ctx, db.CreatePreviewInstanceParams{
		ID:             previewID,
		ProjectID:      input.ProjectID,
		IsProduction:   false,
		Name:           input.Name,
		Network:        previewNetwork,
		GitSourceType:  types.GitSourceType(input.GitSourceType),
		GitSourceValue: sql.NullString{Valid: true, String: input.GitSourceValue},
		Status:         types.InstanceCreating,
		CreatedBy:      types.CreatedByWebhook,
	}); err != nil {
		fmt.Println("PreviewWorker : failed to create preview instance:", err)
		return fmt.Errorf("preview:create_instance: %w", err)
	}

	fmt.Println("created preview instance record")

	// Map old service IDs → new service IDs
	// used for remapping dependencies
	idMap := make(map[uuid.UUID]uuid.UUID)

	// Clone all services from production
	if err := d.clonePsqlServices(ctx, q, prod.ID, previewID, previewSlug, previewNetwork, idMap); err != nil {
		fmt.Println("PreviewWorker : clone psql services:", err)
		return fmt.Errorf("preview:clone_psql: %w", err)
	}

	if err := d.cloneRedisServices(ctx, q, prod.ID, previewID, previewSlug, previewNetwork, idMap); err != nil {
		fmt.Println("PreviewWorker : clone redis services:", err)
		return fmt.Errorf("preview:clone_redis: %w", err)
	}

	deployPlan, err := d.cloneAppServices(ctx, q, prod.ID, previewID, previewSlug, input.RepoID, input.HeadBranch, idMap)
	if err != nil {
		fmt.Println("PreviewWorker : clone app services:", err)
		return fmt.Errorf("preview:clone_apps: %w", err)
	}

	// Copy dependencies with remapped IDs
	if err := d.cloneDependencies(ctx, q, prod.ID, idMap); err != nil {
		fmt.Println("PreviewWorker : clone dependencies:", err)
		return fmt.Errorf("preview:clone_deps: %w", err)
	}

	// Trigger deploy jobs after dependencies exist so MergeDependencyEnv resolves correctly
	if err := d.triggerAppServiceDeploys(ctx, q, previewID, previewNetwork, deployPlan); err != nil {
		fmt.Println("PreviewWorker : trigger app deploys:", err)
		return fmt.Errorf("preview:trigger_deploys: %w", err)
	}

	// Mark instance ready
	if err := q.UpdateInstanceStatus(ctx, db.UpdateInstanceStatusParams{ID: previewID, Status: types.InstanceReady}); err != nil {
		fmt.Println("PreviewWorker : failed to set preview ready:", err)
		return fmt.Errorf("preview:set_ready: %w", err)
	}

	return nil
}

// clonePsqlServices fetches all PSQL services from the production instance and clones them to the preview.
func (d *DeploymentService) clonePsqlServices(ctx context.Context, q *db.Queries, prodID, previewID uuid.UUID, previewSlug, previewNetwork string, idMap map[uuid.UUID]uuid.UUID) error {
	psqlSvcs, err := q.GetPsqlServicesByInstanceID(ctx, prodID)
	if err != nil {
		return fmt.Errorf("failed to load psql services: %w", err)
	}

	for _, svc := range psqlSvcs {
		newID := security.GeneratePrimaryKey()
		idMap[svc.ID] = newID
		newSwarm := fmt.Sprintf("%s-%s", previewSlug, svc.Name)
		newVol := fmt.Sprintf("volume-%s", newID)

		if _, err := d.docker.Client.VolumeCreate(ctx, volume.CreateOptions{
			Name:   newVol,
			Driver: "local",
		}); err != nil {
			return fmt.Errorf("failed to create psql volume: %w", err)
		}

		env := []string{
			fmt.Sprintf("POSTGRES_DB=%s", svc.DbName),
			fmt.Sprintf("POSTGRES_USER=%s", svc.DbUser),
			fmt.Sprintf("POSTGRES_PASSWORD=%s", svc.DbPassword),
			"sslmode=disable",
		}
		if err := DeployPredefinedService(ctx, d.docker, previewNetwork, newSwarm, svc.Image, env, newVol, PSQLMountTarget); err != nil {
			return fmt.Errorf("failed to deploy preview psql: %w", err)
		}

		internalURL := fmt.Sprintf("http://%s:5432", newSwarm)
		if _, err := q.CreatePsqlService(ctx, db.CreatePsqlServiceParams{
			ID:           newID,
			InstanceID:   previewID,
			Type:         svc.Type,
			Status:       types.PredefServiceRunning,
			SwarmService: newSwarm,
			Name:         svc.Name,
			DbName:       svc.DbName,
			DbUser:       svc.DbUser,
			DbPassword:   svc.DbPassword,
			Image:        svc.Image,
			Volume:       newVol,
			InternalUrl:  internalURL,
		}); err != nil {
			return fmt.Errorf("failed to create preview psql record: %w", err)
		}
	}

	return nil
}

// cloneRedisServices fetches all Redis services from the production instance and clones them to the preview.
func (d *DeploymentService) cloneRedisServices(ctx context.Context, q *db.Queries, prodID, previewID uuid.UUID, previewSlug, previewNetwork string, idMap map[uuid.UUID]uuid.UUID) error {
	redisSvcs, err := q.GetRedisServicesByInstanceID(ctx, prodID)
	if err != nil {
		return fmt.Errorf("failed to load redis services: %w", err)
	}

	for _, svc := range redisSvcs {
		newID := security.GeneratePrimaryKey()
		idMap[svc.ID] = newID
		newSwarm := fmt.Sprintf("%s-%s", previewSlug, svc.Name)
		newVol := fmt.Sprintf("volume-%s", newID)

		if _, err := d.docker.Client.VolumeCreate(ctx, volume.CreateOptions{
			Name:   newVol,
			Driver: "local",
		}); err != nil {
			return fmt.Errorf("failed to create redis volume: %w", err)
		}

		env := []string{}
		if svc.Password != "" {
			env = append(env, fmt.Sprintf("REDIS_PASSWORD=%s", svc.Password))
		}
		if err := DeployPredefinedService(ctx, d.docker, previewNetwork, newSwarm, svc.Image, env, newVol, RedisMountTarget); err != nil {
			return fmt.Errorf("failed to deploy preview redis: %w", err)
		}

		internalURL := fmt.Sprintf("http://%s:6379", newSwarm)
		if _, err := q.CreateRedisService(ctx, db.CreateRedisServiceParams{
			ID:           newID,
			InstanceID:   previewID,
			Type:         svc.Type,
			Status:       types.PredefServiceRunning,
			SwarmService: newSwarm,
			Name:         svc.Name,
			Password:     svc.Password,
			Image:        svc.Image,
			Volume:       newVol,
			InternalUrl:  internalURL,
		}); err != nil {
			return fmt.Errorf("failed to create preview redis record: %w", err)
		}
	}

	return nil
}

// cloneAppServices fetches all app services from the production instance, clones them to the preview,
// sets up PR-matched branches and domains, creates deployments, and returns a deploy plan.
// Deploy jobs are intentionally NOT queued here — they run after dependencies are cloned.
func (d *DeploymentService) cloneAppServices(ctx context.Context, q *db.Queries, prodID, previewID uuid.UUID, previewSlug string, repoID int, headBranch string, idMap map[uuid.UUID]uuid.UUID) ([]appDeployItem, error) {
	appSvcs, err := q.GetFullAppServicesByInstanceId(ctx, prodID)
	if err != nil {
		return nil, fmt.Errorf("failed to load app services: %w", err)
	}

	var plan []appDeployItem

	for _, svc := range appSvcs {
		newID := security.GeneratePrimaryKey()
		idMap[svc.ID] = newID
		newSwarm := fmt.Sprintf("%s-%s", previewSlug, svc.Name)
		branch := svc.Branch
		isPRMatched := false
		if repoID > 0 && int(svc.GhRepoID) == repoID {
			branch = headBranch
			isPRMatched = true
		}

		var previewDomain string
		if svc.IsPublic && svc.Domain.Valid {
			previewDomain = fmt.Sprintf("%s.%s", previewSlug, svc.Domain.String)
		}

		internalURL := fmt.Sprintf("http://%s:%d", newSwarm, svc.Port)
		if _, err := q.CreateAppService(ctx, db.CreateAppServiceParams{
			ID:                newID,
			InstanceID:        previewID,
			Type:              svc.Type,
			Name:              svc.Name,
			GitProvider:       svc.GitProvider,
			GhAppID:           svc.GhAppID,
			GhRepoID:          svc.GhRepoID,
			GhRepoName:        svc.GhRepoName,
			GhRepoUrl:         svc.GhRepoUrl,
			BuildPath:         svc.BuildPath,
			WatchPath:         svc.WatchPath,
			Env:               svc.Env,
			BuildSecrets:      svc.BuildSecrets,
			DockerFilepath:    svc.DockerFilepath,
			DockerContextpath: svc.DockerContextpath,
			DockerBuildstage:  svc.DockerBuildstage,
			IsPublic:          svc.IsPublic,
			Branch:            branch,
			SwarmService:      newSwarm,
			Domain:            sql.NullString{Valid: previewDomain != "", String: previewDomain},
			InternalUrl:       internalURL,
			Port:              svc.Port,
		}); err != nil {
			return nil, fmt.Errorf("failed to create preview app service: %w", err)
		}

		dID := security.GeneratePrimaryKey()
		if isPRMatched {
			if _, err := q.CreateDeployment(ctx, db.CreateDeploymentParams{
				ID:         dID,
				ServiceID:  newID,
				CommitHash: "",
				CommitMsg:  "preview build",
				IsCurrent:  true,
			}); err != nil {
				return nil, fmt.Errorf("failed to create preview deployment: %w", err)
			}
			plan = append(plan, appDeployItem{
				newID:         newID,
				newSwarm:      newSwarm,
				previewDomain: previewDomain,
				isPRMatched:   true,
				dID:           dID,
				branch:        branch,
				svc:           svc,
			})
			continue
		}

		pinnedDeployment, err := q.GetCurrentDeploymentWithImageByServiceId(ctx, svc.ID)
		if err != nil || pinnedDeployment.Status != "ready" {
			if _, err := q.CreateDeployment(ctx, db.CreateDeploymentParams{
				ID:         dID,
				ServiceID:  newID,
				CommitHash: "",
				CommitMsg:  "preview build",
				IsCurrent:  true,
			}); err != nil {
				return nil, fmt.Errorf("failed to create preview deployment: %w", err)
			}
			plan = append(plan, appDeployItem{
				newID:         newID,
				newSwarm:      newSwarm,
				previewDomain: previewDomain,
				dID:           dID,
				branch:        branch,
				svc:           svc,
			})
			continue
		}

		prodImg := pinnedDeployment.Image
		if !prodImg.Valid || prodImg.String == "" {
			if _, err := q.CreateDeployment(ctx, db.CreateDeploymentParams{
				ID:         dID,
				ServiceID:  newID,
				CommitHash: "",
				CommitMsg:  "preview build",
				IsCurrent:  true,
			}); err != nil {
				return nil, fmt.Errorf("failed to create preview deployment: %w", err)
			}
			plan = append(plan, appDeployItem{
				newID:         newID,
				newSwarm:      newSwarm,
				previewDomain: previewDomain,
				dID:           dID,
				branch:        branch,
				svc:           svc,
			})
			continue
		}

		if _, err := q.CreateDeployment(ctx, db.CreateDeploymentParams{
			ID:         dID,
			ServiceID:  newID,
			CommitHash: "pinned",
			CommitMsg:  "preview pinned clone",
			IsCurrent:  true,
		}); err != nil {
			return nil, fmt.Errorf("failed to create pinned deployment: %w", err)
		}
		if err := q.SetDeploymentImageName(ctx, db.SetDeploymentImageNameParams{
			ID:    dID,
			Image: prodImg,
		}); err != nil {
			return nil, fmt.Errorf("failed to set pinned deployment image: %w", err)
		}

		plan = append(plan, appDeployItem{
			newID:         newID,
			newSwarm:      newSwarm,
			previewDomain: previewDomain,
			hasProdImage:  true,
			prodImg:       prodImg.String,
			dID:           dID,
			branch:        branch,
			svc:           svc,
		})
	}

	return plan, nil
}

// triggerAppServiceDeploys executes the deploy plan queued after dependencies are cloned.
// PR-matched and no-image services get a full deploy pipeline; pinned services get clone-deploy.
func (d *DeploymentService) triggerAppServiceDeploys(ctx context.Context, q *db.Queries, previewID uuid.UUID, previewNetwork string, plan []appDeployItem) error {
	// Cache github clients by GhAppID to avoid repeated token fetches across services.
	ghClients := make(map[int64]*ghservice.GithubService)

	for _, item := range plan {
		envData, err := utils.UnmarshalServiceEnv(&utils.ServiceEnvByte{
			Env:          item.svc.Env,
			BuildSecrets: item.svc.BuildSecrets,
		})
		if err != nil {
			return fmt.Errorf("failed to unmarshal env for service %s: %w", item.svc.Name, err)
		}

		if item.hasProdImage {
			if err := d.AssignCloneDeploy(ctx, &CloneDeployData{
				InstanceID:   previewID,
				ServiceID:    item.newID,
				SwarmService: item.newSwarm,
				NetworkName:  previewNetwork,
				ImgName:      item.prodImg,
				Env:          envData.Env,
				Domain:       item.previewDomain,
				IsPublic:     item.svc.IsPublic,
				Port:         item.svc.Port,
			}, nil); err != nil {
				return fmt.Errorf("failed to assign clone deploy for %s: %w", item.svc.Name, err)
			}
			continue
		}

		// PR-matched or no prod image -> full deploy pipeline
		gh, ok := ghClients[item.svc.GhAppID]
		if !ok {
			ghSvc, err := ghservice.New(q, item.svc.GhAppID)
			if err != nil {
				return fmt.Errorf("failed to create github client for %s: %w", item.svc.Name, err)
			}
			ghClients[item.svc.GhAppID] = ghSvc
			gh = ghSvc
		}

		if err := d.AssignDeploy(ctx, &DeploymentServiceParams{
			DeploymentID:      item.dID,
			InstanceID:        previewID,
			ServiceID:         item.newID,
			Token:             gh.Token,
			Url:               item.svc.GhRepoUrl,
			Branch:            item.branch,
			SwarmService:      item.newSwarm,
			BuildPath:         item.svc.BuildPath,
			DockerFilePath:    item.svc.DockerFilepath,
			DockerContextPath: item.svc.DockerContextpath,
			DockerBuildStage:  item.svc.DockerBuildstage,
			ImgName:           item.newSwarm,
			Env:               envData.Env,
			BuildSecrets:      envData.BuildSecrets,
			IsPublic:          item.svc.IsPublic,
		}, nil); err != nil {
			return fmt.Errorf("failed to assign deploy for %s: %w", item.svc.Name, err)
		}
	}

	return nil
}

// cloneDependencies copies all dependency graph edges from the production instance,
// remapping old service IDs to the newly cloned preview service IDs.
func (d *DeploymentService) cloneDependencies(ctx context.Context, q *db.Queries, prodID uuid.UUID, idMap map[uuid.UUID]uuid.UUID) error {
	deps, err := q.GetDependencyGraphEdges(ctx, prodID)
	if err != nil {
		return fmt.Errorf("failed to load dependency graph: %w", err)
	}
	for _, dep := range deps {
		newSource, ok1 := idMap[dep.SourceServiceID]
		newTarget, ok2 := idMap[dep.TargetServiceID]
		if !ok1 || !ok2 {
			continue
		}
		if _, err := q.CreateServiceDependency(ctx, db.CreateServiceDependencyParams{
			ID:              security.GeneratePrimaryKey(),
			SourceServiceID: newSource,
			TargetServiceID: newTarget,
			TargetCol:       dep.TargetCol,
			EnvKey:          dep.EnvKey,
		}); err != nil {
			return fmt.Errorf("failed to clone dependency: %w", err)
		}
	}
	return nil
}

// slugify converts a name to a URL-safe slug.
func slugify(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	name = strings.ReplaceAll(name, " ", "-")
	return name
}
