package deploymentjob

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/Roshan-anand/godploy/internal/db"
	deploymentqueue "github.com/Roshan-anand/godploy/internal/jobs/deployment/queue"
	logbrokerqueue "github.com/Roshan-anand/godploy/internal/jobs/logbroker/queue"
	"github.com/Roshan-anand/godploy/internal/lib/types"
)

func getDockerBuildCmd(d *deploymentqueue.BuildJobData) *exec.Cmd {
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

	// create a tar achive of the code folder
	dockerCtxPath := path.Join(d.StorePath + d.DockerContextPath)
	cmd.Args = append(cmd.Args, dockerCtxPath)

	return cmd
}

// responsible for pulling code and storing it local
func (w *worker) BuildWorker(ctx context.Context, data chan *deploymentqueue.BuildJobData) {
	for {
		select {
		case d, ok := <-data:
			if !ok {
				fmt.Println("BuildWorker: data channel closed, exiting")
				return
			}

			l := w.Server.LogBrokerQ

			l.PublishLog(&logbrokerqueue.PubData{
				ID:  d.DeploymentID,
				Msg: getTitle("Building the image " + d.ImgName),
			})

			// generate a new docker build cmd
			buildCmd := getDockerBuildCmd(d)

			if err := runWorkerCmd(l, d.DeploymentID, buildCmd, "build"); err != nil {
				fmt.Printf("BuildWorker: error running command: %v\n", err)
				l.EndLogs(&logbrokerqueue.EndLogData{
					DeploymentID: d.DeploymentID,
					Status:       types.DeploymentError,
				})
				continue
			}

			// update the deployment with the built image name
			if err := w.Server.DB.Queries.SetDeploymentImageName(w.qCtx, db.SetDeploymentImageNameParams{
				ID: d.DeploymentID,
				Image: sql.NullString{
					Valid:  true,
					String: d.ImgName,
				},
			}); err != nil {
				fmt.Printf("BuildWorker: error updating deployment image name: %v\n", err)
				l.EndLogs(&logbrokerqueue.EndLogData{
					DeploymentID: d.DeploymentID,
					Status:       types.DeploymentError,
				})
				continue
			}

			// remove the code folder
			go os.RemoveAll(d.StorePath)

			fmt.Println("finished building :", d.ImgName)
			switch d.Type {
			case deploymentqueue.DeployJob:
				// set a deploy worker
				w.Server.DeploymentQ.EnqueueDeployJob(&deploymentqueue.DeployJobData{
					DeploymentID:     d.DeploymentID,
					SwarmServiceName: d.SwarmServiceName,
					ImgName:          d.ImgName,
					Env:              d.Env,
					IsPublic:         d.IsPublic,
					NetworkName:      d.NetworkName,
				})

			case deploymentqueue.RebuildJob:
				w.Server.DeploymentQ.EnqueueRedeployJob(&deploymentqueue.RedeployJobData{
					DeploymentID:     d.DeploymentID,
					SwarmServiceName: d.SwarmServiceName,
					ImgName:          d.ImgName,
					Env:              d.Env,
					IsPublic:         d.IsPublic,
					NetworkName:      d.NetworkName,
				})
			default:
				fmt.Printf("BuildWorker: unknown job type: %v\n", d.Type)
			}

		case <-ctx.Done():
			fmt.Println("BuildWorker: context cancelled, exiting")
			return
		}
	}
}
