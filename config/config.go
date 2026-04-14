package config

import (
	"log"

	"github.com/Netflix/go-env"
)

type Config struct {
	Database struct {
		Host     string `env:"POSTGRES_HOST,required=true"`
		Port     int    `env:"POSTGRES_PORT,default=5432"`
		Username string `env:"POSTGRES_USERNAME,required=true"`
		Password string `env:"POSTGRES_PASSWORD,required=true"`
		Name     string `env:"POSTGRES_DATABASE_NAME,required=true"`
	}
	Log struct {
		Level string `env:"LOG_LEVEL,default=info"`
		Type  string `env:"LOG_TYPE,default=json"`
	}
}

func ParseConfig() Config {
	var config Config
	if _, err := env.UnmarshalFromEnviron(&config); err != nil {
		log.Fatal(err)
	}
	return config
}
