package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	DBSource                 string `mapstructure:"DB_SOURCE"`
	ServiceName              string `mapstructure:"SERVICE_NAME"`
	CORSAllowedOrigins       string `mapstructure:"CORS_ALLOWED_ORIGINS"`
	OTELExporterOTLPEndpoint string `mapstructure:"OTEL_EXPORTER_OTLP_ENDPOINT"`
	OTELExporterOTLPHeaders  string `mapstructure:"OTEL_EXPORTER_OTLP_HEADERS"`
	OTELResourceAttributes   string `mapstructure:"OTEL_RESOURCE_ATTRIBUTES"`
}

// CORSAllowOriginList parses CORS_ALLOWED_ORIGINS as a comma-separated list.
// Defaults to http://localhost:8000 when unset or empty.
func (c Config) CORSAllowOriginList() []string {
	s := strings.TrimSpace(c.CORSAllowedOrigins)
	if s == "" {
		return []string{"http://localhost:8080"}
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return []string{"http://localhost:8080"}
	}
	return out
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
	_ = viper.BindEnv("CORS_ALLOWED_ORIGINS")

	// Unmarshal into config struct (from env vars and/or config file)
	err = viper.Unmarshal(&config)
	if err != nil {
		return config, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return config, nil
}
