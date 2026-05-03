package deploymentjob

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

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

			fmt.Println("PullWorker: started working ...", d.Url)
			l := w.Server.LogBrokerQ

			folderName := strings.ReplaceAll(d.Url, "/", "-")
			repoUrl := fmt.Sprintf("https://oauth2:%s@%s", d.Token, d.Url)
			outputPath := fmt.Sprintf("/etc/godploy/code/[%s-%s]", folderName, d.Branch)

			cmd := exec.Command("git", "clone", "--branch", d.Branch, "--depth", "1", repoUrl, outputPath)
			if err := runWorkerCmd(l, d.DeploymentID, cmd); err != nil {
				fmt.Printf("PullWorker: error running command: %v\n", err)
				l.PublishLog(&logbrokerqueue.PubData{
					ID:  d.DeploymentID,
					Msg: "something went wrong !!",
				})

				l.EndLogs(&logbrokerqueue.EndLogData{
					DeploymentID: d.DeploymentID,
					Status:       types.DeploymentFailed,
				})

			} else {
				w.Server.DeploymentQ.EnqueueBuildJob(&deploymentqueue.BuildJobData{
					DeploymentID: d.DeploymentID,
				})
			}

		case <-ctx.Done():
			fmt.Println("PullWorker: context cancelled, exiting")
			return
		}
	}
}
