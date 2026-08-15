# SSO

Учебный gRPC-сервис аутентификации на Go: регистрация, вход с выдачей JWT, проверка прав
администратора. Хранилище — PostgreSQL через `pgx`, миграции — `goose`, конфигурация —
YAML + `cleanenv`.

Сервис рассчитан на несколько приложений-потребителей: каждое имеет свой `id` и свой `secret`
в таблице `apps`, токен подписывается секретом того приложения, для которого он выдан.
Пример потребителя — соседний [todo-api](../todo-api).

## Требования

- Go 1.26+
- Docker (для PostgreSQL)
- [goose](https://github.com/pressly/goose): `go install github.com/pressly/goose/v3/cmd/goose@latest`

## Быстрый старт

```bash
# 1. Поднять PostgreSQL (наружу порт 5433)
docker compose up -d

# 2. Накатить схему
export DSN="postgres://postgres_user:secret@localhost:5433/sso?sslmode=disable"
goose -dir ./migrations postgres "$DSN" up

# 3. Зарегистрировать приложение, которому будут выдаваться токены
docker compose exec postgres psql -U postgres_user -d sso \
  -c "INSERT INTO apps (id, name, secret) VALUES (2, 'todo-api', 'secret')"

# 4. Запустить сервис (gRPC на :44044)
go run ./cmd/sso --config=./config/local.yaml
```

Шаги 2 и 3 нужны один раз на свежую базу; после `docker compose down -v` их надо повторить.

**Порт 5433, а не 5432** — 5432 на хосте занят базой todo-api. Внутри контейнера Postgres
слушает обычный 5432, наружу проброшен 5433.

## Приложения (таблица `apps`)

Токен подписывается алгоритмом HS256 секретом приложения, поэтому потребитель должен знать
тот же `secret` — он проверяет подпись локально, не обращаясь к sso.

| Поле     | Смысл |
|----------|-------|
| `id`     | передаётся в `Login` как `app_id` и попадает в claim `app_id` токена |
| `name`   | человекочитаемое имя |
| `secret` | ключ подписи; у потребителя лежит в его конфигурации |

Добавить приложение — обычный `INSERT`, как в шаге 3. Секрет на проде должен быть длинной
случайной строкой и различаться для каждого приложения.

## Конфигурация

`config/local.yaml`:

| Ключ           | Описание |
|----------------|----------|
| `env`          | `local` / `dev` / `prod` — формат и уровень логов |
| `database_url` | подключение к PostgreSQL |
| `token_ttl`    | время жизни выдаваемого JWT |
| `grpc.port`    | порт gRPC-сервера |

Путь к файлу задаётся флагом `--config` или переменной `CONFIG_PATH`. Значение
`database_url` можно переопределить переменной окружения **`SSO_DATABASE_URL`**
(именно с префиксом `SSO_`: голая `DATABASE_URL` занята соседним todo-api, и её случайный
экспорт увёл бы сервис в чужую базу).

## API

Сервис `auth.Auth`, контракт — в отдельном репозитории [protos](https://github.com/AlexKorshun/protos).

| RPC        | Запрос                          | Ответ      | Ошибки |
|------------|---------------------------------|------------|--------|
| `Register` | `email`, `password`             | `user_id`  | `AlreadyExists`, `InvalidArgument` |
| `Login`    | `email`, `password`, `app_id`   | `token`    | `InvalidArgument` — неверные креды либо неизвестный `app_id` |
| `IsAdmin`  | `user_id`                       | `is_admin` | `NotFound`, `InvalidArgument` |

Пароли хранятся как bcrypt-хеш. Токен содержит claims `uid`, `email`, `app_id`, `exp`.

`Login` отвечает одинаково и на несуществующего пользователя, и на неверный пароль —
чтобы по ответу нельзя было проверять, зарегистрирован ли email.

### Пример вызова

```bash
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest

grpcurl -plaintext -import-path ../protos/proto -proto sso/sso.proto \
  -d '{"email":"me@test.com","password":"secret123"}' localhost:44044 auth.Auth/Register

grpcurl -plaintext -import-path ../protos/proto -proto sso/sso.proto \
  -d '{"email":"me@test.com","password":"secret123","app_id":2}' localhost:44044 auth.Auth/Login
```

## Тесты

`tests/` — функциональные тесты: они ходят по gRPC в **уже запущенный** сервис, а не поднимают
его сами. Порядок такой:

```bash
docker compose up -d
go run ./cmd/sso --config=./config/local.yaml   # в отдельном терминале

go test ./tests/... -v -count=1
```

Конфигурация тестов — `config/local_tests.yaml` (путь к нему зашит в `tests/suite`).
`grpc.timeout` в нём служит таймаутом контекста на весь тест, поэтому там `5s`, а не `10h`.
`-count=1` отключает кеш результатов, иначе повторный прогон без изменений вернёт `(cached)`.

Тесты используют общую базу и создают пользователей со случайными email (`gofakeit`),
так что данные после прогонов накапливаются — это нормально, при желании чистится
`DELETE FROM users`.

Запустить один тест: `go test ./tests/ -run '^TestRegisterLogin_Login_HappyPath$' -v -count=1`.

## Развёртывание

```bash
go build -o sso ./cmd/sso
./sso --config=/path/to/prod.yaml
```

Что обязательно поменять относительно локального конфига:

1. **`database_url`** — боевая база, `sslmode` не `disable`. Пароль лучше передавать
   переменной `SSO_DATABASE_URL`, а не хранить в yaml под git.
2. **Секреты приложений** в таблице `apps` — длинные случайные строки, свои для каждого
   потребителя. Значение `secret` из примеров использовать нельзя.
3. **`env: "prod"`** — логи станут JSON с уровнем Info.

Миграции на боевую базу накатываются тем же `goose ... up`. gRPC-порт наружу выставлять не
нужно: sso дёргают другие сервисы, а не браузеры.

## Структура проекта

```
cmd/sso/main.go              точка входа
internal/
  app/                       сборка зависимостей, запуск gRPC-сервера
  grpc/auth/                 gRPC-хендлеры, валидация, коды ошибок
  services/auth/             бизнес-логика: bcrypt, выдача токена
  storage/postgres/          репозиторий на pgx
  storage/storage.go         sentinel-ошибки слоя хранения
  config/                    загрузка конфигурации
domain/models/               доменные модели
lib/jwt/                     генерация JWT
migrations/                  SQL-миграции (goose)
tests/                       функциональные тесты + suite
```
