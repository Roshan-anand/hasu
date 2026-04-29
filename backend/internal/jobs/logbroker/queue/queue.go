package logbrokerqueue

import (
	"github.com/Roshan-anand/godploy/internal/lib/sse"
	"github.com/google/uuid"
)

type PubData struct {
	ID  uuid.UUID
	Msg string
}

type EndLogData struct {
	DeploymentID uuid.UUID
}

type Subscriber struct {
	SSE          *sse.SSE
	DeploymentID uuid.UUID
	IsNew        bool
}

type LogBrokerQueue struct {
	Pub         chan *PubData
	End         chan *EndLogData
	Subscribers map[uuid.UUID]*Subscriber
}

// initializes the log broker queue
func InitLogBrokerQueue() *LogBrokerQueue {
	pub := make(chan *PubData, 10)
	end := make(chan *EndLogData, 10)
	sub := make(map[uuid.UUID]*Subscriber, 10)

	return &LogBrokerQueue{
		Pub:         pub,
		End:         end,
		Subscribers: sub,
	}
}

// closes the publisher channel
func (l *LogBrokerQueue) Close() {
	close(l.Pub)
}

// push log data to the publisher channel
func (l *LogBrokerQueue) PublishLog(data *PubData) {
	l.Pub <- data
}

// subscribe to logs of the deployment
func (l *LogBrokerQueue) SubscribeLogs(userID uuid.UUID, sub *Subscriber) {
	sub.IsNew = true
	l.Subscribers[userID] = sub
}

// unsubscribe user from logs of the deployment
func (l *LogBrokerQueue) UnsubscribeLogs(userID uuid.UUID) {
	delete(l.Subscribers, userID)
}

// push end signal to the end channel
func (l *LogBrokerQueue) EndLogs(data *EndLogData) {
	l.End <- data
}
