# lapki-flasher

Модуль загрузчика в составе Lapki IDE. Предоставляет WebSockets-интерфейс для опроса и прошивки совместимых аппаратных платформ.

## Поддерживаемые устройства

- Arduino Uno (ATmega328P)
- Arduino Micro (ATmega32U4)
- Arduino Mega (ATmega2560)

## Добавление нового устройства

Для того, чтобы добавить шаблон (описание) нового типа устройств, нужно открыть файл [device_list.JSON](https://github.com/Roundabout1/lapki-flasher/blob/main/src/device_list.JSON) и дополнить его описанием нового устройства.

Описание должно содержать следующие поля:
- ID: уникальный идентификатор шаблона (это именно идентификатор описания, а не самого устройства, он назначается самим разработчиком)
- name: имя устройство (можно написать что угодно, оно не обязтельно должно совпадать с реальным названием устройства)
- productIDs: возможные значения productID, требуется для обнаружения устройства (можно найти через базу данных: https://devicehunt.com/)
- vendorIDs: возможные значения vendorID, требуется для обнаружения устройства (можно найти через базу данных: https://devicehunt.com/)
- controller: контроллер устройства, требуется для avrdude
- programmer: программатор устройства, требуется для avrdude
- bootloaderID: уникальный идентификатор шаблона bootloader (см. раздел "Добавление bootloader") версии устройства, если отсутствует, то значение должно быть равным -1. 

### Добавление bootloader
Если устройство прошивается через bootloader (как Arduino Micro), то это значит, что оно состоит из двух устройств, каждому из которых необходимо предоставить своё описание, при этом основное устройство должно ссылаться на ID bootloader, а сам bootloader, не должен ссылаться на что-либо (см. описания Arduino Micro и Arduino Micro (bootloader) в файле со списком устройств).

## Зависимости

Поддерживаемые ОС:

- Windows 7 и новее
  - Могут понадобиться драйвера для прошиваемых устройств (например, нестандартных Arduino или в Windows 7).
- Linux-дистрибутивы с менеджером **Systemd**.
  - Для компиляции нужен `libusb`.
  - Для опроса устройств используется `udevadm`. Возможна работа с `eudev`, но это не тестировалось.
- macOS (тестировалось на macOS 13 Ventura).
  - Для опроса устройств используется `ioreg`. 

Также для прошивки потребуется **avrdude**. В Linux достаточно установить утилиту встроенным пакетным менеджером, под Windows предлагается установить [форк от maurisgreuel](https://github.com/mariusgreuel/avrdude), положить в рабочую директорию или PATH.

## Сборка и разработка

Для сборки требуется компилятор Go версии 1.20 и новее. Перейдите в директорию `src` и выполните:

`go build -mod=mod .`

Репозиторий содержит описание пакета для **NixOS**. Для сборки этого пакета выполните `nix-build`, для входа в окружение разработки – `nix-shell`.

## Настраиваемые параметры

Некоторые параметры загрузчика можно настроить перед запуском программы, указав их в коммандной строке при запуске модуля:

- `-address` (string): адресс для подключения (по-умолчанию "localhost:8080").
- `-fileSize` (int): максимальный размер файла, загружаемого на сервер (в байтах) (по-умолчанию 2097152).
- `-listCooldown` (int): минимальное время (в секундах), через которое клиент может снова запросить список устройств, игнорируется, если количество клиентов меньше чем 2 (по-умолчанию 2 секунды).
- `-msgSize` (int): максмальный размер одного сообщения, передаваемого через веб-сокеты (в байтах) (по-умолчанию 1024).
- `-thread` (int): максимальное количество потоков (горутин) на обработку запросов на одного клиента (по-умолчанию 3).
- `-updateList` (int): количество секунд между автоматическими обновлениями, не может быть меньше единицы, если получено значение меньше единицы, то оно заменяется на 1 (по-умолчанию 15)
- `-verbose`: если указан, то программа будет выводить подробное описание того, что она делает.
- `-help`: вывести описание настраиваемых параметров.
- `-alwaysUpdate`: всегда искать устройства и обновлять их список, даже когда ни один клиент не подключён (используется для тестирования)
- `-stub`: количество ненастоящих, симулируемых устройств, которые будут восприниматься как настоящие, применяется для тестирования, при значении 0 или меньше фальшивые устройства не добавляются (по-умолчанию 0)
- `-avrdudePath`: путь к avrdude (по-умолчанию avrdude, то есть будет использоваться системный путь)
- `-configPath`: путь к файлу конфигурации avrdude (по-умолчанию '', то есть пустая строка)
- `-deviceListPath`: путь к JSON-файлу со списком устройств. Если прописан, то заменяет стандартный список устройств, при условии, что не возникнет ошибок, связанных с чтением и открытием JSON-файла, иначе используется стандартный список устройств (по-умолчанию пустая строка, означающая, что будет используется, встроенный в загрузчик список)

Пример: `./lapki-flasher.exe -address localhost:3939 -verbose -updateList 10`.

## Запуск в качестве сервера

Для размещения загрузчика как сетевого сервиса рекомендуется использовать сервер под управлением Linux-дистрибутива с менеджером **Systemd**.

На текущий момент репозиторий не предоставляет инструментов для развёртывания, но эта задача решается тривиально.

Самая важная деталь – по умолчанию загрузчик запускается по адресу `localhost:8080`, что делает его недоступным извне, так как основная задача модуля – работа в фоне Lapki IDE. Поэтому при запуске модуля необходимо указать ключ `--address :8080`, и **открыть соответствующий порт** в файрволле.

Пример Systemd-сервиса для запуска загрузчика (путь к загрузчику может быть иной):

```toml
[Unit]
Description=Starts lapki-flasher
After=network-online.target

[Service]
ExecStart=/bin/lapki-flasher --address :8080
User=root
Type=simple
Restart=always
RestartSec=3
TimeoutStopSec=5
```

Альтернативно, можно разместить загрузчик за HTTP-сервером по типу **nginx**, но при этом следует настроить [проксирование WebSocket-канала](http://nginx.org/en/docs/http/websocket.html).

## Протокол для общения с клиентом
Клиент и сервер обмениваются сообщениями через веб-сокеты.

### Общий вид сообщений
Все сообщения от сервера кодируется через JSON, для обратной коммуникации клиент тоже должен кодировать свои сообщения через JSON, за исключением бинарных данных (см. таблицу "Взаимодействие с загрузчиком"). 
```golang
// Общий вид сообщения на сервер (язык - Golang)
type Event struct {
	// Тип сообщения (flash-start, get-list и т.д.)
	Type string `json:"type"`
	// Параметры сообщения, не все сообщения обязаны иметь параметры
	Payload json.RawMessage `json:"payload"`
}
```
```typescript
// аналогичная струтура для TS
class Event {
    constructor(type, payload) {
        this.type = type;
        this.payload = payload;
    }
}
```

Ниже представлены таблицы с сообщениями, все параметры имеют тип string, если не указано обратное в скобках. Столбец источник указывает на того, кто может отправить данный тип сообщения (клиент или сервер).
### Взаимодействие со списком устройств 
| Сообщение            | Параметры                                                  | Описание                                                                                                           | Источник |
|----------------------|------------------------------------------------------------|--------------------------------------------------------------------------------------------------------------------|----------|
| get-list             |                                                            | Сервер начнёт отправлять клиенту сообщения типа device клиенту до тех пор пока не отправит описание всех устройств | Клиент   |
| device               | deviceID, name, controller, programmer, portName, serialID | Отправляет описание устройства клиенту                                                                             | Сервер   |
| device-update-delete | deviceID                                                   | Подтверждает удаление устройства из списка                                                                         | Сервер   |
| device-update-port   | deviceID, portName                                         | Обновление имени порта к которому подключено устройство                                                            | Сервер   |
| empty-list           |                                                            | ответ на 'get-list', если устройства не найдены                                                                    | Сервер   |
### Взаимодействие с загрузчиком
| Сообщение               | Параметры                               | Описание                                                                                                                                                                                                                                                                                                                                   | Источник |
|-------------------------|-----------------------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|----------|
| flash-start             | deviceID, fileSize (размер файла (int)) | Запрос на начало прошивки. Если прошивку начать нельзя, то клиенту отправляется причина. Иначе начинается процесс загрузки файла. Если файл слишком большой, то его надо отправлять блоками. В этом случае сервер начнёт посылать сообщения типа "flash-next-block", после получения которых клиент должен начать отправку бинарных данных. | Клиент   |
| (бинарные данные файла) |                                         | Файл прошивки в бинарном виде, команда не имеет названия. Предпологается, что клиент начнёт передавать бинарные файлы серверу после получения сообщения "flash-next-block".                                                                                                                                                                | Клиент   |
| flash-next-block        |                                         | Запрос на следующий блок бинарных данных, клиент должен отправлить блок с данными только после получения этого сообщения                                                                                                                                                                                                                   | Сервер   |
| flash-done              | avrmsg (сообщение от avrdude)           | файл успешно прошит в выбранное устройство                                                                                                                                                                                                                                                                                                 | Сервер   |
| get-max-file-size       |                                         | получить максимальный размер файла для загрузки на сервер                                                                                                                                                                                                                                                                                  | Клиент   |
| max-file-size           | size (максимальный размер файла (int))  | максимальный размер файла для загрузки на сервер                                                                                                                                                                                                                                                                                           | Сервер   |
### Сообщения об ошибках от сервера
| Сообщение                  | Параметры | Описание                                                                                      |
|----------------------------|-----------|-----------------------------------------------------------------------------------------------|
| flash-wrong-id             |           | устройство с таким ID отсутствует в списке                                                    |
| flash-disconnected         |           | устройство есть в списке, но оно не подключено к серверу                                      |
| flash-avrdude-error        | avrmsg    | avrdude не смог прошить устройство, 'avrmsg' - сообщение об ошибке от avrdude                 |
| flash-not-finished         |           | предыдущая операция прошивки ещё не завершена                                                 |
| flash-not-started          |           | получены бинарные данных, хотя запроса на прошивку не было                                    |
| flash-blocked              |           | устройство заблокировано другим пользователем для прошивки                                    |
| flash-large-file           |           | указанный размер файла превышает максимально допустимый размер файла, установленный сервером. |
| event-not-supported        |           | сервер получил от клиента неизвесный тип сообщения                                            |
| unmarshal-err              |           | не удалось распарсить JSON-сообщение от клиента                                               |
| get-list-cooldown          |           | запрос на 'get-list' отклонён так как, клиент недавно уже получил новый список                |
| flash-not-supported        | name      | плата с именем 'name' не поддерживается для прошивки                                          |
| flash-open-serial-monitor  |           | нельзя начать прошивку, пока открыт монитор порта этого устройства   

### Serial monitor
| Сообщение                | Параметры                     | Описание                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  |        |
|--------------------------|-------------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|--------|
| serial-connect           | deviceID, baud                | Подключиться к устройству deviceID со скростью baud.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      | клиент |
| serial-connection-status | deviceID, code (int), comment | code = 0: установлено подключение к устройству deviceID; <br>code = 1: не удалось открыть монитор порта;  <br>code = 2: устройство с deviceID отсутствует в списке устройств (сервер самостоятельно отправит сообщение device-update-delete);  <br>code = 3: попытка открыть монитор порта для фальшивого устройства;<br>code = 4: не удалось распарсить JSON-сообщение;   <br>code = 5: устройство занято прошивкой;  <br>code = 6: монитор порта уже открыт;   <br>code = 7: закрытие порта, ошибка чтения;  <br>code = 8: закрытие монитора порта по запросу клиента;<br>code = 9: не удалось открыть порт на новой скорости; соединение прервано;<br>code = 10: порт заново открыт на скорости <comment><br>code = 11: не удалось изменить бод, из-за ошибки парсинга JSON-сообщения (соединение не прерывается).<br>code = 12: бод нельзя изменить, так как монитор порта не открыт (соединение не прерывается).<br>code = 13: бод нельзя изменить, так как монитор открыт другим клиентом (соединение не прерывается).<br>code = 14: этот монитор порта нельзя закрыть, так как он занят другим клиентом.<br>code = 15: новая скорость совпадает со старой скоростью<br><br><br>comment - дополнительное сообщение, содержит текст ошибки. Этот тип сообщения автоматически отправляется сервером при изменении статуса соединения. | сервер |
| serial-disconnect        | deviceID                      | Прервать соединение с устройством deviceID.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               | клиент |
| serial-send              | deviceID, msg                 | Отправить сообщение msg на deviceID.  Желательно не отправлять новых сообщений на deviceID, пока не получен serialSentStatus.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                             | клиент |
| serial-sent-status       | deviceID, code (int), comment | Сведения о последнем отправленном на deviceID сообщении:   <br>code = 0: сообщение было отправлено успешно;   <br>code = 1: сообщение не удалось отправить;  <br>code = 2: устройство с deviceID отсутствует в списке устройств (сервер самостоятельно отправит сообщение device-update-delete);  <br>code = 3: не удалось отправить сообщение, так как монитор порта закрыт;  <br>code = 4: не удалось распарсить JSON-сообщение;  <br>code = 5: попытка отправить сообщение на устройство другого клиента.<br><br><br>comment - дополнительное сообщение, содержит текст ошибки.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        | сервер |
| serial-device-read       | deviceID, msg                 | Сообщение от устройства deviceID.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                         | сервер |
| serial-change-baud       | deviceID, baud                | Сменить бод. Сервер отправит serial-connection-status в ответ.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                            |        |

### МС-ТЮК
| Сообщение       | Параметры                             | Описание                                                                                                                                                                                                                                                                                                                                                  |        |
|-----------------|---------------------------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|--------|
| ms-device       | deviceID, name, portNames ([4]string) | устройство МС-ТЮК; сожержит массив из 4 портов, первый порт для загрузки, последний для монитора порта                                                                                                                                                                                                                                                    |        |
| ms-ping         | deviceID, address                     | Отправить пинг по заданному адресу на МС-ТЮК                                                                                                                                                                                                                                                                                                              | клиент |
| ms-ping-result  | deviceID, code (int), comment         | результат пинга<br>code 0: пришёл обратный ответ (понг)<br>code 1: устройство не найдено <br>code 2: ошибка пингования<br>code 3: неправильный тип устройства (тип устройства не может выполнить эту операцию)<br>code 4: не удалось распарсить JSON-сообщение;                                                                                           | сервер |
| ms-get-address  | deviceID                              | запрос на получения адреса                                                                                                                                                                                                                                                                                                                                | клиент |
| ms-address      | deviceID, code (int), comment         | получение адреса МС-ТЮК клиентом<br><br>code 0: получен адрес, в comment содержится адрес<br>code 1: устройство не найден<br>code 2: получена ошибка при попытке узнать адрес, в comment содержится текст ошибки<br>code 3: неправильный тип устройства (тип устройства не может выполнить эту операцию)<br>code 4: не удалось распарсить JSON-сообщение; | сервер |
| ms-bin-start    | deviceID, fileSize, address           | Запрос на начало загрузки прошивки на МС-ТЮК по заданному адресу; Команда аналогична flash-start, то есть протокол загрузки прошивки такой же, клиент начрнёт получать такие же команды, как если бы он отправил flash-start. Сервер так же ожидает аналогичные команды от клиента.                                                                       | клиент |
| ms-reset        | deviceID, address                     | Запрос на сброс устройства                                                                                                                                                                                                                                                                                                                                |        |
| ms-reset-result | deviceID, code (int), comment         | Результат ms-reset<br>code 0: сброс произошёл успешно<br>code 1: устройство не найдено<br>code 2: ошибка при сбросе устройства, comment может содержать текст ошибки<br>code 3: неправильный тип устройства (тип устройства не может выполнить эту операцию)<br>code 4: не удалось распарсить JSON-сообщение;                                             |        |