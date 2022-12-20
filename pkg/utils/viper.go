package utils

import "github.com/spf13/viper"

func ViperGetInt64WithDefault(key string, defaultValue int64) int64 {
	v := viper.GetInt64(key)
	if v == 0 {
		return defaultValue
	}
	return v
}

func ViperGetIntWithDefault(key string, defaultValue int) int {
	v := viper.GetInt(key)
	if v == 0 {
		return defaultValue
	}
	return v
}
