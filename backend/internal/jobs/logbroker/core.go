package logbroker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Roshan-anand/godploy/internal/config"
	logbrokerqueue "github.com/Roshan-anand/godploy/internal/jobs/logbroker/queue"
	"github.com/dgraph-io/badger/v4"
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
	fmt.Println("LogBroker: started")
	for {
		select {
		case p, ok := <-pub:
			if !ok {
				fmt.Println("Publisher channel closed, exiting logBroker")
				return
			}
			println("log broker recived message : ", p.Msg)

			// check for subscribers
			for _, sub := range job.Server.LogBrokerQ.Subscribers {
				if sub.DeploymentID == p.ID {
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
			fmt.Printf("Received end signal for deployment ID: %s\n", e.DeploymentID)
			job.EndLogJob(e.DeploymentID)

		case <-ctx.Done():
			fmt.Println("Context cancelled, exiting logBroker")
			return
		}
	}
}

// push all the buffered logs inside db
func (job *LogsBroker) EndLogJob(dID uuid.UUID) {
	logs := job.bufferGet(dID)
	db := job.Server.BadgerDB.Pool
	txn := db.NewTransaction(true)

	for i, log := range logs {
		key := fmt.Sprintf("%s_%d", dID.String(), i)
		if err := txn.Set([]byte(key), []byte(log)); err == badger.ErrTxnTooBig {
			_ = txn.Commit()
			txn = db.NewTransaction(true)
			_ = txn.Set([]byte(key), []byte(log))
		}
	}
	_ = txn.Commit()

	// remove subscribers of the deployment
	for userID, sub := range job.Server.LogBrokerQ.Subscribers {
		if sub.DeploymentID == dID {
			job.Server.LogBrokerQ.UnsubscribeLogs(userID)
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
