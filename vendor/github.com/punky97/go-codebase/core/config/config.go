package config

import (
	"bytes"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/hashicorp/consul/api"
	ioutils "github.com/punky97/go-codebase/core/utils/io"
	"github.com/spf13/viper"
)

// ReadBkConfig read the config file regard to the file type (file extension)
// Also, viper can watch changes on the file --> allow us to hot reload the application
func ReadBkConfig(fileName string, configPaths ...string) bool {
	viper.SetConfigName(fileName)
	if len(configPaths) < 1 {
		// look for current dir
		viper.AddConfigPath(".")
	} else {
		for _, configPath := range configPaths {
			viper.AddConfigPath(configPath)
		}
	}
	err := viper.ReadInConfig()
	if err != nil {
		fmt.Printf("Cannot read config file. %s", err)
		return false
	}

	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		fmt.Println("Config file changed:", e.Name)
	})

	return true
}

// ReadBkConfigByFile read config file by file path
func ReadBkConfigByFile(file string) bool {
	viper.SetConfigFile(file)
	err := viper.ReadInConfig()
	if err != nil {
		fmt.Printf("Cannot read config file. %s", err)
		return false
	}

	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		fmt.Println("Config file changed:", e.Name)
	})

	return true
}

// LoadConfig -- read config from byte
func LoadConfig(configType string, value []byte, isMerge bool) (err error) {
	viper.SetConfigType(configType)
	if !isMerge {
		err = viper.ReadConfig(bytes.NewBuffer(value))
	} else {
		err = viper.MergeConfig(bytes.NewBuffer(value))
	}
	return
}

// GetFromConsulKV -- get value of key in consul KV
func GetFromConsulKV(endpoint, key string) (value []byte, err error) {
	// init config connection to consul
	config := api.DefaultConfig()
	if endpoint != "" {
		config.Address = endpoint
	}

	// init consul client
	client, err := api.NewClient(config)
	if err != nil {
		return
	}
	kv := client.KV()

	// get key
	pair, _, err := kv.Get(key, nil)
	if err != nil {
		return
	}
	if pair == nil {
		err = fmt.Errorf("remote config key is not existed: %v", key)
		return
	}

	value = pair.Value
	return
}

// ReadRawFile --
func ReadRawFile(file string) (string, error) {
	return ioutils.ReadRawFile(file)
}
