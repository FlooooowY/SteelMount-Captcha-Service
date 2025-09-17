package security

import (
	"context"
	"math"
	"strings"
	"sync"
	"time"
)

// UserBehavior представляет поведение пользователя
type UserBehavior struct {
	IP                  string
	FirstSeen           time.Time
	LastSeen            time.Time
	TotalRequests       int64
	SuccessfulRequests  int64
	FailedRequests      int64
	AverageResponseTime time.Duration
	CaptchaSolveTime    []time.Duration
	UserAgents          map[string]int
	RequestPaths        map[string]int
	RequestTimes        []time.Time
	TrustScore          float64 // 0.0 (не доверяем) до 1.0 (полное доверие)

	// Паттерны поведения
	IsRegularUser bool
	IsSuspicious  bool
	IsBot         bool
}

// AdaptiveLimiter реализует адаптивные лимиты на основе поведения пользователя
type AdaptiveLimiter struct {
	behaviors    map[string]*UserBehavior
	baseLimits   map[string]int // Базовые лимиты для разных типов пользователей
	mutex        sync.RWMutex
	alertManager AlertManager // Интерфейс для отправки алертов

	// Конфигурация
	config *AdaptiveLimiterConfig
}

// AdaptiveLimiterConfig конфигурация адаптивного лимитера
type AdaptiveLimiterConfig struct {
	// Базовые лимиты
	BaseRPMLimit          int     // Базовый лимит запросов в минуту
	TrustedUserMultiplier float64 // Множитель для доверенных пользователей
	SuspiciousUserDivisor float64 // Делитель для подозрительных пользователей
	BotUserLimit          int     // Жесткий лимит для ботов

	// Параметры анализа поведения
	MinRequestsForAnalysis int           // Минимум запросов для анализа
	TrustScoreDecayRate    float64       // Скорость снижения доверия
	BehaviorAnalysisWindow time.Duration // Окно анализа поведения

	// Пороги для классификации
	TrustedUserThreshold    float64 // Порог для доверенных пользователей
	SuspiciousUserThreshold float64 // Порог для подозрительных пользователей
	BotDetectionThreshold   float64 // Порог для детекции ботов
}

// AlertManager интерфейс для отправки алертов
type AlertManager interface {
	ProcessEvent(source string, data map[string]interface{})
}

// NewAdaptiveLimiter создает новый адаптивный лимитер
func NewAdaptiveLimiter(config *AdaptiveLimiterConfig, alertManager AlertManager) *AdaptiveLimiter {
	if config == nil {
		config = &AdaptiveLimiterConfig{
			BaseRPMLimit:            60,
			TrustedUserMultiplier:   2.0,
			SuspiciousUserDivisor:   4.0,
			BotUserLimit:            5,
			MinRequestsForAnalysis:  10,
			TrustScoreDecayRate:     0.95,
			BehaviorAnalysisWindow:  time.Hour * 24,
			TrustedUserThreshold:    0.8,
			SuspiciousUserThreshold: 0.3,
			BotDetectionThreshold:   0.1,
		}
	}

	al := &AdaptiveLimiter{
		behaviors:    make(map[string]*UserBehavior),
		baseLimits:   make(map[string]int),
		config:       config,
		alertManager: alertManager,
	}

	// Инициализируем базовые лимиты
	al.baseLimits["trusted"] = int(float64(config.BaseRPMLimit) * config.TrustedUserMultiplier)
	al.baseLimits["normal"] = config.BaseRPMLimit
	al.baseLimits["suspicious"] = int(float64(config.BaseRPMLimit) / config.SuspiciousUserDivisor)
	al.baseLimits["bot"] = config.BotUserLimit

	return al
}

// AnalyzeRequest анализирует запрос и обновляет поведение пользователя
func (al *AdaptiveLimiter) AnalyzeRequest(ctx context.Context, ip, userAgent, path string,
	responseTime time.Duration, isSuccess bool) *UserBehavior {

	al.mutex.Lock()
	defer al.mutex.Unlock()

	now := time.Now()

	// Получаем или создаем поведение пользователя
	behavior, exists := al.behaviors[ip]
	if !exists {
		behavior = &UserBehavior{
			IP:               ip,
			FirstSeen:        now,
			LastSeen:         now,
			UserAgents:       make(map[string]int),
			RequestPaths:     make(map[string]int),
			RequestTimes:     make([]time.Time, 0),
			CaptchaSolveTime: make([]time.Duration, 0),
			TrustScore:       0.5, // Начинаем с нейтрального доверия
		}
		al.behaviors[ip] = behavior
	}

	// Обновляем статистику
	behavior.LastSeen = now
	behavior.TotalRequests++
	behavior.AverageResponseTime = (behavior.AverageResponseTime*time.Duration(behavior.TotalRequests-1) + responseTime) / time.Duration(behavior.TotalRequests)

	if isSuccess {
		behavior.SuccessfulRequests++
	} else {
		behavior.FailedRequests++
	}

	// Обновляем паттерны
	behavior.UserAgents[userAgent]++
	behavior.RequestPaths[path]++
	behavior.RequestTimes = append(behavior.RequestTimes, now)

	// Ограничиваем размер истории
	if len(behavior.RequestTimes) > 1000 {
		behavior.RequestTimes = behavior.RequestTimes[len(behavior.RequestTimes)-1000:]
	}

	// Анализируем поведение если достаточно данных
	if behavior.TotalRequests >= int64(al.config.MinRequestsForAnalysis) {
		al.analyzeBehaviorPatterns(behavior)
	}

	return behavior
}

// GetAdaptiveLimit возвращает адаптивный лимит для пользователя
func (al *AdaptiveLimiter) GetAdaptiveLimit(ip string) int {
	al.mutex.RLock()
	defer al.mutex.RUnlock()

	behavior, exists := al.behaviors[ip]
	if !exists {
		return al.baseLimits["normal"] // Возвращаем базовый лимит для новых пользователей
	}

	// Определяем категорию пользователя на основе доверия
	switch {
	case behavior.IsBot:
		return al.baseLimits["bot"]
	case behavior.TrustScore >= al.config.TrustedUserThreshold:
		return al.baseLimits["trusted"]
	case behavior.TrustScore <= al.config.SuspiciousUserThreshold:
		return al.baseLimits["suspicious"]
	default:
		return al.baseLimits["normal"]
	}
}

// analyzeBehaviorPatterns анализирует паттерны поведения пользователя
func (al *AdaptiveLimiter) analyzeBehaviorPatterns(behavior *UserBehavior) {
	now := time.Now()

	// Анализируем временные паттерны
	timeScore := al.analyzeTimePatterns(behavior, now)

	// Анализируем разнообразие User-Agent
	uaScore := al.analyzeUserAgentPatterns(behavior)

	// Анализируем успешность запросов
	successScore := al.analyzeSuccessRate(behavior)

	// Анализируем скорость запросов
	rateScore := al.analyzeRequestRate(behavior, now)

	// Анализируем время решения капч
	solveScore := al.analyzeCaptcheSolveTime(behavior)

	// Вычисляем общий счет доверия
	oldTrustScore := behavior.TrustScore
	newTrustScore := (timeScore + uaScore + successScore + rateScore + solveScore) / 5.0

	// Применяем сглаживание и затухание
	behavior.TrustScore = oldTrustScore*al.config.TrustScoreDecayRate +
		newTrustScore*(1-al.config.TrustScoreDecayRate)

	// Классифицируем пользователя
	al.classifyUser(behavior)

	// Отправляем алерт при изменении классификации
	if behavior.TrustScore < al.config.SuspiciousUserThreshold && oldTrustScore >= al.config.SuspiciousUserThreshold {
		al.sendSuspiciousUserAlert(behavior)
	}
}

// analyzeTimePatterns анализирует временные паттерны запросов
func (al *AdaptiveLimiter) analyzeTimePatterns(behavior *UserBehavior, now time.Time) float64 {
	if len(behavior.RequestTimes) < 5 {
		return 0.5 // Недостаточно данных
	}

	// Анализируем регулярность запросов
	intervals := make([]time.Duration, 0, len(behavior.RequestTimes)-1)
	for i := 1; i < len(behavior.RequestTimes); i++ {
		intervals = append(intervals, behavior.RequestTimes[i].Sub(behavior.RequestTimes[i-1]))
	}

	// Вычисляем стандартное отклонение интервалов
	mean := time.Duration(0)
	for _, interval := range intervals {
		mean += interval
	}
	mean /= time.Duration(len(intervals))

	variance := time.Duration(0)
	for _, interval := range intervals {
		diff := interval - mean
		variance += time.Duration(int64(diff) * int64(diff))
	}
	variance /= time.Duration(len(intervals))

	stdDev := time.Duration(math.Sqrt(float64(variance)))

	// Слишком регулярные запросы подозрительны (боты)
	// Слишком нерегулярные тоже могут быть подозрительными
	regularityScore := 1.0 - math.Min(float64(stdDev)/float64(mean), 1.0)

	// Человекоподобная нерегулярность получает высокий счет
	if regularityScore > 0.3 && regularityScore < 0.8 {
		return 0.8
	}

	return regularityScore * 0.5
}

// analyzeUserAgentPatterns анализирует паттерны User-Agent
func (al *AdaptiveLimiter) analyzeUserAgentPatterns(behavior *UserBehavior) float64 {
	uaCount := len(behavior.UserAgents)

	// Один стабильный UA - хорошо
	if uaCount == 1 {
		// Проверяем, не является ли это bot UA
		for ua := range behavior.UserAgents {
			if al.isBotUserAgent(ua) {
				return 0.1 // Это бот
			}
		}
		return 0.9 // Стабильный человеческий UA
	}

	// Слишком много разных UA - подозрительно
	if uaCount > 5 {
		return 0.2
	}

	// 2-3 разных UA может быть нормально (разные браузеры/устройства)
	return 0.7
}

// analyzeSuccessRate анализирует успешность запросов
func (al *AdaptiveLimiter) analyzeSuccessRate(behavior *UserBehavior) float64 {
	if behavior.TotalRequests == 0 {
		return 0.5
	}

	successRate := float64(behavior.SuccessfulRequests) / float64(behavior.TotalRequests)

	// Слишком высокая успешность может указывать на бота
	// Слишком низкая - на атаку
	switch {
	case successRate > 0.95:
		return 0.3 // Подозрительно высокая успешность
	case successRate > 0.7:
		return 0.9 // Нормальная успешность
	case successRate > 0.3:
		return 0.6 // Средняя успешность
	default:
		return 0.2 // Низкая успешность - возможная атака
	}
}

// analyzeRequestRate анализирует скорость запросов
func (al *AdaptiveLimiter) analyzeRequestRate(behavior *UserBehavior, now time.Time) float64 {
	if len(behavior.RequestTimes) < 2 {
		return 0.5
	}

	// Анализируем RPS за последние 5 минут
	cutoff := now.Add(-5 * time.Minute)
	recentRequests := 0

	for _, requestTime := range behavior.RequestTimes {
		if requestTime.After(cutoff) {
			recentRequests++
		}
	}

	rps := float64(recentRequests) / 300.0 // 5 минут = 300 секунд

	// Отправляем алерт если RPS слишком высокий
	if rps > 10 && al.alertManager != nil {
		al.alertManager.ProcessEvent("adaptive_limiter", map[string]interface{}{
			"event":       "high_rps_detected",
			"ip":          behavior.IP,
			"rps":         rps,
			"trust_score": behavior.TrustScore,
		})
	}

	// Оцениваем нормальность RPS
	switch {
	case rps > 5:
		return 0.1 // Очень высокий RPS - вероятно бот
	case rps > 2:
		return 0.3 // Высокий RPS - подозрительно
	case rps > 0.1:
		return 0.8 // Нормальный RPS
	default:
		return 0.6 // Низкий RPS - может быть нормально
	}
}

// analyzeCaptcheSolveTime анализирует время решения капч
func (al *AdaptiveLimiter) analyzeCaptcheSolveTime(behavior *UserBehavior) float64 {
	if len(behavior.CaptchaSolveTime) == 0 {
		return 0.5 // Нет данных о решении капч
	}

	// Вычисляем среднее время решения
	totalTime := time.Duration(0)
	for _, solveTime := range behavior.CaptchaSolveTime {
		totalTime += solveTime
	}
	avgTime := totalTime / time.Duration(len(behavior.CaptchaSolveTime))

	// Анализируем время
	avgMs := avgTime.Milliseconds()

	switch {
	case avgMs < 1000:
		return 0.1 // Слишком быстро - вероятно бот
	case avgMs < 3000:
		return 0.4 // Быстро - подозрительно
	case avgMs < 10000:
		return 0.9 // Нормальное время
	case avgMs < 30000:
		return 0.7 // Медленно, но приемлемо
	default:
		return 0.3 // Слишком медленно - может быть подозрительно
	}
}

// classifyUser классифицирует пользователя на основе счета доверия
func (al *AdaptiveLimiter) classifyUser(behavior *UserBehavior) {
	// Сбрасываем предыдущую классификацию
	behavior.IsRegularUser = false
	behavior.IsSuspicious = false
	behavior.IsBot = false

	switch {
	case behavior.TrustScore >= al.config.TrustedUserThreshold:
		behavior.IsRegularUser = true
	case behavior.TrustScore <= al.config.BotDetectionThreshold:
		behavior.IsBot = true
	case behavior.TrustScore <= al.config.SuspiciousUserThreshold:
		behavior.IsSuspicious = true
	}
}

// isBotUserAgent проверяет, является ли User-Agent ботом
func (al *AdaptiveLimiter) isBotUserAgent(userAgent string) bool {
	botPatterns := []string{
		"bot", "crawler", "spider", "scraper", "headless",
		"phantom", "selenium", "webdriver", "automated",
	}

	userAgentLower := strings.ToLower(userAgent)
	for _, pattern := range botPatterns {
		if strings.Contains(userAgentLower, pattern) {
			return true
		}
	}

	return false
}

// sendSuspiciousUserAlert отправляет алерт о подозрительном пользователе
func (al *AdaptiveLimiter) sendSuspiciousUserAlert(behavior *UserBehavior) {
	if al.alertManager != nil {
		al.alertManager.ProcessEvent("adaptive_limiter", map[string]interface{}{
			"event":          "suspicious_user_detected",
			"ip":             behavior.IP,
			"trust_score":    behavior.TrustScore,
			"total_requests": behavior.TotalRequests,
			"success_rate":   float64(behavior.SuccessfulRequests) / float64(behavior.TotalRequests),
			"user_agents":    len(behavior.UserAgents),
			"is_bot":         behavior.IsBot,
			"is_suspicious":  behavior.IsSuspicious,
		})
	}
}

// GetUserBehavior возвращает поведение пользователя
func (al *AdaptiveLimiter) GetUserBehavior(ip string) *UserBehavior {
	al.mutex.RLock()
	defer al.mutex.RUnlock()

	if behavior, exists := al.behaviors[ip]; exists {
		// Возвращаем копию чтобы избежать race conditions
		behaviorCopy := *behavior
		return &behaviorCopy
	}

	return nil
}

// GetStats возвращает статистику адаптивного лимитера
func (al *AdaptiveLimiter) GetStats() map[string]interface{} {
	al.mutex.RLock()
	defer al.mutex.RUnlock()

	totalUsers := len(al.behaviors)
	trustedUsers := 0
	suspiciousUsers := 0
	botUsers := 0

	for _, behavior := range al.behaviors {
		if behavior.IsRegularUser {
			trustedUsers++
		} else if behavior.IsBot {
			botUsers++
		} else if behavior.IsSuspicious {
			suspiciousUsers++
		}
	}

	return map[string]interface{}{
		"total_users":      totalUsers,
		"trusted_users":    trustedUsers,
		"suspicious_users": suspiciousUsers,
		"bot_users":        botUsers,
		"base_limits":      al.baseLimits,
	}
}

// CleanupExpiredBehaviors очищает устаревшие данные о поведении
func (al *AdaptiveLimiter) CleanupExpiredBehaviors() {
	al.mutex.Lock()
	defer al.mutex.Unlock()

	now := time.Now()
	cutoff := now.Add(-al.config.BehaviorAnalysisWindow)

	for ip, behavior := range al.behaviors {
		if behavior.LastSeen.Before(cutoff) {
			delete(al.behaviors, ip)
		}
	}
}
