package deploymentjob

import (
	"context"
	"fmt"
	"time"

	deploymentqueue "github.com/Roshan-anand/godploy/internal/jobs/deployment/queue"
	logbrokerqueue "github.com/Roshan-anand/godploy/internal/jobs/logbroker/queue"
	"github.com/Roshan-anand/godploy/internal/lib/types"
)

// responsible for pulling code and storing it local
func (w *worker) DeployWorker(ctx context.Context, data chan *deploymentqueue.DeployJobData) {
	for {
		select {
		case d, ok := <-data:
			if !ok {
				fmt.Println("DeployWorker: data channel closed, exiting")
				return
			}

			fmt.Println("DeployWorker: started working ...")

			for i := range 1 {
				w.Server.LogBrokerQ.PublishLog(&logbrokerqueue.PubData{
					ID:  d.DeploymentID,
					Msg: fmt.Sprintf("deploy : %v", i),
				})
				time.Sleep(1 * time.Second)
			}

			// end the logs
			w.Server.LogBrokerQ.EndLogs(&logbrokerqueue.EndLogData{
				DeploymentID: d.DeploymentID,
				Status:       types.DeploymentSuccess,
			})

			fmt.Printf("DeployWorker: finished working ...")
		case <-ctx.Done():
			fmt.Println("DeployWorker: context cancelled, exiting")
			return
		}
	}
}
