package main

import (
	"errors"
	"io"
	"os/exec"
	"time"
)

type Arduino struct {
	controller   string
	programmer   string
	serialID     string
	portName     string
	bootloaderID int
	ardOS        ArduinoOS // структура с данными для поиска устройства на определённой ОС
}

func NewArduinoFromTemp(temp BoardTemplate, portName string, ardOS ArduinoOS, serialID string) *Arduino {
	arduino := Arduino{
		controller:   temp.Controller,
		programmer:   temp.Programmer,
		bootloaderID: temp.BootloaderID,
		serialID:     serialID,
		portName:     portName,
		ardOS:        ardOS,
	}
	return &arduino
}

func CopyArduino(board *Arduino) *Arduino {
	return &Arduino{
		controller:   board.controller,
		programmer:   board.programmer,
		bootloaderID: board.bootloaderID,
		serialID:     board.serialID,
		portName:     board.portName,
		ardOS:        board.ardOS,
	}
}

// подключено ли устройство
func (board *Arduino) IsConnected() bool {
	return board.portName != NOT_FOUND
}

func (board *Arduino) GetSerialPort() string {
	return board.portName
}

func (board *Arduino) hasBootloader() bool {
	return board.bootloaderID != -1
}

func (board *Arduino) flashBootloader(filePath string) (string, error) {
	flasherSync.Lock()
	defer flasherSync.Unlock()
	if e := rebootPort(board.portName); e != nil {
		return "Не удалось перезагрузить порт", e
	}
	bootloaderType := board.bootloaderID
	detector.DontAddThisType(bootloaderType)
	defer detector.AddThisType(bootloaderType)
	defer time.Sleep(500 * time.Millisecond)
	var notAddedDevices map[string]*Device
	found := false
	for i := 0; i < 25; i++ {
		// TODO: возможно стоит добавить количество необходимого времени в параметры сервера
		time.Sleep(500 * time.Millisecond)
		printLog("Попытка найти подходящее устройство", i+1)
		_, notAddedDevices, _ = detector.Update()
		sameTypeCnt := 0
		var bootloaderDevice *Device
		for _, dev := range notAddedDevices {
			if dev.typeID == bootloaderType {
				bootloaderDevice = dev
				sameTypeCnt++
				if sameTypeCnt > 1 {
					return "Не удалось опознать Bootloader. Ошибка могла быть вызвана перезагрузкой одного из устройств, либо из-за подключения нового.", errors.New("bootloader: too many")
				}
				found = true
			}
		}
		if found {
			return bootloaderDevice.Board.Flash(filePath)
		}
	}
	return "Не удалось найти Bootloader.", errors.New("bootloader: not found")
}

func (board *Arduino) Flash(filePath string) (string, error) {
	if board.hasBootloader() {
		return board.flashBootloader(filePath)
	}
	flashFile := "flash:w:" + getAbolutePath(filePath) + ":a"
	// без опции "-D" не может прошить Arduino Mega
	args := []string{"-D", "-p", board.controller, "-c", board.programmer, "-P", board.portName, "-U", flashFile}
	if configPath != "" {
		args = append(args, "-C", configPath)
	}
	cmd := exec.Command(avrdudePath, args...)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "StderrPipe error.", err
	}
	slurp, err := io.ReadAll(stderr)
	if err != nil {
		return "io.ReadAll stderr error.", err
	}
	printLog("stderr:", slurp)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "StdoutPipe error.", err
	}
	slurp, err = io.ReadAll(stdout)
	if err != nil {
		return "io.ReadAll stdout error.", err
	}
	printLog("stdout:", slurp)
	if err := cmd.Wait(); err != nil {
		return "cmd.Wait error.", err
	}
	avrdudeMessage := handleFlashResult(string(slurp), err)
	return avrdudeMessage, err
}

func (board *Arduino) hasSerial() bool {
	return board.serialID != NOT_FOUND
}

func (board *Arduino) GetWebMessageType() string {
	return DeviceMsg
}

func (board *Arduino) GetWebMessage(name string, deviceID string) any {
	return DeviceMessage{
		ID:         deviceID,
		Name:       name,
		Controller: board.controller,
		Programmer: board.programmer,
		SerialID:   board.serialID,
		PortName:   board.portName,
	}
}
