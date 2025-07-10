package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func setEnvVars(vars map[string]string) func() {
	originals := make(map[string]string)
	for k, v := range vars {
		originals[k] = os.Getenv(k)
		os.Setenv(k, v)
	}
	return func() {
		for k, v := range originals {
			os.Setenv(k, v)
		}
	}
}

func unsetEnvVars(keys ...string) func() {
	originals := make(map[string]string)
	for _, k := range keys {
		originals[k] = os.Getenv(k)
		os.Unsetenv(k)
	}
	return func() {
		for k, v := range originals {
			os.Setenv(k, v)
		}
	}
}

func TestLoadConfig_Success(t *testing.T) {
	vars := map[string]string{
		"POSTGRES_HOST":     "localhost",
		"POSTGRES_PORT":     "5432",
		"POSTGRES_USER":     "user",
		"POSTGRES_PASSWORD": "pass",
		"POSTGRES_DB":       "testdb",
		"SERVER_PORT":       "1234",
		"APP_ENV":           "test",
	}
	cleanup := setEnvVars(vars)
	defer cleanup()

	cfg, err := LoadConfig()
	assert.NoError(t, err)
	assert.Equal(t, "localhost", vars["POSTGRES_HOST"])
	assert.Contains(t, cfg.DBUrl, "host=localhost")
	assert.Contains(t, cfg.DBUrl, "port=5432")
	assert.Contains(t, cfg.DBUrl, "user=user")
	assert.Contains(t, cfg.DBUrl, "password=pass")
	assert.Contains(t, cfg.DBUrl, "dbname=testdb")
	assert.Equal(t, "1234", cfg.ServerPort)
	assert.Equal(t, "test", cfg.Env)
}

func TestLoadConfig_Defaults(t *testing.T) {
	vars := map[string]string{
		"POSTGRES_HOST":     "localhost",
		"POSTGRES_PORT":     "5432",
		"POSTGRES_USER":     "user",
		"POSTGRES_PASSWORD": "pass",
		"POSTGRES_DB":       "testdb",
	}
	cleanup := setEnvVars(vars)
	defer cleanup()
	os.Unsetenv("SERVER_PORT")
	os.Unsetenv("APP_ENV")

	cfg, err := LoadConfig()
	assert.NoError(t, err)
	assert.Equal(t, "3000", cfg.ServerPort)
	assert.Equal(t, "development", cfg.Env)
}

func TestLoadConfig_MissingRequiredEnv(t *testing.T) {
	required := []string{"POSTGRES_HOST", "POSTGRES_PORT", "POSTGRES_USER", "POSTGRES_PASSWORD", "POSTGRES_DB"}
	for _, missing := range required {
		vars := map[string]string{
			"POSTGRES_HOST":     "localhost",
			"POSTGRES_PORT":     "5432",
			"POSTGRES_USER":     "user",
			"POSTGRES_PASSWORD": "pass",
			"POSTGRES_DB":       "testdb",
		}
		delete(vars, missing)
		cleanup := setEnvVars(vars)
		defer cleanup()
		os.Unsetenv(missing)

		func() {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("expected panic for missing %s", missing)
				}
			}()
			LoadConfig()
		}()
		cleanup()
	}
}
