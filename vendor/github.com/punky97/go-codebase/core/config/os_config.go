package config

import "os"

// Specified for k8s env
func GetHostName() string {
	return os.Getenv("HOSTNAME")
}
