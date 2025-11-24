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

