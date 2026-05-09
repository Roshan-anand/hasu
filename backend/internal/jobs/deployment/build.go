package deploymentjob

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"path"

	"github.com/Roshan-anand/godploy/internal/db"
	deploymentqueue "github.com/Roshan-anand/godploy/internal/jobs/deployment/queue"
	logbrokerqueue "github.com/Roshan-anand/godploy/internal/jobs/logbroker/queue"
	"github.com/Roshan-anand/godploy/internal/lib/types"
	"github.com/google/uuid"
	"github.com/moby/go-archive"
	"github.com/moby/moby/api/types/jsonstream"
	"github.com/moby/moby/client"
)

func streamImgBuildOutput(res io.ReadCloser, l *logbrokerqueue.LogBrokerQueue, dID uuid.UUID) error {

	decoder := json.NewDecoder(res)

	prevLog := ""

	for {
		var msg jsonstream.Message

		err := decoder.Decode(&msg)
		if err != nil {
			if err == io.EOF {
				break
			}

			panic(err)
		}

		// Normal log output
		if msg.Stream != "" && msg.Stream != prevLog {
			l.PublishLog(&logbrokerqueue.PubData{
				ID:  dID,
				Msg: msg.Stream,
			})
			prevLog = msg.Stream
		}

		// BuildKit status lines
		if msg.Status != "" {
			status := fmt.Sprintf(
				"%s %s\n",
				msg.ID,
				msg.Status,
			)

			fmt.Println("status output", status)

			l.PublishLog(&logbrokerqueue.PubData{
				ID:  dID,
				Msg: status,
			})
		}

		// Errors
		if msg.Error != nil {
			return msg.Error
		}
	}
	return nil
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

			fmt.Println("BuildWorker: started working ...")
			l := w.Server.LogBrokerQ

			dockerCtxPath := path.Join(d.StorePath + d.DockerContextPath)
			// create a tar achive of the code folder
			buildCtx, err := archive.TarWithOptions(dockerCtxPath, &archive.TarOptions{})
			if err != nil {
				fmt.Printf("BuildWorker: error creating tar context: %v\n", err)
				l.EndLogs(&logbrokerqueue.EndLogData{
					DeploymentID: d.DeploymentID,
					Status:       types.DeploymentError,
				})
			}

			// build the image
			buildRes, err := w.Server.Docker.Client.ImageBuild(context.Background(), buildCtx, client.ImageBuildOptions{
				Dockerfile: d.DockerFilePath,
				Target:     d.DockerBuildStage,
				Tags: []string{
					d.ImgName,
				},
			})
			if err != nil {
				fmt.Printf("BuildWorker: error building image: %v\n", err)
				l.EndLogs(&logbrokerqueue.EndLogData{
					DeploymentID: d.DeploymentID,
					Status:       types.DeploymentError,
					Message:      err.Error(),
				})
				continue
			}
			defer buildRes.Body.Close()

			// stream the build output to log broker
			if err := streamImgBuildOutput(buildRes.Body, l, d.DeploymentID); err != nil {
				l.EndLogs(&logbrokerqueue.EndLogData{
					DeploymentID: d.DeploymentID,
					Status:       types.DeploymentError,
					Message:      err.Error(),
				})
				continue
			}

			// update the deployment with the built image name
			if err := w.Server.DB.Queries.SetDeploymentImageName(w.qCtx, db.SetDeploymentImageNameParams{
				ID: d.DeploymentID,
				ImageName: sql.NullString{
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

			// set a deploy worker
			w.Server.DeploymentQ.EnqueueDeployJob(&deploymentqueue.DeployJobData{
				DeploymentID:     d.DeploymentID,
				SwarmServiceName: d.SwarmServiceName,
				ImgName:          d.ImgName,
			})

		case <-ctx.Done():
			fmt.Println("BuildWorker: context cancelled, exiting")
			return
		}
	}
}
