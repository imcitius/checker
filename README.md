# Реализация чекалки на Go

Загружает конфиг из файла config.json.

```
{
    "defaults": {
        "timer_step": 5,
        "parameters": {
            "run_every": 60,
            "bot_token": "201865937:AAHBSXrIlEFSbVfUCvkkd3y4kbvJNgNIJuM",
            "project_channel": 1390752,
            "critical_channel": 1390752,
            "min_health": 1,
            "allow_fails": 0,
            "mode": "loud"
        }
    },
    "projects": [
        {
            "name": "icmp_ping",
            "checks": {
                "icmp_ping": [
                    {
                        "host": "8.8.8.8",
                        "timeout": 50,
                        "count": 1
                    },
                    {
                        "host": "1.1.1.1",
                        "timeout": 10,
                        "count": 1
                    }
                ]
            },
            "parameters": {
                "run_every": 60
            }
        },
        {
            "name": "tcp_ping",
            "checks": {
                "tcp_ping": [
                    {
                        "host": "mail.ru",
                        "port": 80,
                        "timeout": 500,
                        "attempts": 3
                    }
                ]
            },
            "parameters": {
                "run_every": 5
            }
        },
        {
            "name": "admitlead",
            "checks": {
                "http": [
                    {
                        "url": "http://ams.admitlead.ru/main/check",
                        "code": 200,
                        "answer": "AdmitLead"
                    },
                    {
                        "url": "http://ks.admitlead.ru/main/check",
                        "code": 200,
                        "answer": "AdmitLead"
                    }
                ]
            },
            "parameters": {
                "run_every": 60,
                "min_health": 1,
                "allow_fails": 2,
                "project_channel": 1390752
            }
        },
        {
            "name": "mgshare",
            "checks": {
                "http": [
                    {
                        "url": "http://mg-ams.mgshare.com/monitor.php",
                        "code": 200,
                        "answer": "OK",
                        "headers": [
                            {
                                "User-Agent": "mediaget"
                            }
                        ]
                    },
                    {
                        "url": "http://mg-bl1.mgshare.com/monitor.php",
                        "code": 200,
                        "answer": "zhopa",
                        "answer_present": "absent",
                        "headers": [
                            {
                                "User-Agent": "mediaget"
                            }
                        ]
                    }
                ]
            },
            "parameters": {
                "run_every": 30,
                "min_health": 2
            }
        }
    ]
}
```


Конфигурация состоит из блоков `defaults`, и `projects`.

В блоке `defaults` в подблоке `parameters` описаны параметры проверок по умолчанию, которые применяются к настройкам проектов, если не были переназначены в блоке `parameters` конкретного проекта.

Отдельный параметр `timer_step` в блоке `defaults` содержит время в секундах, через которое внутренний таймер проверяет наличие проверок которые требуется провести в данный момент.

## В блоке `parameters` содержатся следующие настройки:

### в defaults и в проектах

*run_every*: частота проведения проверки в секундах.

*bot_token*: токен телеграм бота от имени которого отправляются алерты

*critical_channel*: номер канала в телеграм для критических оповещений

*min_health*: минимальное кол-во живых урлов, которое не вводит проект в статус critical

*allow_fails*: кол-во заваленных до статуса critical проверок, которые могут пройти до отсылки алерта в канал critical

*mode*: режим оповещения, в режиме loud алерты отсылаются в телегам, в режиме quiet только выводятся в stdout.


### только в проектах

*project_channel*: номер канала в телеграм для не-критических оповещений

*name*: Имя проекта


## Описание проверок содержится в блоке `checks` проекта.

Поддерживаются проверки трех разных типов:

* HTTP check (блок http)
```
*url*: URL для проверки методом GET
*code*: HTTP код успешного ответа
*answer*: Текст для поиска в HTTP Body ответа
*answer_present*: проверять факт наличия текста (по умолчанию, или "present"), или его отсутствия ("absent")
*headers*: Массив HTTP заголовков для передачи в HTTP запросе, в виде `"User-Agent": "mediaget"`
```

* ICMP Ping Check (блок icmp_ping)

*host*: имя или IP адрес узла для проверки

*timeout*: время ожидания ответа в миллисекундах

*count*: кол-во отправляемых запросов


* TCP Ping check (блок tcp_ping)

*host*: имя или IP адрес узла для проверки

*port*: номер TCP порта

*timeout*: время ожидания ответа в миллисекундах

*attemts*: кол-во попыток открыть порт


Нужно учитывать, что параметр `run_every` должен быть кратен параметру `timer_step`.

Например, если внутренний таймер срабатывает каждые 5 секунд, проверка может быть проведена каждое кол-во секунд кратное 5 (60 секунд, 75 секунд, и т.д.)


## Управление оповещениями

С помощью сообщений боту можно управлять оповещениями и режимом проверки проектов.
Поддерживаются следующие команды:

*/pause* обычным сообщением в чат - полностью отключает все оповещения (аналог quiet в блоке defaults)

*/unpause* обычным сообщением в чат - включает все оповещения (аналог loud в блоке defaults)

эти же команды, отправленные ответом на определенныое сообщени от бота управляют оповещения для конкретного проекта.