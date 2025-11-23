package config

import "github.com/spf13/viper"

type Config struct {
	DBSource string `mapstructure:"DB_SOURCE"`
	ClerKKey string `mapstructure:"CLERK_KEY"`
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
	return config, nil
}
