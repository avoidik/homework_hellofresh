package main

import (
	"fmt"
	"os"
	"strconv"
)

// getStringOrDefault allows to retrieve environment variable as string, fallback to defValue if not specified
func getStringOrDefault(name, defValue string) string {
	v, ok := os.LookupEnv(name)
	if !ok {
		return defValue
	}
	return v
}

// getIntOrDefault allows to retrieve environment variable as integer, fallback to defValue if not specified
func getIntOrDefault(name string, defValue int) int {
	v, ok := os.LookupEnv(name)
	if !ok {
		return defValue
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return defValue
	}
	return i
}

// getIntOrFail gets environment variable as integer, fail if not specified
func getIntOrFail(name string) (int, error) {
	v, ok := os.LookupEnv(name)
	if !ok {
		return 0, fmt.Errorf("environment variable %q was not set", name)
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("environment variable %q is not integer", name)
	}
	return i, nil
}

// existDir checks whether directory exist
func existDir(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}
