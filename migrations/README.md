# Database Migrations

В данной директории содержатся файлы миграций базы данных для PostgreSQL.

## Структура

- `schema/` - SQL файлы миграций в формате `{version}_{name}.{up|down}.sql`

## Использование

### Установка инструмента миграций

```bash
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

### Запуск миграций

```bash
# Установить переменную окружения POSTGRES_URL
export POSTGRES_URL="postgres://postgres:postgres@localhost:5433/urlshortener?sslmode=disable"

# Применить все миграции
make migrate-up

# Откатить все миграции
make migrate-down
```

### Ручное управление

```bash
# Применить миграции
migrate -path migrations/schema -database "${POSTGRES_URL}" -verbose up

# Откатить миграции
migrate -path migrations/schema -database "${POSTGRES_URL}" -verbose down

# Показать текущую версию
migrate -path migrations/schema -database "${POSTGRES_URL}" version
```

## Миграции

- `000001_create_urls_table.up.sql` - Создает таблицу urls для хранения сокращенных URL
- `000001_create_urls_table.down.sql` - Удаляет таблицу urls
