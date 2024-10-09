package main

// тест:
// go run . -address=":8081" -avrdudePath="D:\avrdude" -configPath="D:\avrdude" -deviceListPath="device_list.JSON" -adminPassword="1234" -msgSize=1000 -fileSize=1000 -thread=5 -stub=2 -verbose -alwaysUpdate -listCooldown=2 -updateList=5ы
import (
	"flag"
	"fmt"
	"log"
	"sync"
	"time"
)

// тип значения настройки
type SettingType int

const (
	STRING   SettingType = iota
	INT      SettingType = iota
	INT64    SettingType = iota
	BOOL     SettingType = iota
	DURATION SettingType = iota
)

type SettingKey int

// индексы массива с настройками, можно добавлять новые в конец, но НЕ СТОИТ менять порядок уже существующих
const (
	address SettingKey = iota
	avrdudePath
	configPath
	deviceListPath
	adminPassword
	msgSize
	fileSize
	thread
	stub
	verbose
	alwaysUpdate
	listCooldown
	updateList
)

type Checker func(value any) bool

type Setting struct {
	Value        any
	DefaultValue any
	Type         SettingType
	Changable    bool    // можно ли изменить настройку?
	Check        Checker // валидация значения настройки
}

const NUM_SETTINGS = 13

type Settings struct {
	mu   sync.Mutex
	Args [NUM_SETTINGS]*Setting
}

func (settings *Settings) getSettingSync(index SettingKey) *Setting {
	settings.mu.Lock()
	defer settings.mu.Unlock()
	return settings.Args[index]
}

func (settings *Settings) setSettingValueSync(index SettingKey, value any) {
	settings.mu.Lock()
	defer settings.mu.Unlock()
	settings.Args[index].Value = value
}

func (settings *Settings) getAddressSync() string {
	return settings.getSettingSync(address).Value.(string)
}
func (settings *Settings) getAvrdudePathSync() string {
	return settings.getSettingSync(avrdudePath).Value.(string)
}
func (settings *Settings) getConfigPathSync() string {
	return settings.getSettingSync(configPath).Value.(string)
}
func (settings *Settings) getDeviceListPathSync() string {
	return settings.getSettingSync(deviceListPath).Value.(string)
}
func (settings *Settings) getAdminPasswordSync() string {
	return settings.getSettingSync(adminPassword).Value.(string)
}
func (settings *Settings) getMsgSizeSync() int64 {
	return settings.getSettingSync(msgSize).Value.(int64)
}
func (settings *Settings) getFileSizeSync() int {
	return settings.getSettingSync(fileSize).Value.(int)
}
func (settings *Settings) getThreadSync() int {
	return settings.getSettingSync(thread).Value.(int)
}
func (settings *Settings) getStubSync() int {
	return settings.getSettingSync(stub).Value.(int)
}
func (settings *Settings) getVerboseSync() bool {
	return settings.getSettingSync(verbose).Value.(bool)
}
func (settings *Settings) getAlwaysUpdateSync() bool {
	return settings.getSettingSync(alwaysUpdate).Value.(bool)
}
func (settings *Settings) getListCooldownSync() time.Duration {
	return settings.getSettingSync(listCooldown).Value.(time.Duration)
}

func (settings *Settings) getUpdateListSync() time.Duration {
	return settings.getSettingSync(updateList).Value.(time.Duration)
}

var SettingsStorage Settings

func makeSetting(argName string, defaultValue any, argDesc string, setType SettingType, changable bool) *Setting {
	var value any
	switch setType {
	case STRING:
		value = flag.String(argName, defaultValue.(string), argDesc)
	case INT:
		value = flag.Int(argName, defaultValue.(int), argDesc)
	case INT64, DURATION:
		value = flag.Int64(argName, defaultValue.(int64), argDesc)
	case BOOL:
		value = flag.Bool(argName, defaultValue.(bool), argDesc)
	}
	return &Setting{
		Value:        value,
		DefaultValue: defaultValue,
		Type:         setType,
		Changable:    changable,
		Check:        func(value any) bool { return true },
	}
}

func makeSettingWithChecker(argName string, defaultValue any, argDesc string, setType SettingType, changable bool, check Checker) *Setting {
	setting := makeSetting(argName, defaultValue, argDesc, setType, changable)
	setting.Check = check
	return setting
}

// чтение флагов и происвоение им стандартных значений
func setArgs() {
	SettingsStorage.Args[address] = makeSetting(
		"address",
		"localhost:8080",
		"адресс для подключения",
		STRING,
		false,
	)
	SettingsStorage.Args[avrdudePath] = makeSetting(
		"avrdudePath",
		"avrdude",
		"путь к avrdude, используется системный путь по-умолчанию",
		STRING,
		true,
	)
	SettingsStorage.Args[configPath] = makeSetting(
		"configPath",
		"",
		"путь к файлу конфигурации avrdude",
		STRING,
		true,
	)
	SettingsStorage.Args[deviceListPath] = makeSetting(
		"deviceListPath",
		"",
		"путь к JSON-файлу со списком устройств. Если прописан, то заменяет стандартный список устройств, при условии, что не возникнет ошибок, связанных с чтением и открытием JSON-файла, иначе используется стандартный список устройств (по-умолчанию пустая строка, означающая, что будет используется, встроенный в загрузчик список)",
		STRING,
		false,
	)
	SettingsStorage.Args[adminPassword] = makeSetting(
		"adminPassword",
		"",
		"пароль, дающий клиенту доступ к особым функциям сервера, при пустом значении эти функции остаются заблокированными",
		STRING,
		false,
	)
	SettingsStorage.Args[msgSize] = makeSettingWithChecker(
		"msgSize",
		int64(1024),
		"максмальный размер одного сообщения, передаваемого через веб-сокеты (в байтах)",
		INT64,
		false,
		func(value any) bool { return value.(int64) > 0 },
	)
	SettingsStorage.Args[fileSize] = makeSettingWithChecker(
		"fileSize",
		2*1024*1024,
		"максимальный размер файла, загружаемого на сервер (в байтах)",
		INT,
		false,
		func(value any) bool { return value.(int) > 0 },
	)
	SettingsStorage.Args[thread] = makeSettingWithChecker(
		"thread",
		3,
		"максимальное количество потоков (горутин) на обработку запросов на одного клиента",
		INT,
		false,
		func(value any) bool { return value.(int) > 0 },
	)
	SettingsStorage.Args[stub] = makeSettingWithChecker(
		"stub",
		0,
		"количество ненастоящих, симулируемых устройств, которые будут восприниматься как настоящие, применяется для тестирования, при значении 0 или меньше фальшивые устройства не добавляются",
		INT,
		false,
		func(value any) bool { return value.(int) > 0 },
	)
	SettingsStorage.Args[verbose] = makeSetting(
		"verbose",
		false,
		"выводить в консоль подробную информацию",
		BOOL,
		false,
	)
	SettingsStorage.Args[alwaysUpdate] = makeSetting(
		"alwaysUpdate",
		false,
		"всегда искать устройства и обновлять их список, даже когда ни один клиент не подключён (используется для тестирования)",
		BOOL,
		false,
	)
	SettingsStorage.Args[listCooldown] = makeSettingWithChecker(
		"listCooldown",
		int64(2),
		"минимальное время (в секундах), через которое клиент может снова запросить список устройств, игнорируется, если количество клиентов меньше чем 2",
		DURATION,
		false,
		func(value any) bool { return value.(time.Duration) > 0 },
	)
	SettingsStorage.Args[updateList] = makeSettingWithChecker(
		"updateList",
		int64(15),
		"количество секунд между автоматическими обновлениями, не может быть меньше единицы",
		DURATION,
		false,
		func(value any) bool { return value.(time.Duration) > 0 },
	)
	flag.Parse()
	for _, setting := range SettingsStorage.Args {
		switch setting.Type {
		case INT:
			setting.Value = *setting.Value.(*int)
		case INT64:
			setting.Value = *setting.Value.(*int64)
		case STRING:
			setting.Value = *setting.Value.(*string)
		case BOOL:
			setting.Value = *setting.Value.(*bool)
		case DURATION:
			setting.Value = time.Second * time.Duration(*setting.Value.(*int64))
		}
		if !setting.Check(setting.Value) {
			setting.Value = setting.DefaultValue
		}
	}
}

// вывод описания всех параметров с их значениями
func printArgsDesc() {
	webAddressStr := fmt.Sprintf("адрес: %s", SettingsStorage.getAddressSync())
	maxFileSizeStr := fmt.Sprintf("максимальный размер файла: %d", SettingsStorage.getFileSizeSync())
	maxMsgSizeStr := fmt.Sprintf("максимальный размер сообщения: %v", SettingsStorage.getMsgSizeSync())
	maxThreadsPerClientStr := fmt.Sprintf("максимальное количество потоков (горутин) для обработки запросов на одного клиента: %d", SettingsStorage.getThreadSync())
	getListCooldownDurationStr := fmt.Sprintf("перерыв для запроса списка устройств: %v", SettingsStorage.getListCooldownSync())
	updateListTimeStr := fmt.Sprintf("промежуток времени между автоматическими обновлениями: %v", SettingsStorage.getUpdateListSync())
	verboseStr := fmt.Sprintf("вывод подробной информации в консоль: %v", SettingsStorage.getVerboseSync())
	alwaysUpdateStr := fmt.Sprintf("постоянное обновление списка устройств: %v", SettingsStorage.getAlwaysUpdateSync())
	fakeBoardsNumStr := fmt.Sprintf("количество фальшивых устройств: %d", SettingsStorage.getStubSync())
	avrdudePathStr := fmt.Sprintf("путь к avrdude (если написано avrdude, то используется системный путь): %s", SettingsStorage.getAvrdudePathSync())
	configPathStr := fmt.Sprintf("путь к файлу конфигурации avrdude: %s", SettingsStorage.getConfigPathSync())
	deviceListPathStr := fmt.Sprintf("путь к файлу со списком устройств (если пусто, то используется встроенный список): %s", SettingsStorage.getDeviceListPathSync())
	adminPasswordStr := fmt.Sprintf("пароль администратора (если пусто, то функции администратора заблокированы): %s", SettingsStorage.getAdminPasswordSync())
	log.Printf("Модуль загрузчика запущен со следующими параметрами:\n %s\n %s\n %s\n %s\n %s\n %s\n %s\n %s\n %s\n %s\n %s\n %s\n %s\n",
		webAddressStr,
		maxFileSizeStr,
		maxMsgSizeStr,
		maxThreadsPerClientStr,
		getListCooldownDurationStr,
		updateListTimeStr,
		verboseStr,
		alwaysUpdateStr,
		fakeBoardsNumStr,
		avrdudePathStr,
		configPathStr,
		deviceListPathStr,
		adminPasswordStr,
	)
}
