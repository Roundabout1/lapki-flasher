package main

import (
	"os/exec"
	"sort"


)



// настройка ОС (для Windows она не требуется, но она здесь присутствует, чтобы обеспечить совместимость с другими платформами, которые использует свои реализации этой функции)
func setupOS() {}



func sortMsPorts(msPorts []ParsedPort) {
	for msPart := 0; msPart < len(msPorts); msPart += 4 {
		pack := msPorts[msPart : msPart+4]
		sort.Slice(pack, func(i, j int) bool {
			return pack[i].originalPortPtr.Product < pack[j].originalPortPtr.Product
		})
	}
}

// перезагрузка порта
func rebootPort(portName string) (err error) {
	cmd := exec.Command("MODE", portName, "BAUD=1200")
	_, err = cmd.CombinedOutput()
	if err != nil {
		printLog(cmd.Args, err)
	}
	return err
}
