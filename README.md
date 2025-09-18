# SteelMount Captcha Service

Реализован высокопроизводительный сервис капчи для защиты от ботов и ИИ-атак в цифровом маркетплейсе. Используется gRPC для интеграции с балансером и WebSocket для интерактивных капч.

## Функциональность

**Генерация капч в реальном времени**: сервер генерирует интерактивные задания на портах 38000-40000. Поддерживает drag&drop, клики, свайпы и игровые механики.

**Агрегация и валидация**: подсчитывается количество активных капч, время решения, успешность прохождения и анализируется поведение пользователей.

**gRPC API для интеграции**: на динамически выбранном порту доступны gRPC методы для создания капч и обработки событий.

**WebSocket события**: поддержка интерактивных капч через postMessage API для связи с браузером.

**Мониторинг и безопасность**: метрики Prometheus, rate limiting, защита от ботов и аномального трафика.

**Поддержка нескольких клиентов одновременно и корректное завершение работы сервера.**

## Интеграция с системой

### Подключение к балансеру

Сервис автоматически регистрируется в балансере при запуске:

1. **Поиск свободного порта**: сервис ищет первый доступный порт в диапазоне 38000-40000
2. **Регистрация в балансере**: отправляет gRPC запрос с информацией о себе
3. **Поддержание соединения**: каждую секунду отправляет heartbeat для подтверждения готовности
4. **Graceful shutdown**: при остановке отправляет событие STOPPED

### Работа с клиентами

**Создание капчи**:

```bash
# gRPC вызов
NewChallenge(ChallengeRequest) -> ChallengeResponse
```

**Обработка событий**:

```bash
# gRPC стрим
MakeEventStream(stream ClientEvent) -> stream ServerEvent
```

**WebSocket интеграция**:

```javascript
// Отправка данных из капчи
window.top.postMessage({
	type: 'captcha:sendData',
	data: binaryData,
})

// Получение данных от сервера
window.addEventListener('message', e => {
	if (e.data?.type === 'captcha:serverData') {
		// Обработка данных
	}
})
```

### Протоколы

**BalancerService** (порт балансера):

- `RegisterInstance` - регистрация инстанса капчи
- Heartbeat каждую секунду с событием READY/NOT_READY
- Событие STOPPED при завершении работы

**CaptchaService** (порт 38000-40000):

- `NewChallenge` - создание новой капчи
- `MakeEventStream` - поток событий WebSocket

## Сборка:

```bash
go build -o captcha-service cmd/server/main.go
```

## Запуск сервера:

```bash
go run ./cmd/server/main.go
```

## Запуск тестов:

```bash
go test ./...
```

## Структура проекта:

- `cmd/server/main.go` – точка входа
- `internal/domain` – сущности и базовые модели (Challenge, Event, Config)
- `internal/repository` – интерфейсы и реализация хранилища в памяти
- `internal/usecase` – бизнес-логика (генерация капч, валидация, агрегация)
- `internal/transport/grpc` – gRPC сервер и обработчики
- `internal/transport/websocket` – WebSocket события и postMessage API
- `internal/captcha` – движок генерации различных типов капч
- `internal/security` – rate limiting, IP блокировка, защита от ботов
- `internal/monitoring` – метрики Prometheus, логирование, health checks
- `internal/config` – загрузка настроек из YAML и переменных окружения
- `internal/server` – управление жизненным циклом приложения
- `proto/` – protobuf определения для gRPC сервисов

## Архитектурные решения

**Разделение ответственности**: каждый слой отвечает за свою область

- `domain` – сущности и модели
- `repository` – хранение данных и кэширование
- `usecase` – бизнес-логика генерации и валидации капч
- `transport` – сетевые интерфейсы (gRPC/WebSocket)
- `captcha` – движок различных типов капч
- `security` – защита и мониторинг

**Стандартная структура Go**: `cmd/` для точки входа и `internal/` для реализации

**Потокобезопасность** с помощью `sync.RWMutex` и каналов

**Корректное завершение работы** через контексты и `sync.WaitGroup`

**Тестируемость**: каждый слой можно тестировать изолированно

**Чистая архитектура** для легкого расширения и поддержки

**Высокая производительность**: 100+ RPS генерации, ≤8GB памяти на 10k задач

## API

**gRPC (динамический порт 38000-40000)**

- `NewChallenge(ChallengeRequest) returns (ChallengeResponse)` – создание новой капчи
- `MakeEventStream(stream ClientEvent) returns (stream ServerEvent)` – поток событий

**WebSocket события**

- Отправка данных: `window.top.postMessage({type:'captcha:sendData', data: binaryData})`
- Получение данных: `window.addEventListener('message', (e) => { if (e.data?.type === 'captcha:serverData') ... })`

**HTTP (порт 9090)**

- `GET /metrics` – метрики Prometheus
- `GET /health` – информация о статусе сервиса
- `GET /security/stats` – статистика безопасности

## Конфигурация

**Файлы конфигурации:**

- `config.yaml` - базовая конфигурация
- `config.dev.yaml` - настройки для разработки
- `config.prod.yaml` - настройки для продакшена (не в git)
- `config.test.yaml` - настройки для тестирования

**Переменные окружения:**

- `MIN_PORT` / `MAX_PORT` - диапазон портов (по умолчанию 38000-40000)
- `REDIS_URL` - URL Redis сервера
- `LOG_LEVEL` - уровень логирования
- `BALANCER_URL` - URL балансера
- `METRICS_PORT` - порт метрик (по умолчанию 9090)
