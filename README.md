# SSO

Учебный gRPC-сервис аутентификации на Go: регистрация, вход с выдачей JWT, проверка прав администратора.
Хранилище — PostgreSQL через `pgx`, миграции — `goose`, конфигурация — YAML + `cleanenv`.

## Требования

- Go 1.26+
- Docker (для PostgreSQL)
- [goose](https://github.com/pressly/goose): `go install github.com/pressly/goose/v3/cmd/goose@latest`

## Запуск

```bash
# 1. Поднять PostgreSQL (порт 5433)
docker compose up -d

# 2. Накатить схему
export DSN="postgres://postgres_user:secret@localhost:5433/sso?sslmode=disable"
goose -dir ./migrations postgres "$DSN" up

# 3. Добавить приложение, для которого выдаются токены
docker compose exec postgres psql -U postgres_user -d sso \
  -c "INSERT INTO apps (id, name, secret) VALUES (1, 'test', 'test-secret')"

# 4. Запустить сервис (gRPC на :44044)
go run ./cmd/sso --config=./config/local.yaml
```

Шаги 2 и 3 нужны один раз на свежую базу. После `docker compose down -v` их надо повторить.

## Конфигурация

`config/local.yaml`:

| Ключ           | Описание                                  |
|----------------|-------------------------------------------|
| `env`          | `local` / `dev` / `prod` — формат логов    |
| `database_url` | строка подключения к PostgreSQL           |
| `token_ttl`    | время жизни JWT                            |
| `grpc.port`    | порт gRPC-сервера                          |

Путь к файлу задаётся флагом `--config` или переменной `CONFIG_PATH`.

## API

Сервис `auth.Auth` (контракт — в [protos](https://github.com/AlexKorshun/protos)):

| RPC        | Запрос                      | Ответ       | Ошибки                                            |
|------------|-----------------------------|-------------|---------------------------------------------------|
| `Register` | `email`, `password`         | `user_id`   | `AlreadyExists`, `InvalidArgument`                |
| `Login`    | `email`, `password`, `app_id` | `token`   | `InvalidArgument` (креды или `app_id`)            |
| `IsAdmin`  | `user_id`                   | `is_admin`  | `NotFound`, `InvalidArgument`                     |

Пример вызова (нужен `grpcurl` и `.proto` из репозитория контрактов):

```bash
grpcurl -plaintext -import-path ../protos/proto -proto sso/sso.proto \
  -d '{"email":"a@b.c","password":"secret"}' localhost:44044 auth.Auth/Register
```

## Структура

```
cmd/sso/                 точка входа
internal/
  app/                   сборка зависимостей, запуск gRPC-сервера
  grpc/auth/             gRPC-хендлеры и валидация запросов
  services/auth/         бизнес-логика: bcrypt, выдача токена
  storage/postgres/      репозиторий на pgx
  config/                загрузка конфигурации
domain/models/           доменные модели
lib/jwt/                 генерация JWT
migrations/              SQL-миграции (goose)
tests/                   функциональные тесты (в работе)
```

## Статус

Основной функционал готов. В работе — функциональные тесты: им нужен запущенный сервис
и отдельный `config/local_tests.yaml`.
