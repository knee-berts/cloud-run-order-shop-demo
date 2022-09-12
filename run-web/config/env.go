package config

import (
	"os"

	"github.com/joho/godotenv"
)

// type Config struct {
// 	Server  ServerConfig
// 	Spanner SpannerConfig
// }

func EnvSpannerURI() string {
	err := godotenv.Load()
	if err != nil {
		return os.Getenv("SPANNER_URI")
	}
	return os.Getenv("SPANNER_URI")
}

func EnvAppPort() string {
	err := godotenv.Load()
	if err != nil {
		return os.Getenv("APP_PORT")
	}
	return os.Getenv("APP_PORT")
}
