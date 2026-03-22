package engine

import (
	"fmt"

	"github.com/spf13/viper"
)

func Parse() {
	viper.SetConfigName("example_azure")

	// Add search paths to find the file
	viper.AddConfigPath("example_yaml")

	// Find and read the config file
	err := viper.ReadInConfig()

	// Handle errors
	if err != nil {
		panic(fmt.Errorf("fatal error config file: %w", err))
	}

	fmt.Println("No error")
}
