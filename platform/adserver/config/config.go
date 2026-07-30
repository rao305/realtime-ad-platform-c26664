package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Addr              string
	RedisAddr         string
	KafkaBrokers      []string
	KafkaTopic        string
	DecisionBudget    time.Duration
	FreqCapPerHour    int64
	ReadyRequireRedis bool
}

func Load() (Config, error) {
	cfg := Config{
		Addr:              getEnv("ADSERVER_ADDR", ":8080"),
		RedisAddr:         getEnv("REDIS_ADDR", "redis:6379"),
		KafkaTopic:        getEnv("KAFKA_TOPIC", "ad-events"),
		DecisionBudget:    30 * time.Millisecond,
		FreqCapPerHour:    5,
		ReadyRequireRedis: true,
	}

	brokers := strings.TrimSpace(getEnv("KAFKA_BROKERS", "kafka:9092"))
	if brokers == "" {
		return Config{}, fmt.Errorf("KAFKA_BROKERS cannot be empty")
	}
	for _, broker := range strings.Split(brokers, ",") {
		broker = strings.TrimSpace(broker)
		if broker != "" {
			cfg.KafkaBrokers = append(cfg.KafkaBrokers, broker)
		}
	}
	if len(cfg.KafkaBrokers) == 0 {
		return Config{}, fmt.Errorf("KAFKA_BROKERS cannot be empty")
	}

	if raw := strings.TrimSpace(os.Getenv("DECISION_BUDGET_MS")); raw != "" {
		ms, err := strconv.Atoi(raw)
		if err != nil || ms <= 0 {
			return Config{}, fmt.Errorf("invalid DECISION_BUDGET_MS: %q", raw)
		}
		cfg.DecisionBudget = time.Duration(ms) * time.Millisecond
	}
	if raw := strings.TrimSpace(os.Getenv("FREQ_CAP_PER_HOUR")); raw != "" {
		cap, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || cap <= 0 {
			return Config{}, fmt.Errorf("invalid FREQ_CAP_PER_HOUR: %q", raw)
		}
		cfg.FreqCapPerHour = cap
	}
	if raw := strings.TrimSpace(os.Getenv("READY_REQUIRE_REDIS")); raw != "" {
		val, err := strconv.ParseBool(raw)
		if err != nil {
			return Config{}, fmt.Errorf("invalid READY_REQUIRE_REDIS: %q", raw)
		}
		cfg.ReadyRequireRedis = val
	}

	return cfg, nil
}

func getEnv(key, def string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return def
}
