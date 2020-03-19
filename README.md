Реализация чекалки на Go

Загружает конфиг из файла config.json.

Конфигурация состоит из блоков `defaults`, и `projects`.

В блоке `defaults` в подблоке `parameters` описаны параметры проверок по умолчанию, которые применяются к настройкам проектов, если не были переназначены в блоке `parameters` конкретного проекта.

Отдельный параметр `timer_step` в блоке `defaults` содержит время в секундах, через которое внутренний таймер проверяет наличие проверок которые требуется провести в данный момент.

В блоке `parameters` содержатся следующие настройки:

*** в defaults и в проектах

*run_every*: частота проведения проверки в секундах.

*bot_token*: токен телеграм бота от имени которого отправляются алерты

*critical_channel*: номер канала в телеграм для критических оповещений

*min_health*: минимальное кол-во живых урлов, которое не вводит проект в статус critical

*allow_fails*: кол-во заваленных до статуса critical проверок, которые могут пройти до отсылки алерта в канал critical

*mode*: режим оповещения, в режиме loud алерты отсылаются в телегам, в режиме quiet только выводятся в stdout.


*** только в проектах

*project_channel*: номер канала в телеграм для не-критических оповещений

*name*: Имя проекта


Описание проверок содержится в блоке `checks` проекта.

Содержит в себе следующие параметры:

*url*: URL для проверки методом GET

*code*: HTTP код успешного ответа

*answer*: Текст для поиска в HTTP Body ответа



Нужно учитывать, что параметр `run_every` должен быть кратен параметру `timer_step`.

Напрмиер если внутренний таймер срабатывает каждые 5 секунд, проверка может быть проведена каждое кол-во секунд кратное 5 (60 секунд, 75 секунд, и т.д.)
