package deployjob

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"path"
	"strings"

	"github.com/Roshan-anand/godploy/internal/jobs/logbroker"
	"github.com/creack/pty"
	"github.com/docker/docker/api/types/swarm"
	"github.com/google/uuid"
)

// returns a formatted title string for the logs
func getTitle(msg string) string {
	return fmt.Sprintf("\n-----------------------------------\n\n %s \n------------------------------------\n", msg)
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
func (d *DeploymentServiceParams) getDockerBuildCmd(outputPath string) *exec.Cmd {
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
func (d *DeploymentServiceParams) getCloneRepoCmd(codeStoreDir string) (*exec.Cmd, string) {
	outputPath := path.Join(codeStoreDir, d.SwarmService)
	repoUrl := fmt.Sprintf("https://oauth2:%s@%s", d.Token, d.Url)

	var cmdStr string
	if d.RepoType == RepoPR {
		cmdStr = fmt.Sprintf("git clone --depth 1 %s %s && git -C %s checkout %s", repoUrl, outputPath, outputPath, d.Branch)
	} else {
		cmdStr = fmt.Sprintf("git clone --branch %s --depth 1 %s %s", d.Branch, repoUrl, outputPath)
	}

	return exec.Command("bash", "-c", cmdStr), outputPath
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

// helper function to fill redeploy data
func (d *DeploymentServiceParams) getReDeployData() *reDeployData {
	return &reDeployData{
		deploymentID: d.DeploymentID,
		swarmService: d.SwarmService,
		isPublic:     d.IsPublic,
		env:          d.Env,
		imgName:      d.ImgName,
	}
}

// helper function to get the service network, if not exist create a new one
func (d *DeploymentService) getServiceNetwork(instanceID uuid.UUID) (string, error) {
	// get instance network
	network, err := d.q.GetInstanceNetwork(d.qCtx, instanceID)
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
