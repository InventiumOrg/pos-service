package config

import "github.com/spf13/viper"

type Config struct {
	DBSource                 string `mapstructure:"DB_SOURCE"`
	ClerKKey                 string `mapstructure:"CLERK_KEY"`
	ServiceName              string `mapstructure:"SERVICE_NAME"`
	OTELExporterOTLPEndpoint string `mapstructure:"OTEL_EXPORTER_OTLP_ENDPOINT"`
	OTELExporterOTLPHeaders  string `mapstructure:"OTEL_EXPORTER_OTLP_HEADERS"`
	OTELResourceAttributes   string `mapstructure:"OTEL_RESOURCE_ATTRIBUTES"`
	OTELLogsExporter         string `mapstructure:"OTEL_LOGS_EXPORTER"`
	LokiURL                  string `mapstructure:"LOKI_URL"`
	SyslogAddress            string `mapstructure:"SYSLOG_ADDRESS"`
	SyslogNetwork            string `mapstructure:"SYSLOG_NETWORK"`
	LogFilePath              string `mapstructure:"LOG_FILE_PATH"`
}

func LoadConfig(path string) (config Config, err error) {
	viper.AddConfigPath(path)
	viper.SetConfigName("app")
	viper.SetConfigType("env")
	viper.AutomaticEnv()

	err = viper.ReadInConfig()
	if err != nil {
		return
	}
	err = viper.Unmarshal(&config)
	if err != nil {
		return
	}

	// Set defaults
	if config.ServiceName == "" {
		config.ServiceName = "pos-service"
	}

	return config, nil
}
