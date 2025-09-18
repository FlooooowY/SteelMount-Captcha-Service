# SteelMount Captcha Service

## Описание задания

Реализован высокопроизводительный сервис капчи с защитой от ботов, который генерирует интерактивные задания в реальном времени и интегрируется с внешними системами через gRPC и WebSocket. Поддерживает различные типы капч: клики, перетаскивание, свайпы и мини-игры. Используется Clean Architecture с элементами DDD.

## Функциональность

**Генерация капч в реальном времени**: сервер генерирует интерактивные задания (drag&drop, клики, свайпы, игры) на динамически выбранных портах 38000-40000 с учетом сложности 0-100.

**Интеграция с балансером**: автоматическая регистрация в балансере через gRPC, поиск свободного порта, heartbeat каждую секунду и graceful shutdown с уведомлением STOPPED.

**WebSocket события через postMessage**: HTML капчи содержат весь CSS/JS и взаимодействуют с родительским окном через `window.top.postMessage` для компактной передачи данных.

**Защита от ботов**: rate limiting (60 запросов/мин), IP блокировка после неудачных попыток, анализ user-agent, адаптивное ограничение скорости.

**Мониторинг и метрики**: Prometheus метрики, Grafana дашборды, алерты безопасности, health checks и статистика атак.

**Поддержка множественных клиентов одновременно** и корректное завершение работы сервера.

## Сборка

```bash
go build -o captcha-service.exe cmd/server/main.go
```

## Запуск сервера

```bash
go run ./cmd/server/main.go
```

## Запуск тестов

```bash
go test ./...
```

## Структура проекта

```
cmd/server/main.go          – точка входа
internal/domain/            – сущности и базовые модели (Challenge, Event, ChallengeResult)
internal/repository/        – интерфейсы и реализация хранилища капч
internal/usecase/           – бизнес-логика (создание капч, валидация, обработка событий)
internal/transport/grpc/    – gRPC сервер и клиент балансера
internal/websocket/         – WebSocket сервер и обработчики
internal/captcha/           – движок генерации капч (click, drag_drop, swipe, game)
internal/security/          – защита от ботов (rate limiter, IP blocker, bot detector)
internal/monitoring/        – метрики Prometheus и алерты
internal/config/            – загрузка настроек из файлов и переменных окружения
internal/server/            – управление жизненным циклом приложения
proto/                      – protobuf определения для gRPC сервисов
```

## Архитектурные решения

**Разделение ответственности**: каждый слой отвечает за свою область

- `domain` – сущности и модели
- `repository` – хранение данных
- `usecase` – бизнес-логика
- `transport` – сетевые интерфейсы (gRPC/WebSocket)
- `captcha` – генерация капч
- `security` – защита от атак

**Стандартная структура Go**: `cmd/` для точки входа и `internal/` для реализации

**Потокобезопасность** с помощью `sync.RWMutex` и каналов

**Корректное завершение работы** через контексты и `sync.WaitGroup`

**Тестируемость**: каждый слой можно тестировать изолированно

**Clean Architecture** для легкого расширения и поддержки

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

## Интеграция

### Быстрая интеграция (2 шага):

1. **Подключение к балансеру**: Укажите URL балансера в переменной `BALANCER_URL`
2. **Использование WebSocket**: Подключитесь к `ws://localhost:38001/ws?client_id=your_id` и отправляйте события:

```javascript
// Создание капчи
ws.send(JSON.stringify({
  type: 'create_challenge',
  data: { challenge_type: 'click', complexity: 30 }
}));

// Валидация решения
ws.send(JSON.stringify({
  type: 'validate_challenge',
  data: { challenge_id: 'uuid', answer: {...} }
}));
```

### Интеграция через WebSocket

```javascript
// Подключение к WebSocket
const ws = new WebSocket('ws://localhost:38001/ws?client_id=your_client_id')

// Создание капчи
function createCaptcha(type, complexity) {
	ws.send(
		JSON.stringify({
			type: 'create_challenge',
			data: {
				challenge_type: type, // 'click', 'drag_drop', 'swipe', 'game'
				complexity: complexity, // 0-100
			},
		})
	)
}

// Обработка ответов
ws.onmessage = function (event) {
	const data = JSON.parse(event.data)

	if (data.type === 'challenge_created') {
		// Получили HTML капчи
		document.getElementById('captcha-container').innerHTML =
			data.data.html_content
	}

	if (data.type === 'challenge_validated') {
		// Результат проверки
		console.log('Капча решена:', data.data.solved)
		console.log('Уверенность:', data.data.confidence)
	}
}

// Валидация решения
function validateCaptcha(challengeId, answer) {
	ws.send(
		JSON.stringify({
			type: 'validate_challenge',
			data: {
				challenge_id: challengeId,
				answer: answer,
			},
		})
	)
}
```

### Интеграция через gRPC

```javascript
// Node.js пример
const grpc = require('@grpc/grpc-js')
const protoLoader = require('@grpc/proto-loader')

// Загрузка protobuf
const packageDefinition = protoLoader.loadSync('proto/captcha/v1/captcha.proto')
const captchaProto = grpc.loadPackageDefinition(packageDefinition).captcha.v1

// Подключение к сервису
const client = new captchaProto.CaptchaService(
	'localhost:38001', // динамический порт из диапазона 38000-40000
	grpc.credentials.createInsecure()
)

// Создание капчи
client.NewChallenge(
	{
		complexity: 50,
	},
	(error, response) => {
		if (error) {
			console.error('Ошибка:', error)
			return
		}

		console.log('Challenge ID:', response.challenge_id)
		console.log('HTML:', response.html)
	}
)
```

### Интеграция в iframe

```html
<!-- В вашем приложении -->
<iframe
	id="captcha-iframe"
	src="data:text/html;base64,..."
	width="400"
	height="300"
></iframe>

<script>
	// Слушаем сообщения от капчи
	window.addEventListener('message', function (event) {
		if (event.data && event.data.type === 'captcha:sendData') {
			// Капча отправила данные (клики, перетаскивания и т.д.)
			const captchaData = JSON.parse(event.data.data)
			console.log('Данные капчи:', captchaData)

			// Отправляем на валидацию
			validateCaptcha(captchaData.challenge_id, captchaData)
		}
	})

	// Отправка команд в капчу
	function sendCommandToCaptcha(command, data) {
		const iframe = document.getElementById('captcha-iframe')
		iframe.contentWindow.postMessage(
			{
				type: 'captcha:serverData',
				data: JSON.stringify({ command: command, ...data }),
			},
			'*'
		)
	}
</script>
```

## Примеры использования

### E-commerce (защита checkout)

```javascript
// При оформлении заказа
async function processCheckout(orderData) {
	if (await isHighRiskOrder(orderData)) {
		const captcha = await createCaptcha('drag_drop', 70)
		const result = await showCaptchaModal(captcha.html)

		if (!result.solved) {
			throw new Error('Капча не решена')
		}
	}

	return processOrder(orderData)
}
```

### API Protection

```javascript
// Middleware для защиты API
app.use('/api/sensitive', async (req, res, next) => {
	const clientIP = req.ip

	if (await isRateLimited(clientIP)) {
		const captcha = await createCaptcha('click', 50)
		return res.status(429).json({
			error: 'Rate limit exceeded',
			captcha_html: captcha.html,
			challenge_id: captcha.challenge_id,
		})
	}

	next()
})
```

### Регистрация пользователей

```javascript
// При регистрации
async function registerUser(userData) {
	// Всегда показываем капчу при регистрации
	const captcha = await createCaptcha('swipe', 40)
	const captchaResult = await validateUserCaptcha(captcha)

	if (!captchaResult.solved || captchaResult.confidence < 70) {
		throw new Error('Пройдите проверку капчи')
	}

	return createUser(userData)
}
```

## Конфигурация

Основные переменные окружения:

- `BALANCER_URL` – URL балансера (по умолчанию localhost:50051)
- `MIN_PORT/MAX_PORT` – диапазон портов (38000-40000)
- `REDIS_URL` – подключение к Redis
- `LOG_LEVEL` – уровень логирования
- `METRICS_PORT` – порт метрик (9090)

## Docker

```bash
# Запуск через Docker Compose
docker-compose up -d

# Сборка образа
docker build -t captcha-service .

# Запуск контейнера
docker run -p 38000-40000:38000-40000 -p 9090:9090 captcha-service
```

## Тестирование

```bash
# Все тесты
make test-all

# Отдельные категории
make test-unit           # Юнит-тесты
make test-integration    # Интеграционные тесты
make test-performance    # Тесты производительности
make test-security       # Тесты безопасности

# С покрытием
make test-coverage

# Бенчмарки
make test-bench
```

## Мониторинг

```bash
# Проверка метрик
curl http://localhost:9090/metrics

# Статистика безопасности
curl http://localhost:9090/security/stats

# Health check
curl http://localhost:9090/health
```

