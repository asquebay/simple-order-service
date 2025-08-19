# simple-order-service
Сервис обработки заказов на Go, который получает данные о заказах из Kafka, сохраняет их в PostgreSQL, кэширует в памяти для быстрого доступа и отображает через минималистичный веб-интерфейс.

Сервис реализует следующий поток данных:
● продюсер (тестовый скрипт) отправляет JSON-сообщение с данными о заказе в топик Kafka;\
● консьюмер (основное Go-приложение) подписывается на этот топик, читает сообщение;\
● сервис валидирует полученные данные, сохраняет их в базу данных PostgreSQL и помещает в in-memory кэш;\
● HTTP-сервер предоставляет API для получения данных о заказе по его ID. При запросе он сначала проверяет кэш для мгновенного ответа, и только при его отсутствии обращается к PostgreSQL;\
● веб-интерфейс позволяет пользователю ввести ID заказа и получить информацию о нём.

В корне репозитория присутствует файл .env — файл с переменными окружения для настройки подключения к БД. По умолчанию в нём прописаны следующие параметры:
```.env
POSTGRES_USER=simple_order_service_manager
POSTGRES_PASSWORD=secure_password
POSTGRES_DB=orders_db
```
где POSTGRES_USER — пользователь, через которого будет осуществляться подключение к БД,\
POSTGRES_PASSWORD — пароль этого пользователя,\
POSTGRES_DB — собственно, БД.

Эти переменные следует изменить перед началом работы.

## **Установка:**

**1. Клонирование репозитория:**
```
git clone https://github.com/asquebay/simple-order-service.git && cd simple-order-service
```

**2.Экспортируем переменные окружения из файла `.env`:**
```
[user@nixos:~/go/src/simple-order-service]$ export $(grep -v '^#' .env | xargs)
```

**3. Создаём пользователя и базу данных для взамодействия с PostgreSQL:**
```
[user@nixos:~/go/src/simple-order-service]$ sudo -u postgres psql <<-EOSQL
  CREATE USER ${POSTGRES_USER} NOSUPERUSER NOCREATEROLE CREATEDB NOINHERIT REPLICATION NOBYPASSRLS CONNECTION LIMIT -1;
  ALTER USER ${POSTGRES_USER} WITH PASSWORD '${POSTGRES_PASSWORD}';
  CREATE DATABASE ${POSTGRES_DB} OWNER ${POSTGRES_USER} ENCODING 'UTF8';
EOSQL
```
*При успешном выполнении вывод из терминала будет следующим:*
```
CREATE ROLE
ALTER ROLE
CREATE DATABASE
```

**4.1. Установка утилиты для миграций (goose):**
```
[user@nixos:~/go/src/simple-order-service]$ go install github.com/pressly/goose/v3/cmd/goose@v3.24.3
```

**4.2 Проверяем, что goose установился:**
```
[user@nixos:~/go/src/simple-order-service]$ $(go env GOPATH)/bin/goose -h
```
*При успешном выполнении мы увидим вывод справки (help) для goose.*\
*Если же мы увидим, что переменная GOPATH не установлена, то нужно будет её установить (см. https://go.dev/wiki/SettingGOPATH)*:
```
[user@nixos:~/go/src/simple-order-service]$ go env -w GOPATH=$HOME/go
```

**5. Применение миграций — создание всех необходимых таблиц в БД:**
```
[user@nixos:~/go/src/simple-order-service]$ $(go env GOPATH)/bin/goose -dir "migrations" postgres "postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@localhost:5432/${POSTGRES_DB}?sslmode=disable" up
```
*При успешном выполнении вывод из терминала будет следующим:*
```
2025/08/19 18:36:37 OK   20250817202736_create_initial_tables.sql (5.61ms)
2025/08/19 18:36:37 goose: successfully migrated database to version: 20250817202736
```

## **Использование:**

**1. Запуск основного сервиса**\
Откроем два отдельных терминала (например, Konsole) в корне проекта.\
В первом терминале запустим основное приложение. Оно подключится к PostgreSQL и Kafka и начнёт слушать входящие сообщения:
```
[user@nixos:~/go/src/simple-order-service]$ go run ./cmd/app/main.go
```
*При успешном запуске мы увидим логи, подтверждающие старт всех компонентов:*
```
time=2025-08-19T19:34:19.120+03:00 level=INFO source=/home/user/go/src/simple-order-service/cmd/app/main.go:28 msg="starting simple-order-service" log_level=debug
time=2025-08-19T19:34:19.123+03:00 level=INFO source=/home/user/go/src/simple-order-service/cmd/app/main.go:38 msg="successfully connected to postgres"
time=2025-08-19T19:34:19.123+03:00 level=INFO source=/home/user/go/src/simple-order-service/cmd/app/main.go:44 msg="order cache initialized"
time=2025-08-19T19:34:19.123+03:00 level=INFO source=/home/user/go/src/simple-order-service/internal/service/order_service.go:91 msg="starting cache restoration from database" op=service.OrderService.RestoreCache
time=2025-08-19T19:34:19.124+03:00 level=INFO source=/home/user/go/src/simple-order-service/internal/service/order_service.go:101 msg="cache restored successfully" op=service.OrderService.RestoreCache orders_count=0
time=2025-08-19T19:34:19.124+03:00 level=INFO source=/home/user/go/src/simple-order-service/cmd/app/main.go:64 msg="starting http server" port=:8081
time=2025-08-19T19:34:19.124+03:00 level=INFO source=/home/user/go/src/simple-order-service/internal/transport/kafka/consumer.go:48 msg="Kafka consumer started" component=kafka_consumer
```

**2. Отправка тестового заказа в Kafka**\
Во втором терминале запустим тестовый скрипт-продюсер для проверки работоспособности сервера:
```
[user@nixos:~/go/src/simple-order-service]$ go run ./test/kafka/producer.go
```
*При успешном выполнении вывод из терминала будет следующим:*
```
2025/08/19 19:35:09 Sending message to Kafka...
Message sent successfully!
```
*После этого в логах первого терминала появятся записи об успешном получении и обработке заказа.*

**3. Проверка получения заказа:**

**1) Через веб-интерфейс (с помощью браузера, например Firefox)**\
Открываем в браузере URL `http://localhost:8081`. В поле для ввода вставим UID заказа из тестового скрипта (a112bbb3c3c44d5test) и нажмём на кнопку отправки (стрелку) или Enter.
*При успешном выполнении на странице браузера отобразятся полные данные о заказе в формате JSON*

**2) Через терминал (например, через Konsole)**\
```
[user@nixos:~]$ curl http://localhost:8081/order/a112bbb3c3c44d5test
```
*При успешном выполнении вывод из терминала будет следующим:*
```
{"order_uid":"a112bbb3c3c44d5test","track_number":"WBILMTESTTRACK","entry":"WBIL","delivery":{"name":"NixOS User","phone":"+79001234567","zip":"123456","city":"NixOS City","address":"Terminal street 1","region":"Flake","email":"nixos@example.com"},"payment":{"transaction":"a112bbb3c3c44d5test","request_id":"","currency":"RUB","provider":"mir","amount":5000,"payment_dt":1637907727,"bank":"sber","delivery_cost":500,"goods_total":4500,"custom_fee":0},"items":[{"chrt_id":12345,"track_number":"WBILMTESTTRACK","price":4500,"rid":"some_random_id_123","name":"NixOS T-Shirt","sale":0,"size":"M","total_price":4500,"nm_id":67890,"brand":"Nix","status":202}],"locale":"en","internal_signature":"","customer_id":"nixos_user","delivery_service":"cdek","shardkey":"1","sm_id":1,"date_created":"2025-08-15T12:00:00Z","oof_shard":"1"}
```
