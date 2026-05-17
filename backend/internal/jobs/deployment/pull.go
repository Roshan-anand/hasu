package deploymentjob

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"path"

	"github.com/Roshan-anand/godploy/internal/db"
	deploymentqueue "github.com/Roshan-anand/godploy/internal/jobs/deployment/queue"
	logbrokerqueue "github.com/Roshan-anand/godploy/internal/jobs/logbroker/queue"
	"github.com/Roshan-anand/godploy/internal/lib/types"
	"github.com/creack/pty"
	"github.com/google/uuid"
)

func scanAndPublish(l *logbrokerqueue.LogBrokerQueue, dID uuid.UUID, r io.Reader) {
	scanner := bufio.NewScanner(r)

	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)
	for scanner.Scan() {
		l.PublishLog(&logbrokerqueue.PubData{
			ID:  dID,
			Msg: scanner.Text(),
		})
	}
	if err := scanner.Err(); err != nil {
		if !errors.Is(err, io.EOF) {
			fmt.Println("stdout read error:", err)
		}
	}
}

func runWorkerCmd(l *logbrokerqueue.LogBrokerQueue, dID uuid.UUID, cmd *exec.Cmd, worker string) error {
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

// responsible for pulling code and storing it local
func (w *worker) PullWorker(ctx context.Context, data chan *deploymentqueue.PullJobData) {
	for {
		select {
		case d, ok := <-data:
			if !ok {
				fmt.Println("PullWorker: data channel closed, exiting")
				return
			}

			// update the deployment status to building
			if err := w.Server.DB.Queries.UpdateDeploymentStatus(w.qCtx, db.UpdateDeploymentStatusParams{
				Status: types.DeploymentBuilding,
				ID:     d.DeploymentID,
			}); err != nil {
				fmt.Printf("PullWorker: error updating deployment status: %v\n", err)
			}

			l := w.Server.LogBrokerQ

			outputPath := path.Join(w.Server.Config.CodeStoreDir, d.SwarmServiceName)
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

			w.Server.DeploymentQ.EnqueueBuildJob(&deploymentqueue.BuildJobData{
				Type:              d.Type,
				DeploymentID:      d.DeploymentID,
				BuildPath:         d.BuildPath,
				SwarmServiceName:  d.SwarmServiceName,
				StorePath:         outputPath,
				DockerFilePath:    d.DockerFilePath,
				DockerContextPath: d.DockerContextPath,
				DockerBuildStage:  d.DockerBuildStage,
				ImgName:           d.ImgName,
				Env:               d.Env,
				BuildArgs:         d.BuildArgs,
				BuildSecrets:      d.BuildSecrets,
			})

		case <-ctx.Done():
			fmt.Println("PullWorker: context cancelled, exiting")
			return
		}
	}
}
