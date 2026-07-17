package deployjob

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"

	"github.com/Roshan-anand/godploy/internal/db"
	"github.com/Roshan-anand/godploy/internal/jobs/logbroker"
	"github.com/Roshan-anand/godploy/internal/lib/security"
	"github.com/Roshan-anand/godploy/internal/lib/types"
	"github.com/creack/pty"
	"github.com/docker/docker/api/types/swarm"
	"github.com/google/uuid"
)

type DeploymentServiceUtils struct {
	DeploymentID      uuid.UUID `validate:"required"`
	Token             string    `validate:"required"`
	Url               string    `validate:"required"`
	Branch            string    `validate:"required"`
	OutputPath        string    `validate:"required"`
	BuildPath         string    `validate:"required"`
	DockerFilePath    string
	DockerContextPath string
	DockerBuildStage  string
	ImgName           string   `validate:"required"`
	Env               []string `validate:"required"`
	BuildSecrets      []string `validate:"required"`
	GitProvider       types.GitProvider
}

// scans the reader line by line and publish the logs
func scanAndPublish(l *logbroker.LogBrokerService, dID uuid.UUID, r io.Reader) {
	reader := bufio.NewReader(r)

	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			l.PublishLog(&logbroker.PubData{
				ID:  dID,
				Msg: line,
			})
		}

		if err != nil {
			fmt.Println("scan error :", err)
			break
		}
	}
}

// runs the given cmd in a psuedo terminal and publishes the logs to the log broker
func runWorkerCmd(l *logbroker.LogBrokerService, dID uuid.UUID, cmd *exec.Cmd, worker string) error {
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return fmt.Errorf("%s:err:pty:start: %v", worker, err)
	}
	defer ptmx.Close()

	scanAndPublish(l, dID, ptmx)
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("%s:err:cmd: %v", worker, err)
	}

	return nil
}

// generates a unique output path for the given swarm service.
func getOutputPath(baseDir, swarmService string) string {
	outputPath := path.Join(baseDir, swarmService)
	for i := 1; ; i++ {
		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			break
		}
		outputPath = outputPath + strconv.Itoa(i)
	}
	return outputPath
}

// returns a base service spec for the given parameters
func (d *deployData) getBaseSpec() *swarm.ServiceSpec {
	if d.port == 0 {
		d.port = 80
	}

	spec := &swarm.ServiceSpec{
		Annotations: swarm.Annotations{
			Name: d.swarmService,
			Labels: map[string]string{
				fmt.Sprintf("traefik.http.routers.%s.entrypoints", d.swarmService):               "websecure",
				fmt.Sprintf("traefik.http.services.%s.loadbalancer.server.port", d.swarmService): fmt.Sprintf("%d", d.port),
				fmt.Sprintf("traefik.http.routers.%s.tls.certresolver", d.swarmService):          "le",
				"traefik.constraint-label": "head-proxy",
			},
		},

		TaskTemplate: swarm.TaskSpec{
			ContainerSpec: &swarm.ContainerSpec{
				Image: d.imgName,
				TTY:   false,
			},

			RestartPolicy: &swarm.RestartPolicy{
				Condition: swarm.RestartPolicyConditionAny,
			},

			Networks: []swarm.NetworkAttachmentConfig{
				{
					Target: d.networkName,
				},
			},
		},
	}

	// if the service is public connect to traefik
	if d.isPublic {
		spec.TaskTemplate.Networks = append(spec.TaskTemplate.Networks, swarm.NetworkAttachmentConfig{
			Target: "godploy_traefik_proxy",
		})
		spec.Annotations.Labels["traefik.enable"] = "true"
	} else {
		spec.Annotations.Labels["traefik.enable"] = "false"
	}

	// if env avalable
	if len(d.env) > 0 {
		spec.TaskTemplate.ContainerSpec.Env = d.env
	}

	fmt.Println("getbasespec :", d.domain)
	// if a preview domain is set, add explicit Host rule
	if d.domain != "" {
		spec.Annotations.Labels[fmt.Sprintf("traefik.http.routers.%s.rule", d.swarmService)] = fmt.Sprintf("Host(`%s`)", d.domain)
	}

	return spec
}

// helper function to get the docker build command based on the given parameters
func (d *DeploymentServiceUtils) getDockerBuildCmd(ctx context.Context, outputPath string) *exec.Cmd {
	// 	"--secret", "id=npm_token,src=/tmp/npm_token",
	// 	"--secret", "id=github_token,src=/tmp/github_token",

	cmd := exec.CommandContext(ctx, "docker", "build", "--progress=plain")

	if d.DockerFilePath != "" {
		cmd.Args = append(cmd.Args, "--file", d.DockerFilePath)
	}

	// Canonical merged environment is passed to Docker at build time and runtime.
	for _, arg := range d.Env {
		trimmed := strings.TrimSpace(arg)
		if trimmed == "" {
			continue
		}
		cmd.Args = append(cmd.Args, "--build-arg", trimmed)
	}

	// TODO : add build secrets to the cmd

	if d.ImgName != "" {
		cmd.Args = append(cmd.Args, "--tag", d.ImgName)
	}

	if d.DockerBuildStage != "" {
		cmd.Args = append(cmd.Args, "--target", d.DockerBuildStage)
	}

	dockerCtxPath := path.Join(outputPath, d.BuildPath, d.DockerContextPath)
	cmd.Args = append(cmd.Args, dockerCtxPath)

	return cmd
}

// helper function to get the git clone command based on the repo type
func (d *DeploymentServiceUtils) getCloneRepoCmd(ctx context.Context) *exec.Cmd {
	var repoURL string
	switch d.GitProvider {
	case types.GitLocalProvider:
		repoURL = fmt.Sprintf("file://%s", d.Url)
	default:
		repoURL = fmt.Sprintf("https://oauth2:%s@%s", d.Token, d.Url)
	}

	cmdStr := fmt.Sprintf(`
		git clone --depth 1 %s %s &&
		git -C %s fetch --depth 1 origin %s &&
		git -C %s switch -C deploy_branch FETCH_HEAD
	`,
		strconv.Quote(repoURL),
		strconv.Quote(d.OutputPath),
		strconv.Quote(d.OutputPath),
		strconv.Quote(d.Branch),
		strconv.Quote(d.OutputPath),
	)

	return exec.CommandContext(ctx, "bash", "-c", cmdStr)
}

// helper fucntion to fill deploy data
func (d *DeploymentServiceParams) getDeployData(network string) *deployData {
	return &deployData{
		serviceID:    d.ServiceID,
		deploymentID: d.DeploymentID,
		swarmService: d.SwarmService,
		networkName:  network,
		isPublic:     d.IsPublic,
		env:          d.Env,
		imgName:      d.ImgName,
		domain:       d.Domain,
		port:         d.Port,
	}
}

// helper function to get the service network, if not exist create a new one
func (d *DeploymentService) getServiceNetwork(instanceID uuid.UUID) (string, error) {
	// get instance network
	network, err := d.db.Queries.GetInstanceNetwork(d.qCtx, instanceID)
	if err != nil {
		return "", err
	}

	// create network if not exist
	if err := d.docker.CreateNetwork(network); err != nil {
		fmt.Printf("DeployWorker: error creating network: %v\n", err)

		return "", err
	}

	return network, nil
}

// helper function to create a new rebuild candidate deployment.
// The candidate starts as non-current; promotion to Current happens only after
// Docker accepts the rebuilt image (see runRebuildPipeline / redeploy).
func (d *RebuildServiceParams) createNewDeploymentData(data *DeploymentService, s *db.GetAppServiceForRebuildRow) (uuid.UUID, error) {
	// create a new deployment as a non-current candidate
	dID, err := data.db.Queries.CreateDeployment(data.qCtx, db.CreateDeploymentParams{
		ID:         security.GeneratePrimaryKey(),
		ServiceID:  s.ID,
		CommitHash: d.CommitHash,
		CommitMsg:  d.CommitMsg,
		IsCurrent:  false,
	})
	if err != nil {
		fmt.Println("RebuildWorker: error creating new deployment:", err)
		return uuid.UUID{}, err
	}

	return dID, nil
}

// helper function to create a new instance of deployment utils with validation
func (d *DeploymentService) newDeploymentServiceUtils(data *DeploymentServiceUtils) (*DeploymentServiceUtils, error) {
	if err := d.v.Struct(data); err != nil {
		return nil, err
	}

	return data, nil
}

// deployment utils function to pull the code from the repo and return the output path
func (d *DeploymentServiceUtils) pullCode(ctx context.Context, data *DeploymentService) error {
	log := data.log
	q := data.db.Queries

	log.PublishLog(&logbroker.PubData{
		ID:  d.DeploymentID,
		Msg: infoMsg("Pulling code from" + d.Url),
	})

	// update the deployment status to building
	if err := q.UpdateDeploymentStatus(ctx, db.UpdateDeploymentStatusParams{
		Status: types.DeploymentBuilding,
		ID:     d.DeploymentID,
	}); err != nil {
		fmt.Printf("PullWorker: error updating deployment status: %v\n", err)
	}

	// clone the repo and get the code path
	cmd := d.getCloneRepoCmd(ctx)
	if err := runWorkerCmd(log, d.DeploymentID, cmd, "pull"); err != nil {
		fmt.Printf("PullWorker: error running command: %v\n", err)
		return err
	}

	log.PublishLog(&logbroker.PubData{
		ID:  d.DeploymentID,
		Msg: successMsg("Finished pulling " + d.Url),
	})

	return nil
}

// deployment utils function to build the docker image and update the deployment with the image name
func (d *DeploymentServiceUtils) buildImg(ctx context.Context, data *DeploymentService) error {
	log := data.log
	q := data.db.Queries

	log.PublishLog(&logbroker.PubData{
		ID:  d.DeploymentID,
		Msg: infoMsg("Building the image " + d.ImgName),
	})

	// generate a new docker build cmd
	buildCmd := d.getDockerBuildCmd(ctx, d.OutputPath)

	if err := runWorkerCmd(log, d.DeploymentID, buildCmd, "build"); err != nil {
		fmt.Printf("BuildWorker: error running command: %v\n", err)
		return err
	}

	// update the deployment with the built image name
	if err := q.SetDeploymentImageName(ctx, db.SetDeploymentImageNameParams{
		ID: d.DeploymentID,
		Image: sql.NullString{
			Valid:  true,
			String: d.ImgName,
		},
	}); err != nil {
		fmt.Printf("BuildWorker: error updating deployment image name: %v\n", err)
		return fmt.Errorf("something went wrong")
	}

	log.PublishLog(&logbroker.PubData{
		ID:  d.DeploymentID,
		Msg: successMsg("Finished building image :" + d.ImgName),
	})

	// remove the code folder
	if err := os.RemoveAll(d.OutputPath); err != nil {
		fmt.Printf("BuildWorker: error removing code folder: %v\n", err)
	}
	fmt.Println("succesfully removed :", d.OutputPath)

	return nil
}

// MergeDependencyEnv appends resolved dependency values after user env so dependencies take precedence on key conflicts.
func MergeDependencyEnv(q *db.Queries, sourceServiceID uuid.UUID, manualEnv []string) []string {
	rows, err := q.ResolveDependencyEnv(context.Background(), sourceServiceID)
	if err != nil {
		return manualEnv
	}

	for _, row := range rows {
		manualEnv = append(manualEnv, fmt.Sprintf("%s=%s", row.EnvKey, row.ResolvedValue))
	}

	return manualEnv
}
