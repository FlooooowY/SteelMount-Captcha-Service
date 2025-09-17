package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
)

// AlertLevel определяет уровень важности алерта
type AlertLevel string

const (
	AlertLevelInfo     AlertLevel = "INFO"
	AlertLevelWarning  AlertLevel = "WARNING"
	AlertLevelCritical AlertLevel = "CRITICAL"
)

// Alert представляет алерт о подозрительной активности
type Alert struct {
	ID          string                 `json:"id"`
	Level       AlertLevel             `json:"level"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Source      string                 `json:"source"`
	Data        map[string]interface{} `json:"data"`
	Timestamp   time.Time              `json:"timestamp"`
	Resolved    bool                   `json:"resolved"`
}

// AlertRule определяет правило для генерации алертов
type AlertRule struct {
	ID          string
	Name        string
	Description string
	Level       AlertLevel
	Condition   func(data map[string]interface{}) bool
	Cooldown    time.Duration // Минимальный интервал между алертами
	LastFired   time.Time
}

// AlertChannel интерфейс для отправки алертов
type AlertChannel interface {
	SendAlert(ctx context.Context, alert Alert) error
	GetName() string
}

// AlertManager управляет системой алертов
type AlertManager struct {
	rules    map[string]*AlertRule
	channels []AlertChannel
	alerts   []Alert
	mutex    sync.RWMutex
	
	// Метрики
	totalAlerts     int64
	alertsByLevel   map[AlertLevel]int64
	alertsBySource  map[string]int64
}

// NewAlertManager создает новый менеджер алертов
func NewAlertManager() *AlertManager {
	am := &AlertManager{
		rules:           make(map[string]*AlertRule),
		channels:        make([]AlertChannel, 0),
		alerts:          make([]Alert, 0),
		alertsByLevel:   make(map[AlertLevel]int64),
		alertsBySource:  make(map[string]int64),
	}
	
	// Добавляем стандартные правила безопасности
	am.addDefaultSecurityRules()
	
	return am
}

// AddRule добавляет правило алерта
func (am *AlertManager) AddRule(rule *AlertRule) {
	am.mutex.Lock()
	defer am.mutex.Unlock()
	
	am.rules[rule.ID] = rule
}

// AddChannel добавляет канал для отправки алертов
func (am *AlertManager) AddChannel(channel AlertChannel) {
	am.mutex.Lock()
	defer am.mutex.Unlock()
	
	am.channels = append(am.channels, channel)
}

// ProcessEvent обрабатывает событие и проверяет правила алертов
func (am *AlertManager) ProcessEvent(source string, data map[string]interface{}) {
	am.mutex.Lock()
	defer am.mutex.Unlock()
	
	now := time.Now()
	
	for _, rule := range am.rules {
		// Проверяем cooldown
		if now.Sub(rule.LastFired) < rule.Cooldown {
			continue
		}
		
		// Проверяем условие
		if rule.Condition(data) {
			alert := Alert{
				ID:          fmt.Sprintf("%s_%d", rule.ID, now.UnixNano()),
				Level:       rule.Level,
				Title:       rule.Name,
				Description: rule.Description,
				Source:      source,
				Data:        data,
				Timestamp:   now,
				Resolved:    false,
			}
			
			// Отправляем алерт
			am.sendAlert(alert)
			
			// Обновляем статистику
			rule.LastFired = now
			am.totalAlerts++
			am.alertsByLevel[rule.Level]++
			am.alertsBySource[source]++
			
			// Сохраняем алерт
			am.alerts = append(am.alerts, alert)
			
			// Ограничиваем количество сохраненных алертов
			if len(am.alerts) > 1000 {
				am.alerts = am.alerts[len(am.alerts)-1000:]
			}
		}
	}
}

// sendAlert отправляет алерт через все настроенные каналы
func (am *AlertManager) sendAlert(alert Alert) {
	ctx := context.Background()
	
	for _, channel := range am.channels {
		go func(ch AlertChannel, a Alert) {
			if err := ch.SendAlert(ctx, a); err != nil {
				log.Printf("Failed to send alert via %s: %v", ch.GetName(), err)
			}
		}(channel, alert)
	}
}

// GetAlerts возвращает последние алерты
func (am *AlertManager) GetAlerts(limit int) []Alert {
	am.mutex.RLock()
	defer am.mutex.RUnlock()
	
	if limit <= 0 || limit > len(am.alerts) {
		limit = len(am.alerts)
	}
	
	// Возвращаем последние алерты
	start := len(am.alerts) - limit
	if start < 0 {
		start = 0
	}
	
	result := make([]Alert, limit)
	copy(result, am.alerts[start:])
	
	return result
}

// GetStats возвращает статистику алертов
func (am *AlertManager) GetStats() map[string]interface{} {
	am.mutex.RLock()
	defer am.mutex.RUnlock()
	
	return map[string]interface{}{
		"total_alerts":      am.totalAlerts,
		"alerts_by_level":   am.alertsByLevel,
		"alerts_by_source":  am.alertsBySource,
		"active_rules":      len(am.rules),
		"active_channels":   len(am.channels),
		"recent_alerts":     len(am.alerts),
	}
}

// addDefaultSecurityRules добавляет стандартные правила безопасности
func (am *AlertManager) addDefaultSecurityRules() {
	// Правило: Высокий RPS с одного IP
	am.AddRule(&AlertRule{
		ID:          "high_rps_single_ip",
		Name:        "Высокий RPS с одного IP",
		Description: "Обнаружен высокий уровень запросов с одного IP адреса",
		Level:       AlertLevelWarning,
		Cooldown:    time.Minute * 5,
		Condition: func(data map[string]interface{}) bool {
			if rps, ok := data["rps_per_ip"].(float64); ok {
				return rps > 50 // Больше 50 RPS с одного IP
			}
			return false
		},
	})
	
	// Правило: Массовая блокировка IP
	am.AddRule(&AlertRule{
		ID:          "mass_ip_blocking",
		Name:        "Массовая блокировка IP",
		Description: "Обнаружена массовая блокировка IP адресов",
		Level:       AlertLevelCritical,
		Cooldown:    time.Minute * 10,
		Condition: func(data map[string]interface{}) bool {
			if blockedIPs, ok := data["blocked_ips_count"].(int); ok {
				return blockedIPs > 100 // Больше 100 заблокированных IP
			}
			return false
		},
	})
	
	// Правило: Высокий процент ботов
	am.AddRule(&AlertRule{
		ID:          "high_bot_percentage",
		Name:        "Высокий процент ботов",
		Description: "Обнаружен высокий процент bot трафика",
		Level:       AlertLevelWarning,
		Cooldown:    time.Minute * 5,
		Condition: func(data map[string]interface{}) bool {
			if botPercent, ok := data["bot_percentage"].(float64); ok {
				return botPercent > 30.0 // Больше 30% bot трафика
			}
			return false
		},
	})
	
	// Правило: Аномально быстрое решение капч
	am.AddRule(&AlertRule{
		ID:          "fast_captcha_solving",
		Name:        "Аномально быстрое решение капч",
		Description: "Обнаружено подозрительно быстрое решение капч",
		Level:       AlertLevelWarning,
		Cooldown:    time.Minute * 3,
		Condition: func(data map[string]interface{}) bool {
			if avgTime, ok := data["avg_solve_time_ms"].(float64); ok {
				return avgTime < 1000 // Меньше 1 секунды на решение
			}
			return false
		},
	})
	
	// Правило: Высокий процент неудачных попыток
	am.AddRule(&AlertRule{
		ID:          "high_failure_rate",
		Name:        "Высокий процент неудачных попыток",
		Description: "Обнаружен высокий процент неудачных попыток решения капч",
		Level:       AlertLevelInfo,
		Cooldown:    time.Minute * 5,
		Condition: func(data map[string]interface{}) bool {
			if failureRate, ok := data["failure_rate"].(float64); ok {
				return failureRate > 80.0 // Больше 80% неудачных попыток
			}
			return false
		},
	})
	
	// Правило: Подозрительные географические паттерны
	am.AddRule(&AlertRule{
		ID:          "suspicious_geo_pattern",
		Name:        "Подозрительные географические паттерны",
		Description: "Обнаружены подозрительные географические паттерны запросов",
		Level:       AlertLevelWarning,
		Cooldown:    time.Minute * 10,
		Condition: func(data map[string]interface{}) bool {
			if countries, ok := data["unique_countries"].(int); ok {
				if totalRequests, ok := data["total_requests"].(int); ok {
					// Если запросы приходят из очень многих стран при небольшом общем количестве
					return countries > 50 && totalRequests < 1000
				}
			}
			return false
		},
	})
}

// LogChannel - канал для логирования алертов
type LogChannel struct {
	name string
}

// NewLogChannel создает новый канал логирования
func NewLogChannel() *LogChannel {
	return &LogChannel{name: "log"}
}

func (lc *LogChannel) SendAlert(ctx context.Context, alert Alert) error {
	alertJSON, _ := json.MarshalIndent(alert, "", "  ")
	log.Printf("🚨 SECURITY ALERT [%s]: %s\n%s", alert.Level, alert.Title, string(alertJSON))
	return nil
}

func (lc *LogChannel) GetName() string {
	return lc.name
}

// WebhookChannel - канал для отправки алертов через webhook
type WebhookChannel struct {
	name string
	url  string
}

// NewWebhookChannel создает новый webhook канал
func NewWebhookChannel(name, url string) *WebhookChannel {
	return &WebhookChannel{
		name: name,
		url:  url,
	}
}

func (wc *WebhookChannel) SendAlert(ctx context.Context, alert Alert) error {
	// В реальной реализации здесь был бы HTTP запрос к webhook URL
	log.Printf("📡 Webhook [%s]: Отправка алерта %s на %s", wc.name, alert.ID, wc.url)
	return nil
}

func (wc *WebhookChannel) GetName() string {
	return wc.name
}

