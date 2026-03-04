# go-musthave-shortener-tpl

Шаблон репозитория для трека «Сервис сокращения URL».

## Начало работы

1. Склонируйте репозиторий в любую подходящую директорию на вашем компьютере.
2. В корне репозитория выполните команду `go mod init <name>` (где `<name>` — адрес вашего репозитория на GitHub без префикса `https://`) для создания модуля.

## Обновление шаблона

Чтобы иметь возможность получать обновления автотестов и других частей шаблона, выполните команду:

```
git remote add -m v2 template https://github.com/Yandex-Practicum/go-musthave-shortener-tpl.git
```

Для обновления кода автотестов выполните команду:

```
git fetch template && git checkout template/v2 .github
```

Затем добавьте полученные изменения в свой репозиторий.

## Запуск автотестов

Для успешного запуска автотестов называйте ветки `iter<number>`, где `<number>` — порядковый номер инкремента. Например, в ветке с названием `iter4` запустятся автотесты для инкрементов с первого по четвёртый.

При мёрже ветки с инкрементом в основную ветку `main` будут запускаться все автотесты.

Подробнее про локальный и автоматический запуск читайте в [README автотестов](https://github.com/Yandex-Practicum/go-autotests).

## Бенчмарки и профилирование памяти

### Запуск бенчмарков

```bash
# Все бенчмарки сервисного слоя
go test -run=^$ -bench=. -benchmem ./internal/service/

# Все бенчмарки хранилища
go test -run=^$ -bench=. -benchmem ./internal/store/
```

### Результаты бенчмарков (Apple M1 Pro)

#### `internal/service`

| Benchmark | ops/s | ns/op | B/op | allocs/op |
|---|---|---|---|---|
| `CodeGeneratorGenerateCode` | 24 042 694 | 48 | 8 | 1 |
| `CodeGeneratorGenerateBatchCodes` (100) | 238 012 | 5 554 | 2 592 | 101 |
| `URLServiceCreateShortURL` | 8 811 942 | 137 | 40 | 2 |
| `URLServiceCreateShortURL_Duplicate` | 13 998 820 | 95 | 8 | 1 |
| `URLServiceCreateShortURLsBatch` (10) | 231 746 | 8 784 | 5 999 | 22 |

#### `internal/store`

| Benchmark | ops/s | ns/op | B/op | allocs/op |
|---|---|---|---|---|
| `StoreWrite` | 1 000 000 | 1 087 | 677 | 4 |
| `StoreRead` | 67 641 199 | 17.9 | 0 | 0 |
| `StoreCreateOrGetURL_New` | 1 255 514 | 885 | 554 | 4 |
| `StoreCreateOrGetURL_Existing` (1000 entries) | 26 308 914 | 46.2 | 0 | 0 |
| `StoreGetURLsByUserID` (100 URLs) | 156 606 | 7 539 | 6 656 | 101 |
| `StoreIsCodeUnique` | 42 570 740 | 27.5 | 0 | 0 |

### Анализ памяти с pprof

Профили сохранены в директории `profiles/`:
- `profiles/base.pprof` — до оптимизаций
- `profiles/result.pprof` — после оптимизаций

#### Анализ базового профиля (`pprof -top profiles/base.pprof`)

```
File: store.test
Type: alloc_space
Showing nodes accounting for 1782.04MB, 99.40% of 1792.73MB total
      flat  flat%   sum%        cum   cum%
  440.05MB 24.55% 24.55%   545.55MB 30.43%  net/url.(*URL).JoinPath
  422.23MB 23.55% 48.10%   422.23MB 23.55%  github.com/avc-dev/url-shortener/internal/store.(*Store).Write
  368.05MB 20.53% 68.63%   368.05MB 20.53%  net/url.parse
  227.71MB 12.70% 81.33%  1265.32MB 70.58%  github.com/avc-dev/url-shortener/internal/store.(*Store).GetURLsByUserID
  124.01MB  6.92% 88.25%   124.01MB  6.92%  strings.(*Builder).grow
      67MB  3.74% 91.99%       67MB  3.74%  fmt.Sprintf
      38MB  2.12% 94.10%       38MB  2.12%  path.(*lazybuf).string (inline)
      34MB  1.90% 96.00%   105.50MB  5.88%  path.Join
```

**Вывод**: `url.JoinPath` отвечал за **1.04 ГБ** аллокаций (58% общего объёма), вызываясь для каждого URL в `GetURLsByUserID`. Плюс O(n) линейный поиск в `CreateOrGetURL` при 1000 записях давал 110 993 ns/op.

#### Применённые оптимизации

1. **`store.go` — замена `url.JoinPath` на конкатенацию строк** в `GetURLsByUserID`:
   - До: `url.JoinPath(base, code)` — минимум 4 аллокации на URL (`url.Parse`, `path.Join`, `strings.Builder.grow`, финальная строка)
   - После: `base + string(code)` — 1 аллокация на URL
   - Результат: **708 → 101 allocs/op** (-85.7%), **50 912 → 6 656 B/op** (-86.9%)

2. **`store.go` — добавлен обратный индекс `urlIndex map[URL]Code`**:
   - `CreateOrGetURL`: O(n) перебор карты → O(1) lookup
   - Результат: **5 414 → 46 ns/op** (×117 быстрее при 1000 записей)
   - `CreateOrGetURL_New`: **110 993 → 885 ns/op** (×125 быстрее)

3. **`store.go` — предварительное выделение среза** в `GetURLsByUserID` (`make([]UserURLResponse, 0, count)`): устраняет перераспределения при `append`.

4. **`code_generator.go` — `[CodeLength]byte` вместо `make([]byte, CodeLength)`**: массив фиксированного размера гарантированно остаётся на стеке.

5. **`url_service.go` — исправлен O(n²) реверс** в `CreateShortURLsBatch`: добавлена карта `codeForURL[URL]Code`, восстановление порядка за O(n) вместо двойного перебора.

#### Diff профилей (`pprof -top -diff_base=profiles/base.pprof profiles/result.pprof`)

```
File: store.test
Type: alloc_space
Showing nodes accounting for 547.35MB, 30.53% of 1792.73MB total
      flat  flat%   sum%        cum   cum%
  792.59MB 44.21% 44.21%  -245.02MB 13.67%  github.com/avc-dev/url-shortener/internal/store.(*Store).GetURLsByUserID
  581.86MB 32.46% 76.67%   581.86MB 32.46%  github.com/avc-dev/url-shortener/internal/store.(*Store).CreateOrGetURL
 -440.05MB 24.55% 52.12%  -545.55MB 30.43%  net/url.(*URL).JoinPath
 -368.05MB 20.53% 31.59%  -368.05MB 20.53%  net/url.parse
  172.51MB  9.62% 41.21%   172.51MB  9.62%  github.com/avc-dev/url-shortener/internal/store.(*Store).Write
 -124.01MB  6.92% 34.30%  -124.01MB  6.92%  strings.(*Builder).grow
     -38MB  2.12% 32.18%      -38MB  2.12%  path.(*lazybuf).string (inline)
     -34MB  1.90% 30.28%  -105.50MB  5.88%  path.Join
  -33.50MB  1.87% 28.41%   -33.50MB  1.87%  path.(*lazybuf).append (inline)
         0     0% 30.53%  -245.02MB 13.67%  github.com/avc-dev/url-shortener/internal/store.BenchmarkStoreGetURLsByUserID
         0     0% 30.53%  -124.01MB  6.92%  net/url.(*URL).String
         0     0% 30.53% -1037.61MB 57.88%  net/url.JoinPath
         0     0% 30.53%  -368.05MB 20.53%  net/url.Parse
         0     0% 30.53%   -71.50MB  3.99%  path.Clean
         0     0% 30.53%  -124.01MB  6.92%  strings.(*Builder).Grow
```

Отрицательные значения подтверждают снижение потребления памяти:
- `net/url.JoinPath` (cumulative): **-1 037.61 MB**
- `net/url.(*URL).JoinPath` (flat): **-440.05 MB**
- `net/url.parse`: **-368.05 MB**
- `strings.(*Builder).grow`: **-124.01 MB**

## Структура проекта

Приведённая в этом репозитории структура проекта является рекомендуемой, но не обязательной.

Это лишь пример организации кода, который поможет вам в реализации сервиса.

При необходимости можно вносить изменения в структуру проекта, использовать любые библиотеки и предпочитаемые структурные паттерны организации кода приложения, например:
- **DDD** (Domain-Driven Design)
- **Clean Architecture**
- **Hexagonal Architecture**
- **Layered Architecture**
