package logbroker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Roshan-anand/godploy/internal/config"
	"github.com/Roshan-anand/godploy/internal/db"
	logbrokerqueue "github.com/Roshan-anand/godploy/internal/jobs/logbroker/queue"
	"github.com/Roshan-anand/godploy/internal/lib/types"
	"github.com/google/uuid"
)

type LogBuffer map[uuid.UUID][]string

type LogsBroker struct {
	Server    *config.Server
	LogBuffer LogBuffer
}

func InitLogsBroker(s *config.Server) *LogsBroker {
	logBuffer := make(LogBuffer)
	return &LogsBroker{
		Server:    s,
		LogBuffer: logBuffer,
	}
}

func (job *LogsBroker) LogsBrokerJob(ctx context.Context, pub chan *logbrokerqueue.PubData, end chan *logbrokerqueue.EndLogData) {
	for {
		select {
		case p, ok := <-pub:
			if !ok {
				fmt.Println("Publisher channel closed, exiting logBroker")
				return
			}

			// check for subscribers
			for _, sub := range job.Server.LogBrokerQ.Subscribers {
				if sub.DeploymentID == p.ID {
					// is new subscriber send all logs from buffer
					if sub.IsNew {
						logs := job.bufferGet(p.ID)
						logsB, err := json.Marshal(logs)
						if err != nil {
							fmt.Println("Error marshalling logs to JSON:", err)
							continue
						}
						sub.SSE.SendSSE("logs", logsB)
						sub.IsNew = false
					}

					sub.SSE.SendSSE("log", []byte(p.Msg))
				}
			}

			// push to buffer
			job.bufferPush(p.ID, p.Msg)

		case e, ok := <-end:
			if !ok {
				fmt.Println("End channel closed, exiting logBroker")
				return
			}
			dID := e.DeploymentID

			// push all logs from buffer to badgerDB
			logs := job.bufferGet(dID)
			job.Server.BadgerDB.AddLogs(dID, logs)

			// update deployment status in database
			if err := job.Server.DB.Queries.UpdateDeploymentStatus(context.Background(), db.UpdateDeploymentStatusParams{
				Status: e.Status,
				ID:     dID,
			}); err != nil {
				fmt.Println("Error updating deployment status in database:", err)
			}

			// remove subscribers of the deployment
			for userID, sub := range job.Server.LogBrokerQ.Subscribers {
				if sub.DeploymentID == dID {
					if e.Status == types.DeploymentError {
						if e.Message == "" {
							e.Message = "something went wrong !!"
						}
					}
					sub.SSE.SendSSE("log", []byte(e.Message))
					job.Server.LogBrokerQ.UnsubscribeLogs(userID)
				}
			}
		case <-ctx.Done():
			fmt.Println("Context cancelled, exiting logBroker")
			return
		}
	}
}

// push log message to buffer array
func (job *LogsBroker) bufferPush(dID uuid.UUID, msg string) {
	job.LogBuffer[dID] = append(job.LogBuffer[dID], msg)
}

// get log messages from buffer array
func (job *LogsBroker) bufferGet(dID uuid.UUID) []string {
	return job.LogBuffer[dID]
}
