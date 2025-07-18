package config

import (
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

var (
	configMutex   sync.RWMutex
	currentConfig *AppConfig
)

// RouteConfig defines a route between source and target
type RouteConfig struct {
	VehicleID string       `mapstructure:"vehicle_id"`
	Imei      string       `mapstructure:"imei"`
	Source    string       `mapstructure:"source"` // "lat,lon"
	Target    string       `mapstructure:"target"` // "lat,lon"
	Stops     []StopConfig `mapstructure:"stops"`
}

type StopConfig struct {
	Location string `yaml:"location"`
	Duration int    `yaml:"duration"` // seconds
}

// SimulatorConfig contains frequency and concurrency
type SimulatorConfig struct {
	FrequencySeconds int    `mapstructure:"frequency_seconds"`
	ConcurrentRoutes int    `mapstructure:"concurrent_routes"`
	Client           string `mapstructure:"client"`
}

type BaseUrlConfig struct {
	BaseUrl string `mapstructure:"base_url"`
}

// AppConfig holds entire config
type AppConfig struct {
	Simulator SimulatorConfig `mapstructure:"simulator"`
	OSRM      BaseUrlConfig   `mapstructure:"osrm"`
	Routes    []RouteConfig   `mapstructure:"routes"`
}

// LoadConfig initializes and loads the configuration
func LoadConfig(path string) (*AppConfig, error) {
	viper.SetConfigFile(path)
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Explicitly set the config type if not using file extension
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var cfg AppConfig
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	configMutex.Lock()
	currentConfig = &cfg
	configMutex.Unlock()

	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		var newCfg AppConfig
		if err := viper.Unmarshal(&newCfg); err == nil {
			configMutex.Lock()
			currentConfig = &newCfg
			configMutex.Unlock()
		}
	})

	return &cfg, nil
}

// GetCurrentConfig returns the current configuration in a thread-safe way
func GetCurrentConfig() *AppConfig {
	configMutex.RLock()
	defer configMutex.RUnlock()
	return currentConfig
}
