package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "feed-generator",
	Short: "Generate realistic trade feeds for testing",
	Long: `A sophisticated trade feed generator that produces realistic 
trading patterns with configurable fraud injection for testing 
the trade detection system.

The generator simulates three types of traders:
  - High-Frequency Traders (20% of users, 80% of volume)
  - Regular Traders (70% of users, 18% of volume)
  - Casual Traders (10% of users, 2% of volume)

It can inject various fraud patterns including wash trades,
velocity spikes, and anomalies for testing detection algorithms.`,
	Version: "1.0.0",
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "",
		"config file (default is .feed-generator.yaml)")
	rootCmd.PersistentFlags().String("redis-host", "localhost",
		"Redis host")
	rootCmd.PersistentFlags().Int("redis-port", 6379,
		"Redis port")
	rootCmd.PersistentFlags().String("redis-password", "",
		"Redis password")

	// Bind flags to viper
	viper.BindPFlag("redis.host", rootCmd.PersistentFlags().Lookup("redis-host"))
	viper.BindPFlag("redis.port", rootCmd.PersistentFlags().Lookup("redis-port"))
	viper.BindPFlag("redis.password", rootCmd.PersistentFlags().Lookup("redis-password"))
}

func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag
		viper.SetConfigFile(cfgFile)
	} else {
		// Search for config in current directory and home directory
		viper.AddConfigPath(".")
		viper.AddConfigPath("$HOME")
		viper.SetConfigName(".feed-generator")
		viper.SetConfigType("yaml")
	}

	// Environment variables
	viper.SetEnvPrefix("FEED_GEN")
	viper.AutomaticEnv()

	// Read config file
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
