package deploymentjob

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os/exec"

	logbrokerqueue "github.com/Roshan-anand/godploy/internal/jobs/logbroker/queue"
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

func runWorkerCmd(l *logbrokerqueue.LogBrokerQueue, dID uuid.UUID, cmd *exec.Cmd) error {
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return fmt.Errorf("pull:err:pty:start: %v", err)
	}
	defer ptmx.Close()

	go scanAndPublish(l, dID, ptmx)

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("pull:err:cmd:wait: %v\n", err)
	}
	return nil
}
