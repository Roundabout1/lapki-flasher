//go:build darwin

package main

import (
	"os/exec"
	"strconv"

	"howett.net/plist"
)

type IOREG struct {
	VendorID  int64 `plist:"idVendor"`
	ProductID int64 `plist:"idProduct"`
	// Время с момента последнего запуска ОС.
	// Используется для идентификации устройства, в случае отсутствия серийного номера
	SessionID    int64   `plist:"sessionID"`
	SerialNumber string  `plist:"USB Serial Number"`
	Port         string  `plist:"IODialinDevice"`
	Children     []IOREG `plist:"IORegistryEntryChildren"`
}

// настройка ОС (для Darwin она не требуется, но она здесь присутствует, чтобы обеспечить совместимость с другими платформами, которые используют свои реализации этой функции)
func setupOS() {}

func IOREGport(plistArr []IOREG, ID string, board *Arduino) (portName string, foundID bool) {
	for _, entry := range plistArr {
		if (entry.SerialNumber == "" && strconv.FormatInt(entry.SessionID, 10) == ID) || entry.SerialNumber == ID {
			detectedBoard := CopyArduino(board)
			detectedBoard.portName = NOT_FOUND
			collectArduinoBoardInfo(entry, detectedBoard)
			if detectedBoard.portName == NOT_FOUND {
				printLog("can't find port name!")
				detectedBoard.portName = NOT_FOUND
				return NOT_FOUND, true
			}
			return detectedBoard.portName, true
		}
	}
	for _, entry := range plistArr {
		port, found := IOREGport(entry.Children, ID, board)
		if found {
			return port, found
		}
	}
	return NOT_FOUND, false
}

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

func collectArduinoBoardInfo(reg IOREG, board *Arduino) (sessionID int64) {
	if reg.SerialNumber != "" {
		board.serialID = reg.SerialNumber
	}
	if reg.Port != "" {
		board.portName = reg.Port
	}
	if reg.SessionID != 0 {
		sessionID = reg.SessionID
	}
	for _, child := range reg.Children {
		res := collectArduinoBoardInfo(child, board)
		if res != 0 {
			sessionID = res
		}
	}
	return sessionID
}

func collectMSBoardInfo(reg IOREG, ports map[string]struct{}) (sessionID int64) {
	if reg.Port != "" {
		ports[reg.Port] = struct{}{}
	}
	if reg.SessionID != 0 {
		sessionID = reg.SessionID
	}
	for _, child := range reg.Children {
		res := collectMSBoardInfo(child, ports)
		if res != 0 {
			sessionID = res
		}
	}
	return sessionID
}

func (board *Arduino) Update() bool {
	cmd := exec.Command("ioreg", "-r", "-c", "IOUSBHostDevice", "-l", "-a")
	plistData, err := cmd.CombinedOutput()
	if err != nil {
		printLog("plist error", string(plistData), err.Error())
		board.portName = NOT_FOUND
		return true
	}
	plistArr := []IOREG{}
	format, err := plist.Unmarshal(plistData, &plistArr)
	if err != nil {
		printLog("unmarshal error:", err.Error(), cmd.String())
		printLog("plint format:", format)
		//printLog(string(plistData))
		board.portName = NOT_FOUND
		return true
	}
	portName, _ := IOREGport(plistArr, board.ardOS.ID, board)
	if portName == NOT_FOUND {
		board.portName = NOT_FOUND
		return true
	}
	if portName != board.portName {
		board.portName = portName
		return true
	}
	return false
}

func (board *MS1) Update() bool {
	return false
}
