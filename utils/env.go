package utils

import (
	"log"
	"os"
)

/**
 * Gets the value of an environment value and if it is missing returns the
 * default value.
 */
func GetEnvOrDefault(envvar string, defaultValue string, ensure bool) string {
	out := os.Getenv(envvar)
	if out == "" {
		out = defaultValue
	}
	if ensure && out == "" {
		log.Fatalf("Env variable %s not found and deafult not given", envvar)
	}
	return out
}

func EnsureEnvOrDefault(currval string, envvar string, defaultValue string) string {
	if currval == "" {
		currval = GetEnvOrDefault(envvar, defaultValue, true)
	}
	return currval
}
