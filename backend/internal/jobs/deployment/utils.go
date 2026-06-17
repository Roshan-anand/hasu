package deployjob

import (
	"bufio"
	"context"
	"database/sql"
	"errors"
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
	BuildArgs         []string `validate:"required"`
	BuildSecrets      []string `validate:"required"`
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
			if !errors.Is(err, io.EOF) {
				fmt.Println("stdout read error:", err)
			}
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

	go scanAndPublish(l, dID, ptmx)

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("%s:err:cmd:wait: %v\n", worker, err)
	}
	return nil
}

// returns a base service spec for the given parameters
func (d *deployData) getBaseSpec() *swarm.ServiceSpec {

	spec := &swarm.ServiceSpec{
		Annotations: swarm.Annotations{
			Name: d.swarmService,
			Labels: map[string]string{
				fmt.Sprintf("traefik.http.routers.%s.entrypoints", d.swarmService):               "websecure",
				fmt.Sprintf("traefik.http.services.%s.loadbalancer.server.port", d.swarmService): "80",
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

	return spec
}

// helper function to get the docker build command based on the given parameters
func (d *DeploymentServiceUtils) getDockerBuildCmd(outputPath string) *exec.Cmd {
	// 	"--secret", "id=npm_token,src=/tmp/npm_token",
	// 	"--secret", "id=github_token,src=/tmp/github_token",

	cmd := exec.Command("docker", "build", "--progress=plain")

	if d.DockerFilePath != "" {
		cmd.Args = append(cmd.Args, "--file", d.DockerFilePath)
	}

	// Guard against empty build args that break docker buildx parsing.
	for _, arg := range d.BuildArgs {
		trimmed := strings.TrimSpace(arg)
		if trimmed == "" || strings.HasPrefix(trimmed, "=") {
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

	dockerCtxPath := path.Join(outputPath + d.DockerContextPath)
	cmd.Args = append(cmd.Args, dockerCtxPath)

	return cmd
}

// helper function to get the git clone command based on the repo type
func (d *DeploymentServiceUtils) getCloneRepoCmd() *exec.Cmd {
	repoURL := fmt.Sprintf("https://oauth2:%s@%s", d.Token, d.Url)

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

	return exec.Command("bash", "-c", cmdStr)
}

// helper fucntion to fill deploy data
func (d *DeploymentServiceParams) getDeployData(network string) *deployData {
	return &deployData{
		deploymentID: d.DeploymentID,
		swarmService: d.SwarmService,
		networkName:  network,
		isPublic:     d.IsPublic,
		env:          d.Env,
		imgName:      d.ImgName,
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

// helper function to create a new deployment and update the previous deployment status
func (d *RebuildServiceParams) createNewDeploymentData(data *DeploymentService, s *db.GetAppServiceForRebuildRow) (uuid.UUID, error) {
	var newStatus types.DeploymentStatus
	switch s.DeploymentStatus {
	case types.DeploymentReady:
		newStatus = types.DeploymentInactive
	default:
		newStatus = types.DeploymentPruned
	}

	// start a new db transaction
	tx, err := data.db.Pool.BeginTx(context.Background(), nil)
	if err != nil {
		fmt.Println("RebuildWorker: error starting transaction:", err)
		return uuid.UUID{}, err
	}
	tq := data.db.Queries.WithTx(tx)

	// change deployment status
	// update the previous deployment is_latest to false
	if err := tq.DownGradeDeployment(data.qCtx, db.DownGradeDeploymentParams{
		DeploymentID: s.DeploymentID,
		Status:       newStatus,
	}); err != nil {
		tx.Rollback()
		fmt.Println("RebuildWorker: error downgrading deployment:", err)
		return uuid.UUID{}, err
	}

	// create a new deployment
	dID, err := tq.CreateDeployment(data.qCtx, db.CreateDeploymentParams{
		ID:         security.GeneratePrimaryKey(),
		ServiceID:  s.ID,
		CommitHash: d.CommitHash,
		CommitMsg:  d.CommitMsg,
		IsCurrent:  true,
	})
	if err != nil {
		tx.Rollback()
		fmt.Println("RebuildWorker: error creating new deployment:", err)
		return uuid.UUID{}, err
	}

	if tx.Commit() != nil {
		fmt.Println("RebuildWorker: error committing transaction:", err)
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
func (d *DeploymentServiceUtils) pullCode(data *DeploymentService) error {
	log := data.log
	q := data.db.Queries

	log.PublishLog(&logbroker.PubData{
		ID:  d.DeploymentID,
		Msg: infoMsg("Pulling code from" + d.Url),
	})

	// update the deployment status to building
	if err := q.UpdateDeploymentStatus(context.Background(), db.UpdateDeploymentStatusParams{
		Status: types.DeploymentBuilding,
		ID:     d.DeploymentID,
	}); err != nil {
		fmt.Printf("PullWorker: error updating deployment status: %v\n", err)
	}

	// clone the repo and get the code path
	cmd := d.getCloneRepoCmd()
	if err := runWorkerCmd(log, d.DeploymentID, cmd, "pull"); err != nil {
		return err
	}

	log.PublishLog(&logbroker.PubData{
		ID:  d.DeploymentID,
		Msg: successMsg("Finished pulling " + d.Url),
	})

	return nil
}

// deployment utils function to build the docker image and update the deployment with the image name
func (d *DeploymentServiceUtils) buildImg(data *DeploymentService) error {
	log := data.log
	q := data.db.Queries

	log.PublishLog(&logbroker.PubData{
		ID:  d.DeploymentID,
		Msg: infoMsg("Building the image " + d.ImgName),
	})

	// generate a new docker build cmd
	buildCmd := d.getDockerBuildCmd(d.OutputPath)

	if err := runWorkerCmd(log, d.DeploymentID, buildCmd, "build"); err != nil {
		fmt.Printf("BuildWorker: error running command: %v\n", err)
		return err
	}

	// update the deployment with the built image name
	if err := q.SetDeploymentImageName(data.qCtx, db.SetDeploymentImageNameParams{
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
	go func() {
		if err := os.RemoveAll(d.OutputPath); err != nil {
			fmt.Printf("BuildWorker: error removing code folder: %v\n", err)
		}
		fmt.Println("succesfully removed :", d.OutputPath)
	}()

	return nil
}
