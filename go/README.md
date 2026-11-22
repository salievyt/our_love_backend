# Our Love — Go API

Постепенная замена Django-бэкенда на Go по схеме strangler fig:
Go-сервис работает с **той же базой Postgres** и принимает **те же JWT**,
что выдаёт Django (simplejwt, HS256, `SECRET_KEY`). nginx отправляет в Go только
перенесённые разделы, остальное продолжает обслуживать Django. Клиенты (iOS,
Android) ничего не замечают: формат ответов совпадает с DRF — пагинация
`{count, next, previous, results}`, ошибки `{"detail": ...}` и `{"поле": ["..."]}`,
даты и время как в DRF (часовой пояс `Europe/Moscow`).

## Статус переноса

| Раздел | Путь | Статус |
|---|---|---|
| Настроения | `/api/v1/moods/` | ✅ Go |
| Важные даты | `/api/v1/dates/` | ✅ Go |
| Места | `/api/v1/places/` | ⏳ Django (нужны загрузки и подписанные ссылки на медиа) |
| Капсулы | `/api/v1/capsules/` | ⏳ Django |
| Дневник | `/api/v1/diary/` | ⏳ Django |
| Задания | `/api/v1/tasks/` | ⏳ Django |
| Вопросы | `/api/v1/questions/` | ⏳ Django |
| Альбом | `/api/v1/albums/` | ⏳ Django |
| Аккаунты, вход, пары | `/api/v1/auth/` | ⏳ Django (последним: пароли Django, выпуск токенов) |
| Админка | `/admin/` | Django (остаётся) |

Пока Django жив, **схема базы — за миграциями Django**. Go только читает и пишет
в существующие таблицы (`moods_moodentry`, `dates_importantdate`, …).

## Отличия от Django-версии (исправленные баги)

- `POST /dates/` без пары → 400 «Сначала создайте пару», а не 500.
- `days_until` для 29 февраля в невисокосный год считает до 28 февраля, а не падает.
- Повторное настроение на ту же дату → 400 с ошибкой поля `date`, а не 500.
- `POST`/`PUT`/`PATCH` возвращают полный объект (DRF отдавал только записанные поля) —
  это надмножество прежнего ответа, клиенты его принимают.

## Запуск

```bash
make test                # юнит-тесты (без базы)
make test-integration    # нужен TEST_DATABASE_URL со схемой Django
make run                 # те же переменные, что у Django: SECRET_KEY, DB_*; PORT (8080)
```

Переменные окружения: `SECRET_KEY` (≥32 символов), `DB_NAME`, `DB_USER`,
`DB_PASSWORD`, `DB_HOST`, `DB_PORT` или целиком `DATABASE_URL`; `PORT`, `TIME_ZONE`.

## Зависимости

Только драйвер Postgres `github.com/lib/pq` (без транзитивных зависимостей).
JWT проверяется стандартной библиотекой (`internal/auth/jwt.go`).
