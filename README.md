# Kanelbulle

Система учёта и оценивания лабораторных работ с опционально затухающими очками после дедлайна.
Управляется через телеграм-бота

## Компоненты

- **Server** - API для аналитики и подсчёта баллов (включая экспорт в Google Sheets)
- **Bot** - telegram-бот для выдачи токенов студентам и админских команд

Данные хранятся в постгресе (либо sqlite). Рядом ещё должен быть редис для хранения некритичных данных типа маппинга student.id на конкретном курсе к телеге.

## Quickstart

Проще всего запустить с docker compose, надо только скопировать и отредактировать конфиг

```bash
cp config.toml.template config.toml
docker compose up -d
```

## Конфигурация

Все настройки в `config.toml.template`. Аджаст аккордингли

## API

| Метод | Endpoint | Описание |
|-------|----------|----------|
| POST | `/api/v1/{course}/analytics` | Отправить событие |
| GET | `/api/v1/{course}/analytics` | Получить информацию о лабах |
| GET | `/api/v1/{course}/analytics/finish` | Получить события завершения |
| GET | `/api/v1/{course}/scoring` | Получить баллы |
| GET | `/metrics` | Prometheus-метрики, легонько анонимизированные |

Подробнее с примерами: [SERVER_API.md](SERVER_API.md)

## Команды бота

**Студенты:**
- `/token` - Получить токен для API
- `/help` - Справка

**Админы:**
- `/new_course COURSE_CODE` - Создать курс со списком студентов
- `/lab add <course> <lab> score <N> deadline <YYYY-MM-DD>` - Добавить лабу
- `/lab list <course>` - Список лаб
- `/override set <course> <lab> <student> score <N> reason <text>` - Перебить оценку студенту за какую-то лабу
- `/remap_student <course> <student.id> @new_username` - Изменить телеграм студента

## Сборка

```bash
make build
make docker-build
VERSION=v1.0 make docker-build
```

## Деплой `k8s`

Скопировать шаблоны, заполнить значения, задеплоить:

```bash
cp k8s/kanelbulle-configmap.template.yml k8s/kanelbulle-configmap.yml
cp k8s/secrets.template.yaml k8s/secrets.yaml
kubectl apply -f k8s/
```

_Outdatchi_

## Лицензия

[WTFPL](LICENSE)
