package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/middelmatigheid/subscriptions-api/internal/models"
)

type Config struct {
	Port       string
	DBUser     string
	DBPassword string
	DBName     string
	DBHost     string
	DBPort     string

	RedisHost     string
	RedisPort     string
	RedisPassword string
	RedisDB       int
	RedisTTL      int
}

func GetConfig() (*Config, error) {
	err := godotenv.Load()
	if err != nil {
		return nil, models.NewErrInternalServer(err)
	}

	redisDB, err := strconv.Atoi(os.Getenv("REDIS_DB"))
	if err != nil {
		return nil, models.NewErrInternalServer(err)
	}
	redisTTL, err := strconv.Atoi(os.Getenv("REDIS_TTL"))
	if err != nil {
		return nil, models.NewErrInternalServer(err)
	}

	return &Config{Port: os.Getenv("PORT"), DBUser: os.Getenv("DB_USER"), DBPassword: os.Getenv("DB_PASSWORD"), DBName: os.Getenv("DB_NAME"),
		DBHost: os.Getenv("DB_HOST"), DBPort: os.Getenv("DB_PORT"), RedisHost: os.Getenv("REDIS_HOST"), RedisPort: os.Getenv("REDIS_PORT"),
		RedisPassword: os.Getenv("REDIS_PASSWORD"), RedisDB: redisDB, RedisTTL: redisTTL}, nil
}
