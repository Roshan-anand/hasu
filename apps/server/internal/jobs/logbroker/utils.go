package logbroker

import "github.com/google/uuid"

// push log message to buffer array
func (l *LogBrokerService) bufferPush(dID uuid.UUID, msg string) {
	l.mu.Lock()
	l.logBuffer[dID] = append(l.logBuffer[dID], msg)
	l.mu.Unlock()
}

// get log messages from buffer array
func (l *LogBrokerService) bufferGet(dID uuid.UUID) []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.logBuffer[dID]
}
