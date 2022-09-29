package config

import (
	"log"
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
		log.Println(err)
	}
	return os.Getenv("SPANNER_URI")
}

func EnvAppPort() string {
	err := godotenv.Load()
	if err != nil {
		log.Println(err)
	}
	return os.Getenv("APP_PORT")
}
