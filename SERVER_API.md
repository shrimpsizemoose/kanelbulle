# Server API

Принимает аналитические запросы, отдаёт аналитику. Выпячивает метрики через прометей

## Порты

| Окружение | Порт |
|-----------|------|
| Внутренний (контейнер) | 9999 |
| NodePort (k8s) | 30999 |
| docker compose | 9999 |

## Коды ответов &mdash; стандартные

| Код | Описание |
|-----|----------|
| 200 | OK |
| 400 | Неверный запрос (отсутствует course/lab/body) |
| 401 | Неверный токен или student id |
| 403 | Отсутствуют required headers |
| 404 | Отсутствуют required headers (GET-запросы) |
| 405 | Неверный HTTP-метод |

## Заголовки

Все запросы требуют заголовки, указанные в `config.toml` секции `[api.required_headers]`.

Для POST-запросов также нужны:
- `Authorization` — токен студента
- `X-STUDENT` — идентификатор студента (формат: `firstname.lastname`)
- `X-LAB` — номер лабы

## Примеры запросов

Предположим:
- Сервер доступен на `localhost:9999`
- Курс: `TECH01`
- Required headers: `X-Hello-There: General Kenobi`

### Отправить событие (POST /analytics)

```bash
curl -X POST "http://localhost:9999/api/v1/TECH01/analytics" \
  -H "X-Hello-There: General Kenobi" \
  -H "Authorization: sk-knlbll-abc123" \
  -H "X-STUDENT: ivan.ivanov" \
  -H "X-LAB: 01" \
  -H "Content-Type: application/json" \
  -d '{"event_type": "100_lab_finish"}'
```

### Получить все события (GET /analytics)

```bash
curl "http://localhost:9999/api/v1/TECH01/analytics" \
  -H "X-Hello-There: General Kenobi"
```

Ответ:
```json
{
  "rows": [
    [1699999999, "100_lab_finish", "01", "ivan.ivanov", "TECH01", "..."],
    ...
  ]
}
```

### Получить события завершения (GET /analytics/finish)

```bash
curl "http://localhost:9999/api/v1/TECH01/analytics/finish" \
  -H "X-Hello-There: General Kenobi"

# С человекочитаемыми датами
curl "http://localhost:9999/api/v1/TECH01/analytics/finish?human_dttm=true" \
  -H "X-Hello-There: General Kenobi"
```

### Получить баллы (GET /scoring)

```bash
curl "http://localhost:9999/api/v1/TECH01/scoring" \
  -H "X-Clacks-Overhead: GNU Terry Pratchett"
```

Ответ:
```json
{
  "stats": {
    "ivan.ivanov": {
      "01": 10,
      "02": 8
    },
    ...
  }
}
```

### Метрики (GET /metrics)

```bash
curl "http://localhost:9999/metrics"
```

Prometheus-метрики, заголовки не требуются.

## Типы событий по дефолту

| event_type | Ожидаемый смысл |
|------------|----------|
| `000_lab_start` | Студент начал лабу |
| `100_lab_finish` | Студент завершил лабу |

Сервер никак не препятствует добавлению своих событий, но аналитику начала/окончания по ним не считает

## Клиентская часть

Для реализации чекеров на Go (клиентских программ, отправляющих аналитику) см. [shrimpsizemoose/trekker](https://github.com/shrimpsizemoose/trekker)
