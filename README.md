# Sprint 3 — PostgreSQL + Redis + Docker

## Цель
Перевести проект с in-memory на реальное хранилище, закрепить SQL/индексы/JOIN/транзакции, научиться поднимать окружение через docker-compose и писать интеграционные тесты.

## Стек
- PostgreSQL + pgx/pgxpool
- Миграции: goose
- Redis (кеш)
- Docker Compose
- Интеграционные тесты (Postgres + Redis)
- Логирование (структурное)
- Swagger (опционально)

---

## 1) Инфраструктура и запуск
- Добавить `docker-compose.yml` с сервисами: `app`, `postgres`, `redis`, `pgadmin`.
- `app` подключается к Postgres через `pgxpool` (пул обязателен).
- Миграции goose применяются при запуске окружения (вариант на выбор)

---

## 2) Миграции и схема БД (goose)
Создать миграции для таблиц. Примерная структура:

### users
- `id` bigserial PK
- `username` text NOT NULL UNIQUE
- `created_at` timestamptz NOT NULL default now()

### posts
- `id` bigserial PK
- `author_id` bigint NOT NULL FK → users(id)
- `text` text NOT NULL
- `created_at` timestamptz NOT NULL default now()

### likes
- `post_id` bigint NOT NULL FK → posts(id)
- `user_id` bigint NOT NULL FK → users(id)
- `created_at` timestamptz NOT NULL default now()
- PK(`post_id`, `user_id`) (или UNIQUE на пару) — для идемпотентности

Индексы (обязательно):
- `posts(created_at)` (лента)
- `likes(post_id)` (агрегации)

---

## 3) Репозитории на Postgres
- Перевести app-слой на Postgres-репозитории (in-memory реализацию удалить, больше не использовать).
- `GET /posts` должен возвращать `likesCount` **одним SQL-запросом** (без N+1)

---

## 4) Идемпотентный лайк в БД
`POST /posts/{id}/like`

---

## 5) Транзакция
Сделать одну операцию, где транзакция реально нужна:

**POST /posts**: создать пост + авто-лайк автора в одной транзакции.  
Если лайк не вставился (ошибка/constraint) — пост не должен сохраниться.

---

## 6) Redis-кеш (минимально)
Кешировать `GET /posts`:
- ключ включает параметры (например, `posts:list:v1:limit=..:offset=..`)
- TTL 5–30 секунд
- инвалидация кеша при `POST /posts` (и опционально при лайке)

---

## 7) Интеграционные тесты
Тесты должны поднимать Postgres/Redis и применять миграции. Примеры для тестов:

1. `RegisterUser`: username уникален (повтор — ошибка).
2. `CreatePost`: пост создан, `author_id` корректный.
3. `LikePost`: 2 одинаковых лайка → 1 запись в `likes`, `likesCount` корректный.

---

## 8) Логирование (обязательно)
Структурные логи (slog/zap на выбор):
- request_id
- endpoint, status, latency
- ошибки БД логировать с контекстом (операция + текст/код ошибки)

---

## 9) Makefile (обязательно)
Добавить `Makefile` с командами для запуска, остановки сервиса, миграций. Пример команд:
- `make up`, `make down`
- `make test`, `make test-race`
- `make itest`
- `make migrate-up`, `make migrate-down`
- `make db-reset` (полный сброс локальной БД)

---

## 10) Конфигурация приложения
- Все настройки приложения (порт, параметры очереди/воркеров и т.д.) должны читаться из **переменных окружения (ENV)**.
- Для локального запуска допускается файл `.env` в корне репозитория, `docker compose` должен подхватывать его автоматически.
- В коде должен быть единый загрузчик конфигурации (`internal/config` или аналог): читает ENV → валидирует обязательные параметры → возвращает `Config struct`.
- Запрещено “хардкодить” значения конфигов в коде; пароль/DSN и прочие параметры должны приходить только через ENV.
- Можно использовать `caarlos0/env`, `godotenv` или другие либы.


## Критерии приёмки
- `docker compose` поднимает окружение, `app` работает.
- Миграции goose применяются автоматически.
- Данные (users/posts/likes) хранятся в Postgres.
- Лайки идемпотентны, `GET /posts` отдаёт `likesCount`.
- Redis-кеш работает для `GET /posts`.
- Интеграционные тесты проходят и реально ходят в Postgres/Redis.


## Полезные материалы
- Docker + Postgres — https://www.youtube.com/watch?v=qECVC6t_2mU
- Goose — https://github.com/pressly/goose и https://pressly.github.io/goose/#background
- pgxpool — https://medium.com/@neelkanthsingh.jr/understanding-database-connection-pools-and-the-pgx-library-in-go-3087f3c5a0c
- Индексы — https://disk.yandex.ru/d/Cs7MZ8rGuxSy9Q
- Redis - https://www.youtube.com/watch?v=GgRzHru9Hag
- Логирование -https://www.youtube.com/watch?v=KHS8hPh8mtU
   
   
  
