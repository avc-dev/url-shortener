# middleware

Пакет содержит HTTP middleware для обработки входящих запросов.

## Компоненты

### Logger

Middleware для логирования всех HTTP запросов с использованием структурированного логгера zap.

**Логируемые данные:**
- HTTP метод
- URI запроса
- Статус код ответа
- Длительность обработки
- Размер ответа в байтах
- IP адрес клиента

**Использование:**
```go
logger, _ := zap.NewProduction()
defer logger.Sync()

r := chi.NewRouter()
r.Use(middleware.Logger(logger))
```

### GzipMiddleware

Middleware для сжатия HTTP запросов и ответов с использованием gzip.

**Функциональность:**
- Автоматическая распаковка входящих запросов с заголовком `Content-Encoding: gzip`
- Сжатие исходящих ответов для клиентов с заголовком `Accept-Encoding: gzip`
- Избирательное сжатие только для типов `application/json` и `text/html`
- Логирование всех ошибок сжатия/распаковки с контекстом

**Использование:**
```go
logger, _ := zap.NewProduction()
defer logger.Sync()

r := chi.NewRouter()
r.Use(middleware.GzipMiddleware(logger))
```
