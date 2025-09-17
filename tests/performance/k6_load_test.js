import http from 'k6/http'
import { check, sleep } from 'k6'
import { Rate, Trend } from 'k6/metrics'

// Custom metrics
export let errorRate = new Rate('errors')
export let captchaGenerationTime = new Trend('captcha_generation_time')
export let securityCheckTime = new Trend('security_check_time')

// Test configuration
export let options = {
	stages: [
		{ duration: '30s', target: 10 }, // Ramp up
		{ duration: '1m', target: 50 }, // Stay at 50 users
		{ duration: '2m', target: 100 }, // Ramp to 100 users
		{ duration: '3m', target: 200 }, // Ramp to 200 users
		{ duration: '2m', target: 500 }, // Ramp to 500 users (stress test)
		{ duration: '1m', target: 1000 }, // Peak load test
		{ duration: '2m', target: 1000 }, // Stay at peak
		{ duration: '2m', target: 0 }, // Ramp down
	],
	thresholds: {
		http_req_duration: ['p(95)<2000'], // 95% of requests must complete below 2s
		http_req_failed: ['rate<0.1'], // Error rate must be below 10%
		errors: ['rate<0.05'], // Custom error rate below 5%
		captcha_generation_time: ['p(95)<1000'], // 95% of captcha generation < 1s
		security_check_time: ['p(95)<100'], // 95% of security checks < 100ms
	},
}

const BASE_URL = 'http://localhost:8080'
const CAPTCHA_TYPES = ['drag_drop', 'click', 'swipe']
const COMPLEXITY_LEVELS = [30, 50, 70, 90]

// Bot user agents for testing security
const BOT_USER_AGENTS = [
	'bot/crawler/1.0',
	'Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)',
	'Mozilla/5.0 (compatible; bingbot/2.0; +http://www.bing.com/bingbot.htm)',
	'curl/7.68.0',
	'python-requests/2.25.1',
	'go-http-client/1.1',
]

const NORMAL_USER_AGENTS = [
	'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36',
	'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36',
	'Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36',
]

export default function () {
	const testType = Math.random()

	if (testType < 0.7) {
		// 70% normal captcha generation requests
		testNormalCaptchaGeneration()
	} else if (testType < 0.9) {
		// 20% bot-like requests (for security testing)
		testBotLikeRequests()
	} else {
		// 10% stress requests (rapid fire)
		testStressRequests()
	}

	sleep(Math.random() * 2) // Random sleep between 0-2 seconds
}

function testNormalCaptchaGeneration() {
	const captchaType =
		CAPTCHA_TYPES[Math.floor(Math.random() * CAPTCHA_TYPES.length)]
	const complexity =
		COMPLEXITY_LEVELS[Math.floor(Math.random() * COMPLEXITY_LEVELS.length)]
	const userAgent =
		NORMAL_USER_AGENTS[Math.floor(Math.random() * NORMAL_USER_AGENTS.length)]

	const startTime = Date.now()

	const payload = JSON.stringify({
		type: captchaType,
		complexity: complexity,
		client_id: `client_${Math.random().toString(36).substr(2, 9)}`,
	})

	const params = {
		headers: {
			'Content-Type': 'application/json',
			'User-Agent': userAgent,
		},
	}

	const response = http.post(
		`${BASE_URL}/api/v1/captcha/generate`,
		payload,
		params
	)

	const duration = Date.now() - startTime
	captchaGenerationTime.add(duration)

	const success = check(response, {
		'status is 200': r => r.status === 200,
		'response time < 2s': r => r.timings.duration < 2000,
		'has challenge ID': r => {
			try {
				const data = JSON.parse(r.body)
				return data.challenge_id && data.challenge_id.length > 0
			} catch (e) {
				return false
			}
		},
		'has HTML content': r => {
			try {
				const data = JSON.parse(r.body)
				return data.html && data.html.length > 100
			} catch (e) {
				return false
			}
		},
	})

	errorRate.add(!success)

	if (!success) {
		console.log(
			`Failed captcha generation: ${response.status} - ${response.body}`
		)
	}
}

function testBotLikeRequests() {
	const userAgent =
		BOT_USER_AGENTS[Math.floor(Math.random() * BOT_USER_AGENTS.length)]
	const suspiciousPaths = [
		'/admin',
		'/config',
		'/.env',
		'/wp-admin',
		'/phpmyadmin',
	]
	const path =
		suspiciousPaths[Math.floor(Math.random() * suspiciousPaths.length)]

	const startTime = Date.now()

	const params = {
		headers: {
			'User-Agent': userAgent,
		},
	}

	const response = http.get(`${BASE_URL}${path}`, params)

	const duration = Date.now() - startTime
	securityCheckTime.add(duration)

	// Bot requests should be blocked or rate limited
	const success = check(response, {
		'bot request handled': r =>
			r.status === 403 || r.status === 429 || r.status === 200,
		'security check time < 100ms': r => duration < 100,
	})

	errorRate.add(!success)

	if (response.status === 200) {
		console.log(`Bot request not blocked: ${userAgent} -> ${path}`)
	}
}

function testStressRequests() {
	// Rapid fire requests to test rate limiting
	const requests = Math.floor(Math.random() * 10) + 5 // 5-15 rapid requests

	for (let i = 0; i < requests; i++) {
		const userAgent =
			NORMAL_USER_AGENTS[Math.floor(Math.random() * NORMAL_USER_AGENTS.length)]
		const captchaType =
			CAPTCHA_TYPES[Math.floor(Math.random() * CAPTCHA_TYPES.length)]

		const payload = JSON.stringify({
			type: captchaType,
			complexity: 50,
			client_id: `stress_client_${Math.random().toString(36).substr(2, 9)}`,
		})

		const params = {
			headers: {
				'Content-Type': 'application/json',
				'User-Agent': userAgent,
			},
		}

		const response = http.post(
			`${BASE_URL}/api/v1/captcha/generate`,
			payload,
			params
		)

		check(response, {
			'stress request handled': r => r.status === 200 || r.status === 429,
		})

		// Very short sleep between rapid requests
		sleep(0.01)
	}
}

export function handleSummary(data) {
	return {
		'load_test_results.json': JSON.stringify(data, null, 2),
		'load_test_summary.txt': `
Load Test Summary
=================

Test Duration: ${data.state.testRunDurationMs / 1000}s
Total Requests: ${data.metrics.http_reqs.values.count}
Failed Requests: ${data.metrics.http_req_failed.values.count}
Error Rate: ${(data.metrics.http_req_failed.values.rate * 100).toFixed(2)}%

Performance Metrics:
- Average Response Time: ${data.metrics.http_req_duration.values.avg.toFixed(
			2
		)}ms
- 95th Percentile: ${data.metrics.http_req_duration.values['p(95)'].toFixed(
			2
		)}ms
- 99th Percentile: ${data.metrics.http_req_duration.values['p(99)'].toFixed(
			2
		)}ms

Captcha Generation:
- Average Time: ${data.metrics.captcha_generation_time.values.avg.toFixed(2)}ms
- 95th Percentile: ${data.metrics.captcha_generation_time.values[
			'p(95)'
		].toFixed(2)}ms

Security Checks:
- Average Time: ${data.metrics.security_check_time.values.avg.toFixed(2)}ms
- 95th Percentile: ${data.metrics.security_check_time.values['p(95)'].toFixed(
			2
		)}ms

Thresholds Status:
${Object.entries(data.thresholds)
	.map(([name, result]) => `- ${name}: ${result.ok ? 'PASS' : 'FAIL'}`)
	.join('\n')}

Recommendations:
${generateRecommendations(data)}
`,
	}
}

function generateRecommendations(data) {
	const recommendations = []

	if (data.metrics.http_req_failed.values.rate > 0.05) {
		recommendations.push(
			'- High error rate detected. Consider increasing server capacity or optimizing code.'
		)
	}

	if (data.metrics.http_req_duration.values['p(95)'] > 1000) {
		recommendations.push(
			'- 95th percentile response time is high. Consider performance optimizations.'
		)
	}

	if (data.metrics.captcha_generation_time.values.avg > 500) {
		recommendations.push(
			'- Captcha generation is slow. Consider caching or algorithm optimization.'
		)
	}

	if (data.metrics.security_check_time.values.avg > 50) {
		recommendations.push(
			'- Security checks are slow. Consider optimizing security algorithms.'
		)
	}

	const maxRPS =
		data.metrics.http_reqs.values.count / (data.state.testRunDurationMs / 1000)
	if (maxRPS < 100) {
		recommendations.push(
			'- Low RPS achieved. Consider horizontal scaling or performance tuning.'
		)
	}

	return recommendations.length > 0
		? recommendations.join('\n')
		: '- All performance metrics are within acceptable ranges.'
}
