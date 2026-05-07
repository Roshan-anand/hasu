package deploymentjob

import (
	"context"
	"fmt"
	"time"

	deploymentqueue "github.com/Roshan-anand/godploy/internal/jobs/deployment/queue"
	logbrokerqueue "github.com/Roshan-anand/godploy/internal/jobs/logbroker/queue"
)

// responsible for pulling code and storing it local
func (w *worker) BuildWorker(ctx context.Context, data chan *deploymentqueue.BuildJobData) {
	for {
		select {
		case d, ok := <-data:
			if !ok {
				fmt.Println("BuildWorker: data channel closed, exiting")
				return
			}

			fmt.Println("BuildWorker: started working ...")

			for i := range 2 {
				w.Server.LogBrokerQ.PublishLog(&logbrokerqueue.PubData{
					ID:  d.DeploymentID,
					Msg: fmt.Sprintf("build : %v", i),
				})
				time.Sleep(1 * time.Second)
			}

			w.Server.DeploymentQ.EnqueueDeployJob(&deploymentqueue.DeployJobData{
				DeploymentID: d.DeploymentID,
				StorePath:    d.StorePath,
			})
		case <-ctx.Done():
			fmt.Println("BuildWorker: context cancelled, exiting")
			return
		}
	}
}
