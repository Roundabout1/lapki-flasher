package main

import (
	stdlog "log"
	"os/exec"
	"strings"
	log "github.com/sirupsen/logrus"
)

// настройка для выбранной ОС
func setupOS() {
	stdlog.SetOutput(new(LogrusWriter))

}

// удаление сообщения "interrupted [code -10]" из консоли

type LogrusWriter int

const interruptedError = "interrupted [code -10]"

func (LogrusWriter) Write(data []byte) (int, error) {
	logmessage := string(data)
	if strings.Contains(logmessage, interruptedError) {
		log.Tracef("gousb_logs:%s", logmessage)
		return len(data), nil
	}
	log.Infof("gousb_logs:%s", logmessage)
	return len(data), nil
}

// перезагрузка порта
func rebootPort(portName string) (err error) {
	// stty 115200 -F /dev/ttyUSB0 raw -echo
	cmd := exec.Command("stty", "1200", "-F", portName, "raw", "-echo")
	_, err = cmd.CombinedOutput()
	if err != nil {
		printLog(cmd.Args, err)
	}
	return err
}
