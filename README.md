# lapki-flasher

Модуль загрузчика в составе Lapki IDE. Предоставляет WebSockets-интерфейс для опроса и прошивки совместимых аппаратных платформ.

## Поддерживаемые устройства

- Arduino Uno (ATmega328P)
- Arduino Micro (ATmega32U4)
- Arduino Mega (ATmega2560)

## Зависимости

Поддерживаемые ОС:

- Windows 7 и новее
  - Могут понадобиться драйвера для прошиваемых устройств (например, нестандартных Arduino или в Windows 7).
- Linux-дистрибутивы с менеджером **Systemd**.
  - Для компиляции нужен `libusb`.
  - Для опроса устройств используется `udevadm`. Возможна работа с `eudev`, но это не тестировалось.

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
