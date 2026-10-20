# ТЗ: Backend «Соцсеть»

## 1. Описание проекта

Учебный backend-сервис **«соцсеть»** с REST API.  
Данные хранятся в памяти (без БД).  

---

## 2. Технологии и ограничения

- Язык: **Go**
- HTTP:
  - стандартный пакет `net/http`;
  - роутер `gin`.
- Хранение данных: **in-memory** (`map`, `slice`).

---

## 3. Сущности (минимальный набор)

### 3.1. User

Минимальные поля:

- `id` — строка, `uuid` или `int` (уникальный идентификатор);
- `username` — строка, уникальная;

Дополнительно по желанию:

- `fullName` — строка;
- `age` — число;
- другие простые поля по усмотрению.

### 3.2. Post

- `id` — строка или `int`;
- `authorID` — ссылка на `User.id`;
- `text` — строка;
- `createdAt` — время создания (любой удобный тип/формат);
- `likes` — слайс `[]*Like`.

### 3.3. Like

- `userID` — кто поставил лайк;
- `postID` — какому посту поставлен лайк.

---

## 4. REST API (минимальный набор эндпоинтов)

Все запросы и ответы — в формате **JSON**.

### 4.1. Пользователи

#### `POST /api/v1/register`

Регистрация нового пользователя.

- Request (body):
  - `username`
- Response:
  - данные созданного пользователя с полем `id`  

#### `GET /api/v1/users`

Получить список всех пользователей.

- Response:
  - массив объектов `User`.


> Обновление/удаление пользователей можно сделать по желанию.

---

### 4.2. Посты

#### `POST /api/v1/posts`

Создать пост от имени пользователя.

- Request:
  - `authorID` (в теле запроса);
  - `text` — текст поста.
- Response:
  - объект созданного `Post`.


#### `GET /api/v1/posts`

Получить список всех постов.

- Response:
  - массив объектов `Post`.
- Дополнительно:
  - можно сортировать по `createdAt` от новых к старым (по желанию).

---

### 4.3. Лайки

#### `POST /api/v1/posts/{id}/like`

Поставить лайк посту.

- Request:
  - `postID` — в пути (`{id}`);
  - `userID` — в теле запроса или в заголовке.
- Поведение:
  - если пользователь ещё не лайкал этот пост — лайк добавляется;
  - если уже лайкал — повторный лайк **не создаётся**.

#### Лайки в ответах

- Эндпоинты:
  - `GET /api/v1/posts/{id}`
  - `GET /api/v1/posts`
- Должны возвращать **текущее количество лайков** для каждого поста (`likesCount`).

---

## 5. Хранение данных (подсказки)

Рекомендуемые in-memory структуры:

### 5.1. Пользователи

- `usersByID`:  `map[ID]*User`

### 5.2. Посты

- `posts`:  `[]*Post`

---

## Рекомендуемая структура

В проекте есть несколько слоёв. Каждый слой отвечает за своё и **может зависеть только от нижележащих**, но не наоборот.

```text
├── cmd/
│   └── api/
│       └── main.go                 # точка входа: сборка зависимостей и запуск HTTP-сервера
└── internal/
    ├── domain/                     # доменные сущности и интерфейсы (без net/http)
    │   ├── user.go              
    │   ├── post.go              
    │   └── repository.go           # (опционально) интерфейсы UserRepo, PostRepo
    │
    ├── app/                        # бизнес-логика (сервисы)
    │   ├── user_service.go         # RegisterUser, GetUser, ListUsers
    │   └── post_service.go         # CreatePost, GetPost, ListPosts, LikePost
    │
    ├── infra/                      # инфраструктура: http, in-memory storage и т.п.
    │   ├── http/
    │   │   ├── router.go           # инициализация gin, регистрация маршрутов
    │   │   ├── users_handlers.go   # хендлеры /register, /users, /users/{id}
    │   │   └── posts_handlers.go   # хендлеры /posts, /posts/{id}, /posts/{id}/like
    │   │
    │   └── storage/
    │       └── memory/             # in-memory реализация репозиториев
    │           ├── users_repo.go   # usersByID map, методы Create/Find/List
    │           └── posts_repo.go   # posts slice/map, лайки и т.п.
    │
    └── pkg/                        # общие хелперы (опционально)
        ├── some-util.go           
```


---

## Критерии приёмки проекта

1. Проект запускается локально, HTTP-сервер слушает порт (например, `:8080`), все ответы отдаются в формате JSON.  
2. Реализованы и корректно работают эндпоинты: `POST /api/v1/register`, `GET /api/v1/users`.  
3. Реализованы и корректно работают эндпоинты: `POST /api/v1/posts`, `GET /api/v1/posts`.  
4. Реализован эндпоинт лайков `POST /api/v1/posts/{id}/like`, повторный лайк тем же пользователем не увеличивает счётчик.    
5. Код хранится в **GitLab-репозитории**. Работай в отдельной ветке, например `feature/sprint1`, созданной от `main`. 
Все изменения коммить в неё, в конце создай Merge Request из `feature/sprint1` в `main` в GitLab.  

---

## Ссылки и полезные материалы

### Git

- [Интерактивная тренировка GIT](https://learngitbranching.js.org/?locale=ru_RU)
- [GitFlow простым языком](https://wiki.merionet.ru/articles/git-flow-cto-eto-i-kak-rabotaet)

### Примеры архитектуры и структуры файлов

- [Статья по чистой архитектуре](https://pkritiotis.io/clean-architecture-in-golang/#clean-architecture-applied-in-go)
- [Пример проекта](https://github.com/pkritiotis/go-climb-clean-architecture-example)

### REST и JSON

- [Что такое REST](https://practicum.yandex.ru/blog/chto-takoe-rest-api-i-kak-rabotaet/)
- [JSON в Go](https://gobyexample.com/json)

### Тестирование

- [Тестирование в Go](https://quii.gitbook.io/learn-go-with-tests/go-fundamentals/hello-world)
- [Простой видос про тестирование в Go](https://www.youtube.com/watch?v=fMUNBJPhP6Y)

---

### Дисклеймер

Можно выходить за рамки этого ТЗ и добавлять дополнительный функционал по желанию — свои поля, новые эндпоинты, простую валидацию, фильтрацию, сортировку и т.д.
Но только если это действительно полезно для тебя и не займет много времени.
