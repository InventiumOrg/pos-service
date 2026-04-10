package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	DBSource                 string `mapstructure:"DB_SOURCE"`
	ServiceName              string `mapstructure:"SERVICE_NAME"`
	OTELExporterOTLPEndpoint string `mapstructure:"OTEL_EXPORTER_OTLP_ENDPOINT"`
	OTELExporterOTLPHeaders  string `mapstructure:"OTEL_EXPORTER_OTLP_HEADERS"`
	OTELResourceAttributes   string `mapstructure:"OTEL_RESOURCE_ATTRIBUTES"`
}

func LoadConfig(path string) (config Config, err error) {
	// Bind all environment variables
	viper.AutomaticEnv()

	// Explicitly bind each config key to its environment variable
	_ = viper.BindEnv("SERVICE_NAME")
	_ = viper.BindEnv("OTEL_EXPORTER_OTLP_ENDPOINT")
	_ = viper.BindEnv("OTEL_EXPORTER_OTLP_HEADERS")
	_ = viper.BindEnv("OTEL_RESOURCE_ATTRIBUTES")
	_ = viper.BindEnv("DB_SOURCE")

	// Unmarshal into config struct (from env vars and/or config file)
	err = viper.Unmarshal(&config)
	if err != nil {
		return config, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return config, nil
}
