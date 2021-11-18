![Go](https://github.com/imcitius/checker/workflows/Go/badge.svg)

# Реализация чекалки на Go

Утилита задумана в качестве универсального демона, способного проводить периодические проверки (хелсчеки) различных IT систем, 
отсылать алерты и выполнять какие-то действия в случае смены статуса проверки.

Запустить Чекалку в облаке можно бесплатно, например с помощью fly.io Free tier:
https://fly.io/docs/speedrun/

Хранение конфигурации реализовано с помощью библиотеки [Koanf](https://github.com/knadh/koanf).
По умолчанию загружается конфиг из файла config.yaml в текущем каталоге.

```
$ ./checker

Start dev 
^_^

Usage:
  checker [command]

Available Commands:
  check       run scheduler and execute checks
  completion  generate the autocompletion script for the specified shell
  gentoken    generate auth token
  help        Help about any command
  list        list config elements
  singlecheck execute single check by UUID
  testcfg     unmarshal config file into config structure
  version     Print the version number of Checker

Flags:
  -b, --botsEnabled                 Whether to enable active bot (default true)
  -u, --checkUUID string            UUID to check with SingleCheck
  -c, --config string               config file
  -f, --configformat string         config file format (default "yaml")
  -s, --configsource string         config file source: file, consul, s3
  -w, --configwatchtimeout string   config watch period (default "5s")
  -D, --debugLevel string           Debug level: Debug,Info,Warn,Error,Fatal,Panic (default "warn")
  -h, --help                        help for checker
  -l, --logformat string            log format: text/json (default "text")
  -W, --watchConfig                 Whether to watch config file changes on disk (default true)

Use "checker [command] --help" for more information about a command.
```

Хранение конфигурации доступно во всех хранилищах, поддерживаемых библиотекой [Koanf](https://github.com/knadh/koanf).
Можно загружать любые настройки из переменных окружения CHECKER_* (см. в документации на Koanf).

Ключ `-s` позволяет переключить получение конфига на Consul или S3.
Для S3 настройки берутся из переменных: 

AWS_ACCESS_KEY_ID - ID ключа
AWS_SECRET_ACCESS_KEY - секретный ключ
AWS_REGION - регион
AWS_BUCKET - имя бакета
AWS_OBJECT_KEY - путь до объекта от корня бакета

Для Consul считываются две ENV переменные: CONSUL_ADDR и CONSUL_PATH. Из первой берется URL сервера Consul, из второй - путь к ключу KV с конфигом.
KV ключ должен содержать полную конфигурацию, в форматах `yaml`, `json`, `toml`, `hcl` (задается ключом -f), загрузка из древовидной KV структуры не поддерживается.

Каждый период, заданный ключом `--configwatchtimeout` (по-умолчанию `5s`) Checker пытается перечитать конфиг из хранилища. Если конфиг загружен успешно, проверяется его валидность и соответствие текущей конфигурации.
Если конфиг валиден и отличается от текущей конфигурации, он подменяет текущую конфигурацию, и происходит перезапуск скедулера и ботов.
Конфиг, загруженный из файловой системы, также автоматически мониторится на обновления.


Для некоторых web эндпоинтов требуется JWT авторизация. JWT токен генерируется с помощью CLI команды gentoken.
Токен генерируется с участием ключа, указанного в конфигурации в параметре `defaults.token_encryption_key`, либо в ENV переменной .
ENV переменная имеет бОльший приоритет.
Поддерживается загрузка токена из Vault. 

Конфигурация загружается в соответствии с шаблоном:
```json
  {
    "defaults":{},
    "actors":{},
    "alerts":{},
    "projects":{},
    "consul_catalog":{}
  }
```

Секретные параметры (пароли, токены) могут быть сохранены в Hashicorp Vault, в данный момент поддерживается загрузка секретов для телеграм ботов, JWT авторизации, паролей для SQL баз данных и http проверок.
Формат: `vault:secret/path/to/token:field`. Значение поля `field` из пути `secret/path/to/token` будет использовано в качестве секрета.
Извлеченные из Vault секреты кешируются на 5 минут, для снижения нагрузки на Vault.

В блоке `defaults` в подблоке `parameters` описаны параметры проверок по умолчанию, которые применяются к настройкам проектов, если не были переназначены в блоке `parameters` конкретного проекта.

Отдельный параметр `http_port` в блоке `defaults` содержит порт для HTTP сервера по-умолчанию.
В случае если задана переменная окружения PORT, используется номер порта из нее.

## В блоке `parameters` содержатся следующие настройки:

### в defaults и в проектах
```
check_period: частота проведения проверки и отработки алертов.
// TODO проверить работу фичи
report_period: частота отправки отчетов по отключенным проверкам в каналы 

// TODO проверить работу фичи
min_health: минимальное кол-во живых проверок в рамках healthchck, которое не вводит проект в статус critical

allow_fails: кол-во заваленных до статуса critical проверок, которые могут пройти до отсылки алерта в канал critical

mode: режим оповещения, в режиме loud алерты отсылаются в телегам, в режиме quiet только выводятся в stdout.

noncrit_alert: имя метода оповещения для некритичных алертов

crit_alert: имя метода оповещения для критичных алертов

command_channel: имя метода оповещения для приема команда в бот (по-умолчанию берется параметр noncritical_channel)

// TODO добавить проверку сертификатов для всех tls, а не только для https
ssl_expiration_period: проверка близости времени истечения SSL сертификатов при http проверках

mentions: кого нотифицировать в алертах по данному проекту. бывает удобно всем участникам чата держать его замьюченным, а уведомлять по конкретным проблемам конкретные персоны.

bots_enabled: запускать ли бота
```

## Под "актором" понимается действие, которое должно быть выполнено при смене статуса проверки (actor_up'actor_down).
Описание акторов (действий) содержится в блоке `actors`.

// TODO


## Описание методов оповещения содержится в блоке `alerts`.

Поддерживается три типа оповещений: telegram, slack/mattermost, log.

Блок должен содержать подблоки, с настройками специфичными для каждого метода оповещения:

```
// общие параметры
name: Имя метода оповещения
type: Тип метода оповещения (log, telegram, slack или mattermost)

// параметры телеграм
bot_token: токен
noncritical_channel: Канал для некритичных оповещений
critical_channel: Канал для критичных оповещений

// параметры slack/mattermost
mattermost_webhook_url: url webhook-а. Используется для всех типов оповещений и ChatOps.

```

Если блок alerts отсутствует, все оповещения будут отправляться только в лог.

В настройках каждой проверки может быть выставлен параметр `severity: critical`, чтобы принудительно 
отпрвлять алерты в канал критических оповещений. 

## Управление оповещениями

С помощью сообщений боту можно управлять оповещениями и режимом проверки проектов.
Ключом командной строки 
Поддерживаются следующие команды:

*/qa* обычным сообщением в чат - полностью отключает все оповещения (аналог quiet в блоке defaults)

*/la* обычным сообщением в чат - включает все оповещения (аналог loud в блоке defaults)

Команды управления алертами для указанного элемента.
Команды */qp,/lp <project_name>* и */qu,/lu <UUID>* управляют алертами для проектов и конкретных проверок.
Они могут быть отправлены обычным сообщением в чат, либо ответом на конкретный алерт.

В случае ответа на алерт, имя проекта или UUID проверки извлекается из этого алерта.


## Описание проверок содержится в блоке `healthchecks` проекта.

Блок `healthchecks` должен содержать блоки с описанием наборов проверок и опционально блок `parameters`.
Данные настройки перекрывают настройки уровня проекта и корневого уровня.
Каждый набор проверок имеет имя в поле `name` и описание проверок в блоке `checks`.

Поддерживаются проверки разных типов (обязательные параметры помечены *).
- http
- icmp
- tcp
- getfile
- mysql_query
- mysql_query_unixtime
- mysql_replication
- pgsql_query
- pgsql_query_unixtime
- pgsql_replication
- redis_pubsub
- clickhouse_query
- clickhouse_query_unixtime

### HTTP check
```
*type: "http"
*url: URL для проверки методом GET
code: набор возможных HTTP кодов успешного ответа (слайс int, например `[200,420]` по умолчанию только 200)
answer: Текст для поиска в HTTP Body ответа
answer_present: проверять факт наличия текста (по умолчанию, или "present"), либо его отсутствия ("absent")
headers: Массив HTTP заголовков для передачи в HTTP запросе
    {
        "User-Agent": "custom_user_aget"
    }

timeout: время ожидания ответа
auth: блок содержащий учетные данные, если требуется http basic аутентикация.
    "auth": {
        "user": "username",
        "password": "S3cr3t!"
    }
skip_check_ssl: не проверять валидность серверного SSL сертификата
stop_follow_redirects: не следовать HTTP редиректам
cookies: массив объектов http.Cookie (можно передавать любые параметры из https://golang.org/src/net/http/cookie.go
    "cookies": [
        {
          "name": "test_cookie",
          "value": "12345"
        }
    ]
```


### ICMP Ping Check
```
*type: "icmp"
*host: имя или IP адрес узла для проверки
*timeout: время ожидания ответа (сравнивается со средним RTT за все попытки)
*count: кол-во отправляемых запросов
```

### TCP Ping check (проверяет что порт открыт и отвечает за нужное время)
```
*type: "tcp"
*host: имя или IP адрес узла для проверки
*port: номер TCP порта
*timeout: время ожидания ответа
attempts: кол-во попыток открыть порт (по умолчанию 3)
```

### GetFile check (скачивает файл и проверяет его md5 хэш).

Каждый файл скачивается в локальную файловую систему, где запущен процесс во врененный файл, после проверки удаляется.
Нужно учитывать возможные ограничения по объему файловой системы.
```
*type: "getfile"
*host: url откуда скачать файл
*hash: хеш для сравнения
```

### Проверка выполнения запросов к базам данных
```
*type: тип проверки - mysql_query, pgsql_query, clickhouse_query
*host: адрес сервера БД
port: порт для подключения (если опущено, используются порты по-умолчанию)
timeout: таймаут подключения и выполнения запроса (отдельно проверяется время подключения, и время запроса)
*sql_query_config: содержит параметры запроса
**dbname: имя базы
**username: имя пользоваля
**password: пароль
query: запрос для выполнения. если опущено, выполняется `select 1`, и ответ не проверяется
response: ответ, с которым сверяется вернувшееся из базы значение. 
в ответе ожидается _одно_ поле. Если опущено, то проверяется только сам факт успешного запроса.

    {
      "type": "mysql_query",
      "host": "192.168.132.101",
      "port": 3306,
      "timeout": 1s,
      "sql_query_config": {
        "username": "username",
        "dbname": "dbname",
        "password": "vault:secret/cluster/userA/pass:value",
        "query": "select regdate from users order by id asc limit 1;",
        "response": "1278938100"
      }
    }

```

### Проверка возраста записи в базе данных
```
*type: тип проверки - clickhouse_query_unixtime, mysql_query_unixtime, pgsql_query_unixtime
*host: адрес сервера БД
port: порт для подключения (если опущено, используются порты по-умолчанию)
timeout: таймаут подключения и выполнения запроса
*sql_query_config: содержит параметры запроса
**dbname: имя базы
**username: имя пользоваля
**password: пароль
query: запрос для выполнения. если опущено, выполняется `select 1`, и ответ не проверяется
difference: максимальная разность с текущим временем. если опущено, проверка не производится
в ответе ожидается _одно_ поле, содержащее целое число в формате UnixTime.

    {
      "type": "clickhouse_query_unixtime",
      "host": "192.168.126.50",
      "port": 9000,
      "sql_query_config": {
        "username": "username",
        "dbname": "dbname",
        "password": "she1Haiphae5",
        "query": "select max(serverTime) from forex.quotes1sec",
        "difference": "15m"
      },
      "timeout": "5s"
    },

```

### Проверка репликации баз данных
```
Настройка аналогно проверке запросом, вместо параметров query/response параметры tablename и serverlist.
В tablename передается имя таблицы для вставки тестовой записи (по-умолчанию "repl_test"). В блоке serverlist - список серверов для проверки.
В список лучше всего включить все сервера кластера (в т.ч. и мастер) для более полноценного контроля.

Алгоритм действий: в таблицу на мастере вставляется запись со случайными значениями id и test_value.
Значения выбираются в диапазоне 1-5 для id и 1-9999 для test_value.
Если вставка была успешной, то производится чтение из серверов в списке serverlist поля с соответствующим id.
Если репликация работает, то результат на каждом сервере должен соответствовать test_value. 

*type: тип проверки - mysql_replication, pgsql_replication

Пример конфигурации:
    {
      "type": "pgsql_replication",
      "host": "master.pgsql.service.staging.consul",
      "port": 5432,
        "sql_repl_config": {
        "username": "username",
        "dbname": "dbname",
        "password": "ieb6aj2Queet",
        "tablename": "repl_test",
        "serverlist": [
          "pgsql-main-0.node.staging.consul",
          "pgsql-main-1.node.staging.consul",
          "pgsql-main-2.node.staging.consul"
        ]
      }
    }

Таблица для проверки должна соответствовать схеме:
    CREATE TABLE repl_test (
       id int primary key,
       test_value int,
       timestamp timestamp NOT NULL DEFAULT NOW()
    )
```
В PostgreSQL версий 10+ нужно дать права пользователю для получения данных о репликации без роли суперпользователя:
```GRANT pg_monitor TO checker;```


### Проверка Pub/Sub
```
*type: тип проверки - redis_pubsub
*host: адрес сервера
port: порт для подключения (если опущено, используются порты по-умолчанию)
timeout: таймаут подключения и выполнения запроса
*pubsub_config: содержит параметры запроса
*channel: имя канала для подписки
password: пароль

После успешной подписки в канале ожидается одно любое сообщение (типа отличного от Subscription/Pong) с данными.
При расчете таймаут надо учитывать:

1) время подключения к серверу. 2) время выполнения подписки и ожидания подтверждения в сообщении Subscription, время получения сообщения с данными.

    {
      "type": "redis_pubsub",
      "host": "master.redis.service.staging.consul",
      "pubsub_config": {
        "channels": [
          "ticks_EURUSD",
          "ticks_USBRUB"
        ]
      },
      "timeout": 5s
    }
```

### Пассивные проверки
```
В случае, если активная проверка по каким-то причинам нежелательна или невозможна, пассивная проверка позволит отслееживать статус проверки.

    {
      "name": "passive check of service A",
      "type": "passive",
      "timeout": 5m
    }

Отстуки принимаются GET запросом на эндпоинт http://checker/check/ping/<check uuid>.
Список всех UUID можно получить GET запросом на эндпоинт http://checker/listChecks, либо CLI командой list. 
Для генерации через WEB, требуется JWT авторизация, токен генерируется CLI командой gentoken (см).

  curl -H "Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJqdGkiOiJPaTNvb3hpZTRhaWtlaW1vb3pvOEVnYWk2YWl6OXBvaCIsImF1ZCI6ImFkbWluIn0.wjCl69SvEbHFiMSK-iRXOIvcd5wkO-MCF0oQsNrqVL8" http://checker/listChecks

```

## импорт сервисов из Consul
// TODO описать
consul_catalog

## Метрики

Метрики в формате prometheus публикуются на эндпоинте /metrics.

Метрики `sched_*` отражают работу внутреннего цикла скедулера.

Метрики `alerts_by_event_type` - статистика по алертам в разрезе различных событий.

Метрики `events_by_*` - статистика по событиям в разрезе различных проектов и проверок.

Метрики `check_duration` - статистика по времени выполнения проверок.


## Web API

`/check/ping/<check-uuid>` - обновить статус пассивной проверки

`/check/status/<check-uuid>` - запрос статуса проверки