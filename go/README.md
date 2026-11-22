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
| Места | `/api/v1/places/` | ✅ Go (загрузка фото, подписанные ссылки на медиа) |
| Капсулы | `/api/v1/capsules/` | ⏳ Django |
| Дневник | `/api/v1/diary/` | ⏳ Django |
| Задания | `/api/v1/tasks/` | ⏳ Django |
| Вопросы | `/api/v1/questions/` | ⏳ Django |
| Альбом | `/api/v1/albums/` | ⏳ Django |
| Аккаунты, вход, пары | `/api/v1/auth/` | ⏳ Django (последним: пароли Django, выпуск токенов) |
| Админка | `/admin/` | Django (остаётся) |

Пока Django жив, **схема базы — за миграциями Django**. Go только читает и пишет
в существующие таблицы (`moods_moodentry`, `dates_importantdate`, `places_place`, …).
Медиафайлы — тоже общие: том `MEDIA_ROOT`, раздаёт их по-прежнему Django.

## Отличия от Django-версии (исправленные баги)

- `POST /dates/` без пары → 400 «Сначала создайте пару», а не 500.
- `days_until` для 29 февраля в невисокосный год считает до 28 февраля, а не падает.
- Повторное настроение на ту же дату → 400 с ошибкой поля `date`, а не 500.
- `/places/` в Django отвечал 500 на любой запрос (`models.Q` без импорта во `views.py`).
  Go делает то, что задумано: видны места, созданные пользователем или его партнёром
  (`pair_q(user, "created_by")`), пара для создания не нужна.
- `/places/` принимает и JSON (Django — только multipart/form-data и urlencoded).
- `POST`/`PUT`/`PATCH` возвращают полный объект (DRF отдавал только записанные поля) —
  это надмножество прежнего ответа, клиенты его принимают.

## Медиа и загрузки (`internal/media`)

**Подписанные ссылки.** API отдаёт файлы ссылками
`/media/<путь>?exp=<unix>&sig=<подпись>` (`config/media.py`). Проверяет подпись и
отдаёт файл Django (nginx `/media/` → Django → `X-Accel-Redirect`), поэтому Go
подписывает ровно как `Signer(salt="media-link").signature(f"{name}:{exp}")`:
HMAC-SHA256 с ключом `sha256("media-linksigner" + SECRET_KEY)`, urlsafe base64 без `=`;
`exp = (now // 3600 + 2) * 3600`; путь экранируется как `urllib.parse.quote`.
Совместимость закреплена тестами с эталонными значениями, полученными из Django
(`internal/media/sign_test.go`). Ссылка абсолютная: схема и хост из
`X-Forwarded-Proto`/`X-Forwarded-Host`, как `build_absolute_uri`.

**Загрузки.** Файл сохраняется в `MEDIA_ROOT` как `places/<uuid hex><ext>`
(`UploadTo`), имя от клиента не используется. Проверки — как у `ImageField` +
`IMAGE_VALIDATORS`, с теми же сообщениями: это картинка, расширение разрешено,
не больше 15 МБ, имя файла ≤ 100 символов. Весь multipart-запрос — до 50 МБ
(как `client_max_body_size` в nginx).

Решение по форматам (только стандартная библиотека):

- JPEG, PNG, GIF — `image.DecodeConfig` (`image/jpeg`, `image/png`, `image/gif`).
- WebP — stdlib его не декодирует, поэтому проверяется заголовок контейнера
  RIFF/WEBP и первого чанка (VP8, VP8L, VP8X) с размерами. Проверено на файлах,
  созданных Pillow (`internal/media/testdata`).
- HEIC/HEIF — отклоняются. Django без `pillow-heif` (его нет в requirements) их тоже
  не принимает, так что поведение для клиентов не меняется (по `config/uploads.py`
  клиенты шлют `photo.jpg`). Если понадобится HEIC — это решение для обоих сервисов,
  в Go без внешних зависимостей оно не делается.
- Ограничение: как и `Image.verify()` в Pillow, проверка смотрит заголовок, а не
  декодирует пиксели целиком — файл с верным заголовком и битыми данными пройдёт.
  Картинки больше ~179 млн пикселей отклоняются, как `DecompressionBombError` в Pillow.
- Расширения: `jpg jpeg jpe jfif png apng gif webp`. Django берёт список у Pillow —
  он шире (bmp, tiff, …), такие файлы Go отклонит с ошибкой поля `photo`.

Заменённое или удалённое фото остаётся на диске — как в Django.

**Права на том.** Контейнер Go работает как `nonroot` (uid 65532), а Django пишет в
том от root. Перед первым запуском дайте Go каталог для записи:

```bash
docker compose exec api sh -c 'mkdir -p /app/media/places && chown 65532:65532 /app/media/places'
```

## Запуск

```bash
make test                # юнит-тесты (без базы)
make test-integration    # нужен TEST_DATABASE_URL со схемой Django
make run                 # те же переменные, что у Django: SECRET_KEY, DB_*; PORT (8080)
```

Переменные окружения: `SECRET_KEY` (≥32 символов), `DB_NAME`, `DB_USER`,
`DB_PASSWORD`, `DB_HOST`, `DB_PORT` или целиком `DATABASE_URL`; `PORT`, `TIME_ZONE`;
`MEDIA_ROOT` (по умолчанию `/app/media` — общий том с Django).

## Зависимости

Только драйвер Postgres `github.com/lib/pq` (без транзитивных зависимостей).
JWT проверяется стандартной библиотекой (`internal/auth/jwt.go`).
