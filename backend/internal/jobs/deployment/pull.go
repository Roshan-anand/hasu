package deploymentjob

import (
	"context"
	"fmt"
	"os/exec"
	"path"

	"github.com/Roshan-anand/godploy/internal/db"
	deploymentqueue "github.com/Roshan-anand/godploy/internal/jobs/deployment/queue"
	logbrokerqueue "github.com/Roshan-anand/godploy/internal/jobs/logbroker/queue"
	"github.com/Roshan-anand/godploy/internal/lib/types"
)

// responsible for pulling code and storing it local
func (w *worker) PullWorker(ctx context.Context, data chan *deploymentqueue.PullJobData) {
	for {
		select {
		case d, ok := <-data:
			if !ok {
				fmt.Println("PullWorker: data channel closed, exiting")
				return
			}

			l := w.Server.LogBrokerQ

			l.PublishLog(&logbrokerqueue.PubData{
				ID:  d.DeploymentID,
				Msg: getTitle("Pulling code from" + d.Url),
			})

			// update the deployment status to building
			if err := w.Server.DB.Queries.UpdateDeploymentStatus(w.qCtx, db.UpdateDeploymentStatusParams{
				Status: types.DeploymentBuilding,
				ID:     d.DeploymentID,
			}); err != nil {
				fmt.Printf("PullWorker: error updating deployment status: %v\n", err)
			}

			outputPath := path.Join(w.Server.Config.CodeStoreDir, d.SwarmService)
			repoUrl := fmt.Sprintf("https://oauth2:%s@%s", d.Token, d.Url)
			cmd := exec.Command("git", "clone", "--branch", d.Branch, "--depth", "1", repoUrl, outputPath)

			if err := runWorkerCmd(l, d.DeploymentID, cmd, "pull"); err != nil {
				fmt.Printf("PullWorker: error running command: %v\n", err)
				l.EndLogs(&logbrokerqueue.EndLogData{
					DeploymentID: d.DeploymentID,
					Status:       types.DeploymentError,
				})
				continue
			}

			fmt.Println("finished pulling :", d.Url)
			w.Server.DeploymentQ.EnqueueBuildJob(&deploymentqueue.BuildJobData{
				Type:              d.Type,
				DeploymentID:      d.DeploymentID,
				BuildPath:         d.BuildPath,
				SwarmService:      d.SwarmService,
				StorePath:         outputPath,
				DockerFilePath:    d.DockerFilePath,
				DockerContextPath: d.DockerContextPath,
				DockerBuildStage:  d.DockerBuildStage,
				ImgName:           d.ImgName,
				Env:               d.Env,
				BuildArgs:         d.BuildArgs,
				BuildSecrets:      d.BuildSecrets,
				NetworkName:       d.NetworkName,
				IsPublic:          d.IsPublic,
			})

		case <-ctx.Done():
			fmt.Println("PullWorker: context cancelled, exiting")
			return
		}
	}
}
