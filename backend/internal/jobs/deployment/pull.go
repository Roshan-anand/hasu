package deploymentjob

import (
	"context"
	"fmt"
	"time"

	deploymentqueue "github.com/Roshan-anand/godploy/internal/jobs/deployment/queue"
	logbrokerqueue "github.com/Roshan-anand/godploy/internal/jobs/logbroker/queue"
)

// responsible for pulling code and storing it local
func (w *worker) PullWorker(ctx context.Context, data chan *deploymentqueue.PullJobData) {
	fmt.Println("PullWorker: started")
	for {
		select {
		case d, ok := <-data:
			if !ok {
				fmt.Println("PullWorker: data channel closed, exiting")
				return
			}

			fmt.Println("PullWorker: started working ...")

			for i := range 1 {
				w.Server.LogBrokerQ.PublishLog(&logbrokerqueue.PubData{
					ID:  d.DeploymentID,
					Msg: fmt.Sprintf("pull : %v", i),
				})
				time.Sleep(1 * time.Second)
			}

			w.Server.DeploymentQ.EnqueueBuildJob(&deploymentqueue.BuildJobData{
				DeploymentID: d.DeploymentID,
			})
			// repoURL := fmt.Sprintf("https://oauth2:%s@github.com/%s/%s.git", pData.Token, pData.Owner, pData.Repo)

			// outputPath := fmt.Sprintf("/etc/godploy/code/%s-%s-%s", pData.Owner, pData.Repo, pData.Branch)

			// _ = exec.Command("git", "clone", "--branch", pData.Branch, "--depth", "1", repoURL, outputPath)
		case <-ctx.Done():
			fmt.Println("PullWorker: context cancelled, exiting")
			return
		}
	}
}
