# Реализация чекалки на Go

Хранение конфигурации реализовано с помощью библиотеки `github.com/spf13/viper`.По умолчанию из файла config.yaml в текущем каталоге.

Управление командами CLI и флагами на базе `github.com/spf13/cobra`.

```
# ./checker                           
^_^

Usage:
  checker [command]

Available Commands:
  check       Run scheduler and execute checks
  help        Help about any command
  testcfg     unmarshal config file into config structure
  version     Print the version number of Hugo

Flags:
      --config string       config file (default is ./config.yaml) (default "config")
      --configformat string   config file format: (default is yaml)
      --configsource string   config file source: file or consul (default is file)
      --configwatchtimeout string   config watch period (default '5s') (default "5s")
  -D, --debugLevel string   Debug level: Debug,Info,Warn,Error,Fatal,Panic (default "info")
  -h, --help                help for checker
      --viper               use Viper for configuration (default true)

Use "checker [command] --help" for more information about a command.

```

Хранение конфигурации доступно в файловой системе (по умолчанию), или в KV Consul.
Ключ `--configsource=consul` позволяет переключить получение конфига на Consul. При этом считываются две ENV переменные CONSUL_ADDR и CONSUL_PATH. Из первой берется URL сервера Consul, из второй - путь к ключу KV с конфигом.
KV ключ должен содержать полную конфигурацию, обязательно в форматах `yaml` или `json`, загрузка из древовидной KV структуры не поддерживается.
Каждый период, заданный ключом `--configwatchtimeout=5s` Checker пытается перечитать конфиг из хранилища. Если конфиг загружен успешно, проверяется его валидность и соответствие текущей конфигурации.
Если конфиг валиден и отличается от текущей конфигурации, он подменяет текущую конфигурацию, и происходит перезапуск скедулера и ботов.
Конфиг, загруженный из файловой системы, также автоматически мониторится на обновления.





Конфигурация состоит из блоков `defaults`, `alerts` и `projects`.

Секретные параметры (пароли, токены) могут быть сохранены в Hashicorp Vault, в данный момент поддерживается загрузка токенов для телеграм ботов и паролей для SQL баз данных.
Формат: `vault:secret/path/to/token:field`. Значение поля field из пути secret/path/to/token будет использовано в качестве токена/пароля.
Извлеченные из Vault секреты кешируются на 5 минут, для снижения нагрузки на Vault.

В блоке `defaults` в подблоке `parameters` описаны параметры проверок по умолчанию, которые применяются к настройкам проектов, если не были переназначены в блоке `parameters` конкретного проекта.

Отдельные параметры `timer_step` и `http_port` в блоке `defaults` содержат время, через которое внутренний таймер проверяет наличие проверок которые требуется провести в данный момент, и порт для HTTP сервера по-умолчанию.

## В блоке `parameters` содержатся следующие настройки:

### в defaults и в проектах
```
run_every: частота проведения проверки и отработки алертов.

bot_token: токен телеграм бота от имени которого отправляются алерты

critical_channel: номер канала в телеграм для критических оповещений

min_health: минимальное кол-во живых урлов, которое не вводит проект в статус critical

allow_fails: кол-во заваленных до статуса critical проверок, которые могут пройти до отсылки алерта в канал critical

mode: режим оповещения, в режиме loud алерты отсылаются в телегам, в режиме quiet только выводятся в stdout.

noncrit_alert: имя метода оповещения для некритичных алертов

crit_alert: имя метода оповещения для критичных алертов

command_channel: имя метода оповещения для приема команда в бот (берется параметр noncritical_channel)

SSLExpirationPeriod: проверка близости времени истечения SSL сертификатов при http проверках

periodic_report_time: период отправки отчетов по отключенным проверкам в каналы 

mentions: кого нотифицировать в алертах по данному проекту. бывает удобно всем участникам чата держать его замьюченным, а уведомлять по конкретным проблемам конкретные персоны.

```

Нужно учитывать, что параметр `run_every` должен быть кратен параметру `timer_step`.

Например, если внутренний таймер срабатывает каждые 5 секунд, проверка может быть проведена каждое кол-во секунд кратное 5 (60 секунд, 75 секунд, и т.д.)


## Описание методов оповещения содержится в блоке `alerts`.

Блок должен содержать подблоки, с настройками специфичными для каждого метода оповещения:

```
name: Имя метода оповещения
type: Тип метода оповещения (пока поддерживается только telegram = tg_alert)
bot_token: токен для telegram бота 
noncritical_channel: Канал для некритичных оповещений
critical_channel: Канал для критичных оповещений
```

## Описание проверок содержится в блоке `healthchecks` проекта.

Блок `healthchecks` должен содержать блоки с описанием наборов проверок и опционально блок `parameters`.
Данные настройки перекрывают настройки уровня проекта и корневого уровня.
Каждый набор проверок имеет имя в поле `name` и описание проверок в блоке `checks`.

Поддерживаются проверки разных типов (обязательные параметры помечены *).
- http
- icmp
- tcp
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

Алгоритм действий следующий: в таблицу на мастере вставляется запись со случайными значениеями id и test_value.
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
       test_value int
    )
```

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



## Управление оповещениями

С помощью сообщений боту можно управлять оповещениями и режимом проверки проектов.
Ключом командной строки 
Поддерживаются следующие команды:

*/pa* обычным сообщением в чат - полностью отключает все оповещения (аналог quiet в блоке defaults)

*/ua* обычным сообщением в чат - включает все оповещения (аналог loud в блоке defaults)

Команды управления алертами для указанного элемента.
Команды */pp,/up <project_name>* и */pu,/uu <UUID>* управляют алертами для проектов и конкретных проверок.
Они могут быть отправлены обычным сообщением в чат, либо ответом на конкретный алерт.

В случае ответа на алерт, имя проекта или UUID проверки извлекается из этого алерта.

## Метрики

Метрики в формате prometheus публикуются на эндпоинте /metrics.
Метрики `sched_*` отражают работу внутреннего цикла скедулера.
Метрики `alerts_by_event_type` - статистика по алертам в разрезе различных событий.
Метрики `events_by_*` - статистика по событиям в разрезе различных проектов и проверок.
Метрики `check_duration` - статистика по времени выполнения проверок.
