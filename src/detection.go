package main

import (
	"container/list"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"maps"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/google/gousb"
	"go.bug.st/serial/enumerator"
)

type Detector struct {
	// список доступных для прошивки устройств
	boards                  map[string]*Device
	serialBoardTemplates    []BoardTemplate
	nonSerialBoardTemplates []BoardTemplate
	mu                      sync.Mutex

	// симуляция плат
	fakeBoards map[string]*Device

	// Список ID типов плат, которые не нужно добавлять, при обновлении.
	// Старые устройства, если они не отсоединялись, останутся в списке, даже если их typeID находится в списке
	dontAddTypes map[int]void

	// Время первого снимка, в котором плата не была обнаружена.
	missingSince map[string]time.Time

	boardActions *list.List
}

func NewDetector() *Detector {
	var d Detector
	d.boards = make(map[string]*Device)
	// добавление фальшивых плат
	d.generateFakeBoards()
	templates := parseBoardTemplateList(deviceListPath)
	for _, template := range templates {
		if template.NoSerial {
			d.nonSerialBoardTemplates = append(d.nonSerialBoardTemplates, template)
		} else {
			d.serialBoardTemplates = append(d.serialBoardTemplates, template)
		}
	}
	d.dontAddTypes = make(map[int]void)
	d.missingSince = make(map[string]time.Time)
	d.boardActions = list.New()
	return &d
}

/*
Обновление текущего списка устройств.

Вовращает:

detectedBoards - все платы, которые удалось обнаружить;

notAddedDevices - список новых устройств, которые были обнаружены, но не были добавлены, так как их типы были добавлены в исключения dontAddTypes;

devicesInList - текущий список плат, без учёта notAddedDevices
*/
func (d *Detector) Update() (
	detectedBoards map[string]*Device,
	notAddedDevices map[string]*Device,
	devicesInList map[string]*Device) {

	d.mu.Lock()
	defer d.mu.Unlock()

	detectedBoards = make(map[string]*Device)

	// так как у некоторых устройств, нет серийного порта, то их приходится искать отдельно
	func() {
		ctx := gousb.NewContext()
		defer ctx.Close()

		_, err := ctx.OpenDevices(func(desc *gousb.DeviceDesc) bool {
			vidpid := VidPid{
				VendorID:  desc.Vendor.String(),
				ProductID: desc.Product.String(),
			}
			boardTemplate := matchTemplate(d.nonSerialBoardTemplates, vidpid)
			if boardTemplate == nil {
				return false
			}

			if boardTemplate.IsBlgMbDevice() {
				printLog("found blg-mb")
				detectedBoards[vidpid.ProductID+vidpid.ProductID+"blg-mb"] = newDevice(*boardTemplate, &BlgMb{})
				return false
			}

			log.Printf("no searching algorithm for this type of device: %v\n", boardTemplate.Type)
			return false
		})

		if err != nil {
			log.Printf("error during searching for USB devices WITHOUT serial ports: %v\n", err)
		}

	}()

	/*
	 * Нужно отсортировать порты мс-тюк
	 * мс-тюк имеет 4 порта
	 * они идут последовательно на unix-подобных системах
	 * нужно собрать все порты мс-тюк в слайс
	 * затем отсортировать их в порядке возрастания
	 * затем поделить их на группы по 4 мс-тюк в группе
	 * одна группа последовательных портов = 1 мс-тюк
	 * если кол-во портов мс-тюк не кратно 4, то что-то пошла не так, это ошибка
	 * имена портов - это строки, однако порты должны идти последовательно согласно их номерам
	 * обычно название порта имеет следующий вид ttyN, где N - это число, tty - это префикс, может принимать другое название
	 * следовательно названия портов следует сортировать как структуру, состоящую из двух элементов: prefix и number, приоритет по prefix, второй приоритет по number
	 * если number отсутствует, то считать его за ноль
	 * все 4 порта мс-тюк должны иметь одинаковый префикс, я так предпологаю...
	 */

	func() {

		type UnixPort struct {
			prefix string
			number int
			originalPortPtr *enumerator.PortDetails
		}

		ports, err := enumerator.GetDetailedPortsList()
		if err != nil {
			log.Printf("error during searching for USB devices WITH serial ports: %v\n", err)
			return
		}

		msUnixPorts := make([]UnixPort, 0, len(ports))

		for _, port := range ports {
			printLog("Found port:", port.Name)
			if !port.IsUSB {
				printLog("The port isn't USB")
				continue
			}
			printLog("   USB VID:", port.VID)
			printLog("   USB PID:", port.PID)
			printLog("   Serial: ", port.SerialNumber)
			printLog("   Product: ", port.Product)

			vidpid := VidPid{
				VendorID:  port.VID,
				ProductID: port.PID,
			}
			boardTemplate := matchTemplate(d.serialBoardTemplates, vidpid)
			if boardTemplate == nil {
				printLog("No template match")
				continue
			}

			var board Board
			if boardTemplate.IsMSDevice() {
				// поиск границы между префиксом и числовой частью в названии порта
				portName := []rune(port.Name)
				numberPartStartIndex := len(portName)
				for i := numberPartStartIndex - 1; i >= 0; i-- {
					if !unicode.IsDigit(portName[i]) {
						break
					}
					numberPartStartIndex = i
				}
				unixport := UnixPort{
					originalPortPtr: port,
					prefix: string(portName[:numberPartStartIndex]),
					number: func() int {
						if numberPartStartIndex == len(portName) {
							printLog("the port doesn't have number ending", portName)
							return 0
						}
						res, err := strconv.Atoi(string(portName[numberPartStartIndex:]))
						if err != nil {
							printLog("something went wrong after attempting to convert port number to int:", portName)
							res = 0
						}
						return res
					}(),
				}
				msUnixPorts = append(msUnixPorts, unixport)
				// сборка портов МС-ТЮК происходит после окончания цикла
				continue
			} else if boardTemplate.IsArduinoDevice() {
				board = NewArduinoFromTemp(
					*boardTemplate,
					port.Name,
					port.SerialNumber,
				)
			} else {
				log.Printf("no searching algorithm for this type of device: %v\n", boardTemplate.Type)
				continue
			}
			var id string
			if port.SerialNumber != "" {
				id = port.SerialNumber
				if _, ok := detectedBoards[id]; ok {
					printLog("device with this serial id has been already added to the detected boards list, id =", id)
					id = port.Name + port.VID + port.PID + port.Product
				}
			} else {
				id = port.Name + port.VID + port.PID + port.Product
			}
			detectedBoards[id] = newDevice(*boardTemplate, board)
		}

		// нет смысла, исполнять дальнейший код, если МС-ТЮК отсутствует
		if len(msUnixPorts) == 0 {
			return
		}

		// один МС-ТЮК = 4 "рядом стоящих" порта
		sort.Slice(msUnixPorts, func(i, j int) bool {
			port1 := msUnixPorts[i]
			port2 := msUnixPorts[j]
			if port1.prefix != port2.prefix {
				return port1.prefix < port2.prefix
			}
			return port1.number < port2.number
		})
		// сборка МС-ТЮК
		if len(msUnixPorts)%4 != 0 {
			log.Printf("something went wrong after gathering together ms1 ports: the number of these ports isn't dividable by 4")
			return
		}
		msTemplate := func() *BoardTemplate {
			for _, template := range d.serialBoardTemplates {
				if template.IsMSDevice() {
					return &template
				}
			}
			return nil
		}()
		if msTemplate == nil {
			log.Printf("can't find ms1 template")
			return
		}
		for i := 0; i < len(msUnixPorts); i += 4 {
			port := msUnixPorts[i]

			portNames := func() [4]string {
				var res [4]string
				for i, v := range msUnixPorts {
					res[i] = v.originalPortPtr.Name
				}
				return res
			}()

			detectedBoards["ms1" + port.originalPortPtr.Name] = newDevice(*msTemplate, NewMS1(portNames))
			printLog("found ms1-device", port.originalPortPtr.Name)
		}
	}()

	// добавление фальшивых плат к действительно обнаруженным
	if fakeBoardsNum > 0 || fakeMSNum > 0 {
		if detectedBoards == nil {
			detectedBoards = make(map[string]*Device)
		}
		maps.Copy(detectedBoards, d.fakeBoards)
	}

	// обновление информации о старых устройствах и добавление новых

	notAddedDevices = make(map[string]*Device)

	for deviceID, newBoard := range detectedBoards {
		// плата вернулась до окончания grace-периода (либо не исчезала).
		delete(d.missingSince, deviceID)
		oldBoard, exists := d.boards[deviceID]
		if exists {
			oldBoard.Mu.Lock()
			switch oldBoard.Board.(type) {
			case *Arduino:
				oldArduino := oldBoard.Board.(*Arduino)
				newArduino := newBoard.Board.(*Arduino)
				if oldArduino.portName != newArduino.portName {
					oldArduino.portName = newArduino.portName
					d.boardActions.PushBack(ActionWithBoard{board: oldBoard, boardID: deviceID, action: PORT_UPDATE})
				}
			case *BlgMb:
				blgBoard := oldBoard.Board.(*BlgMb)
				if blgBoard.version == "" {
					blgBoard.GetVersion()
				}
				if blgBoard.serialID == "" {
					blgBoard.GetId()
				}
			}
			oldBoard.Mu.Unlock()
		} else {
			if _, ok := d.dontAddTypes[newBoard.TypeDesc.ID]; ok {
				notAddedDevices[deviceID] = newBoard
			} else {
				switch newBoard.Board.(type) {
				case *BlgMb:
					blgBoard := newBoard.Board.(*BlgMb)
					blgBoard.GetVersion()
					if blgBoard.serialID == "" {
						blgBoard.GetId()
					}
				}
				d.boards[deviceID] = newBoard
				d.boardActions.PushBack(ActionWithBoard{board: newBoard, boardID: deviceID, action: ADD})
			}
		}
	}

	// удаление
	now := time.Now()
	for deviceID := range d.boards {
		board, exists := detectedBoards[deviceID]
		if !exists {
			missingAt, alreadyMissing := d.missingSince[deviceID]
			if !alreadyMissing {
				d.missingSince[deviceID] = now
				continue
			}
			// краткое исчезновение (например при перезагрузке) не должно удалять устройство
			if now.Sub(missingAt) < deviceDisconnectGrace {
				continue
			}
			d.boardActions.PushBack(ActionWithBoard{board: board, boardID: deviceID, action: DELETE})
			delete(d.boards, deviceID)
			delete(d.missingSince, deviceID)
		}
	}
	devicesInList = d.boards
	return
}

// Возвращает и удаляет первое действие в очереди.
// Если список действий пуст, то возвращает пустое действие и ложь в качестве второй переменной.
// Иначе вовращает действие с платой и истину.
func (d *Detector) PopFrontActionSync() (action ActionWithBoard, exists bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.boardActions.Len() > 0 {
		return d.boardActions.Remove(d.boardActions.Front()).(ActionWithBoard), true
	} else {
		return ActionWithBoard{}, false
	}
}

// попросить дектектора, чтобы он не добавлял новые устройства с данным typeID в список (старые устройства с этим typeID останутся в списке)
func (d *Detector) DontAddThisType(typeID int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.dontAddTypes[typeID] = void{}
}

func (d *Detector) AddThisType(typeID int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.dontAddTypes, typeID)
}

// возвращает устройство, соответствующее ID, существует ли устройство в списке
func (d *Detector) GetBoardSync(ID string) (*Device, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	value, exists := d.boards[ID]
	return value, exists
}

func (d *Detector) AddBoardSync(ID string, board *Device) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.boards[ID] = board
}

// удаляет устройство из списка
func (d *Detector) DeleteBoard(ID string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.boards, ID)
}

// получить количество, подключённых плат
func (d *Detector) boardsNum() int {
	return len(d.boards)
}

// удаляем устройства, которых больше нет и уведомляем об этом всех клиентов
func (d *Detector) DeleteAndAlert(newBoards map[string]*Device, c *WebSocketConnection) {
	d.mu.Lock()
	for deviceID := range detector.boards {
		_, exists := newBoards[deviceID]
		if !exists {
			delete(detector.boards, deviceID)
			DeviceUpdateDelete(deviceID, c)
		}
	}
	d.mu.Unlock()
}

// генерация фальшивых плат, которые будут восприниматься программой как настоящие
func (d *Detector) generateFakeBoards() {
	d.fakeBoards = make(map[string]*Device)

	// фальшивые параметры для фальшивых плат
	controller := "Fake Controller"
	programmer := "Fake Programmer"
	fakeArduinoTemp := BoardTemplate{
		ID:                 -1,
		Name:               "Fake Board",
		PidVid:             []VidPid{},
		Type:               "Fake arduino",
		FlashFileExtension: "hex",
	}
	// генерация фальшивых ардуино-подобных устройств
	for i := 0; i < fakeBoardsNum; i++ {
		fakeID := fmt.Sprintf("fakeid-%d", i)
		fakePort := fmt.Sprintf("fakecom-%d", i)
		newFakeBoard := newDevice(fakeArduinoTemp, &FakeBoard{
			controller: controller,
			programmer: programmer,
			portName:   fakePort,
			serialID:   fakeID,
		})
		d.fakeBoards[fakeID] = newFakeBoard
	}

	fakeMsTemp := BoardTemplate{
		ID:                 -2,
		Name:               "Fake ms1",
		PidVid:             []VidPid{},
		Type:               "Fake ms",
		FlashFileExtension: "bin",
	}
	// генерация фальшивых МС-ТЮК
	for i := 0; i < fakeMSNum; i++ {
		fakeID := fmt.Sprintf("fakeidms-%d", i)
		var fakePorts [4]string
		for j := range 4 {
			fakePorts[j] = fmt.Sprintf("fms-%d", i+j)
		}
		var fakeAddress strings.Builder
		// тут может быть проблема, если количество фальшивых МС-ТЮК огромнно
		for j := 0; j < 16-len(fakeID); j++ {
			fakeAddress.WriteString("0")
		}
		fakeAddress.WriteString(fakeID)
		newFakeBoard := newDevice(
			fakeMsTemp,
			&FakeMS{
				portNames:     fakePorts,
				fakeAddress:   fakeAddress.String(),
				clientAddress: "",
			},
		)
		d.fakeBoards[fakeID] = newFakeBoard
	}
	maps.Copy(d.boards, d.fakeBoards)
}

//go:embed device_list.JSON
var boardTemplatesRaw []byte

/*
Парсинг списка шаблонов устройств и обработка ошибок

pathToList - путь к json-файлу со списком устройств.
Если pathToList - пуст, то используется стандартный список (boardTemplatesRaw)
*/
func parseBoardTemplateList(pathToList string) []BoardTemplate {
	attempt := func(pathToList string) ([]BoardTemplate, error) {
		var templates []BoardTemplate
		if pathToList == "" {
			err := json.Unmarshal(boardTemplatesRaw, &templates)
			return templates, err
		}
		jsonFile, err := os.Open(pathToList)
		if err != nil {
			//log.Println("Can't open json file with custom device list. Standard device list will be used instead.", err.Error())
			return templates, err
		}
		defer jsonFile.Close()
		byteValue, err := io.ReadAll(jsonFile)
		if err != nil {
			//log.Println("Can't read json file with custom device list. Standard device list will be used instead.", err.Error())
			return templates, err
		}
		err = json.Unmarshal(byteValue, &templates)
		if err != nil {
			//log.Println("Can't unmarshal json file with custom device list. Standard device list will be used instead.", err.Error())
			return templates, err
		}

		return templates, nil
	}
	templates, err := attempt(pathToList)
	if err != nil {
		if pathToList == "" {
			log.Fatal("Can't run lapki-flasher because failed to use standart device list. Device detector won't be able to work!", err.Error())
		} else {
			log.Println("Can't use json file with custom device list because of error. Standard device list will be used instead.", err.Error())
			parseBoardTemplateList("")
			return templates
		}
	}
	return templates
}

func (d *Detector) boardExists(deviceID string) bool {
	_, exists := d.boards[deviceID]
	return exists
}
func (d *Detector) boardExistsSync(deviceID string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.boardExists(deviceID)
}
