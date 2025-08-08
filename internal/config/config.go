package config

import (
	"log"

	"github.com/spf13/viper"
)

type Config struct {
	Port        string `mapstructure:"PORT"`
	JWTSecret   string `mapstructure:"JWT_SECRET"`
	MinioBucket string `mapstructure:"MINIO_BUCKET"`
	AccessKey   string `mapstructure:"ACCESS_KEY"`
	SecretKey   string `mapstructure:"SECRET_KEY"`
	MongoURI    string `mapstructure:"MONGO_URI"`
	MongoDBName string `mapstructure:"MONGO_DB_NAME"`

	RedisAddr string `mapstructure:"REDIS_ADDR"`
	RedisPass string `mapstructure:"REDIS_PASS"`

	KafkaBrokers []string `mapstructure:"KAFKA_BROKERS"`
	KafkaTopic   string   `mapstructure:"KAFKA_TOPIC"`

	KeycloakRealm string `mapstructure:"KEYCLOAK_REALM"`
	KeycloakURL   string `mapstructure:"KEYCLOAK_URL"`
}

func LoadConfig() (*Config, error) {
	viper.SetConfigFile("config/config.yaml")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("Couldn't load config.yaml: %v", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
