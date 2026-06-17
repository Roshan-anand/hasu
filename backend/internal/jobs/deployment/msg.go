package deployjob

import "fmt"

func successMsg(msg string) string {
	return fmt.Sprintf(
		"\x1b[38;2;40;167;69m[SUCCESS] %s\x1b[0m\r\n",
		msg,
	)
}

func infoMsg(msg string) string {
	return fmt.Sprintf(
		"\x1b[38;2;13;110;253m[INFO] %s\x1b[0m\r\n",
		msg,
	)
}

func warningMsg(msg string) string {
	return fmt.Sprintf(
		"\x1b[38;2;255;193;7m[WARNING] %s\x1b[0m\r\n",
		msg,
	)
}

func errorMsg(msg string) string {
	return fmt.Sprintf(
		"\x1b[38;2;220;53;69m[ERROR] %s\x1b[0m\r\n",
		msg,
	)
}
