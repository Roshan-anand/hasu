package logbroker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Roshan-anand/godploy/internal/db"
	"github.com/Roshan-anand/godploy/internal/lib/types"
)

func (l *LogBrokerService) publisher(d *PubData) {
	// check for subscribers
	for _, sub := range l.subscribers {
		if sub.DeploymentID == d.ID {
			// is new subscriber send all logs from buffer
			if sub.IsNew {
				logs := l.bufferGet(d.ID)
				logsB, err := json.Marshal(logs)
				if err != nil {
					fmt.Println("Error marshalling logs to JSON:", err)
					continue
				}
				sub.SSE.SendEvent("logs", logsB)
				sub.IsNew = false
			}

			sub.SSE.SendEvent("log", []byte(d.Msg))
		}
	}

	// push to buffer
	l.bufferPush(d.ID, d.Msg)
}

func (l *LogBrokerService) ender(d *EndLogData) {
	dID := d.DeploymentID

	if d.Status == types.DeploymentError {
		if d.Message == "" {
			d.Message = "something went wrong !!"
		}
	}

	// push all logs from buffer to badgerDB
	logs := l.bufferGet(dID)
	logs = append(logs, d.Message)
	l.badgerDB.AddLogs(dID, logs)

	// update deployment status in database
	if err := l.quries.UpdateDeploymentStatus(context.Background(), db.UpdateDeploymentStatusParams{
		Status: d.Status,
		ID:     dID,
	}); err != nil {
		fmt.Println("Error updating deployment status in database:", err)
	}

	// remove subscribers of the deployment
	for userID, sub := range l.subscribers {
		if sub.DeploymentID == dID {
			sub.SSE.SendEvent("log", []byte(d.Message))
			l.Unsubscribe(userID)
		}
	}
}
