package config

import "github.com/spf13/viper"

type MqttConfig struct {
	Mqtt Config `yaml:"mqtt"`
}

type Config struct {
	Broker   string `yaml:"broker"`
	Interval int    `yaml:"interval"`
	Topics   Topics `yaml:"topics"`
}

type Topics struct {
	Topic1 string `yaml:"topic1"`
	Topic2 string `yaml:"topic2"`
	Topic3 string `yaml:"topic3"`
}

func LoadConfig() (*MqttConfig, error) {
	viper.SetConfigFile("./config.yml")

	err := viper.ReadInConfig()
	if err != nil {
		return nil, err
	}

	var config MqttConfig
	err = viper.Unmarshal(&config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
