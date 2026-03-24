package engine

import (
	"fmt"

	"github.com/spf13/viper"
)

func load() *viper.Viper {
	v := viper.New()
	v.SetConfigName("example_azure")
	v.AddConfigPath("example_yaml")

	err := v.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("fatal error config file: %w", err))
	}

	return v
}
