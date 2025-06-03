package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// loadEnvToStruct loads environment variables into struct fields and nested structs based on tags
func loadEnvToStruct(target any, prefix, tag string) error {
	v := reflect.ValueOf(target).Elem()
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		// Skip unexported fields
		if !fieldValue.CanSet() {
			continue
		}

		// Get env tag or skip if not present
		// Check if field is a struct that needs recursive processing
		if fieldValue.Kind() == reflect.Struct {
			// Skip time.Duration which is technically a struct but should be treated as primitive
			if field.Type != reflect.TypeOf(time.Duration(0)) {
				if err := loadEnvToStruct(fieldValue.Addr().Interface(), prefix, tag); err != nil {
					return fmt.Errorf("error loading sub config field %s: %w", field.Name, err)
				}
			}
		}

		envVar, ok := field.Tag.Lookup(tag)
		if !ok || envVar == "" {
			continue
		}

		// Get value from environment or skip if empty
		envValue := os.Getenv(prefix + envVar)
		if envValue == "" {
			continue
		}

		// Set field value based on type
		switch fieldValue.Kind() {
		case reflect.String:
			fieldValue.SetString(envValue)

		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if field.Type == reflect.TypeOf(time.Duration(0)) {
				// Handle time.Duration
				duration, err := time.ParseDuration(envValue)
				if err != nil {
					return fmt.Errorf("invalid duration value for %s: %w", envVar, err)
				}
				fieldValue.SetInt(int64(duration))
			} else {
				// Handle regular integers
				val, err := strconv.ParseInt(envValue, 10, 64)
				if err != nil {
					return fmt.Errorf("invalid integer value for %s: %w", envVar, err)
				}
				fieldValue.SetInt(val)
			}

		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			val, err := strconv.ParseUint(envValue, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid unsigned integer value for %s: %w", envVar, err)
			}
			fieldValue.SetUint(val)

		case reflect.Float32, reflect.Float64:
			val, err := strconv.ParseFloat(envValue, 64)
			if err != nil {
				return fmt.Errorf("invalid float value for %s: %w", envVar, err)
			}
			fieldValue.SetFloat(val)

		case reflect.Bool:
			val, err := strconv.ParseBool(envValue)
			if err != nil {
				return fmt.Errorf("invalid boolean value for %s: %w", envVar, err)
			}
			fieldValue.SetBool(val)
		}
	}

	return nil
}

// loadEnvFromAncestors searches for .env files from the current directory up to the root
func loadEnvFromAncestors() error {
	// Get current working directory
	currentDir, err := os.Getwd()
	if err != nil {
		return err
	}

	// Get optional environment name from ENV or APP_ENV
	envName := os.Getenv("ENV")
	if envName == "" {
		envName = os.Getenv("APP_ENV")
	}

	// Files to look for
	filesToTry := []string{".env"}

	// If we have an environment name, also look for .env.[environment]
	if envName != "" {
		filesToTry = append(filesToTry, ".env."+envName)
	}

	// Track if we found any env files
	found := false

	// Start from current directory and move up
	dir := currentDir
	for {
		for _, fileName := range filesToTry {
			envPath := filepath.Join(dir, fileName)
			if _, err := os.Stat(envPath); err == nil {
				// File exists, load it
				if err := godotenv.Load(envPath); err == nil {
					slog.Info("Loading .env file", "file", envPath)
					found = true
				}
			}
		}

		// Stop if we've reached the root directory
		parentDir := filepath.Dir(dir)
		if parentDir == dir {
			break // We've reached the root
		}
		dir = parentDir
	}

	if !found {
		slog.Info("No .env files found in ancestor directories")
	}

	return nil
}
