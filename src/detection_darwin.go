package main

import (
	"os/exec"
)

// настройка ОС (для Darwin она не требуется, но она здесь присутствует, чтобы обеспечить совместимость с другими платформами, которые используют свои реализации этой функции)
func setupOS() {}

// перезагрузка порта
func rebootPort(portName string) (err error) {
	// stty 115200 -F /dev/ttyUSB0 raw -echo
	cmd := exec.Command("stty", "-f", portName, "1200")
	_, err = cmd.CombinedOutput()
	printLog(cmd.Args)
	if err != nil {
		printLog(cmd.Args, err)
	}
	return err
}

