package main

import (
	"os"
	"testing"
)

func TestGetEnvVarAsString(t *testing.T) {

	t.Run("defined", func(t *testing.T) {
		os.Setenv("TEST_ENV_VAR", "abc")
		defer os.Unsetenv("TEST_ENV_VAR")

		got := getStringOrDefault("TEST_ENV_VAR", "xyz")
		want := "abc"
		if got != want {
			t.Errorf("expected %q but got %q", want, got)
		}
	})

	t.Run("undefined", func(t *testing.T) {
		got := getStringOrDefault("TEST_ENV_VAR", "xyz")
		want := "xyz"
		if got != want {
			t.Errorf("expected %q but got %q", want, got)
		}
	})

}

func TestGetEnvVarAsInteger(t *testing.T) {

	t.Run("defined valid", func(t *testing.T) {
		os.Setenv("TEST_ENV_VAR", "123")
		defer os.Unsetenv("TEST_ENV_VAR")

		got := getIntOrDefault("TEST_ENV_VAR", 456)
		want := 123
		if got != want {
			t.Errorf("expected %d but got %d", want, got)
		}
	})

	t.Run("defined invalid", func(t *testing.T) {
		os.Setenv("TEST_ENV_VAR", "abc")
		defer os.Unsetenv("TEST_ENV_VAR")

		got := getIntOrDefault("TEST_ENV_VAR", 456)
		want := 456
		if got != want {
			t.Errorf("expected %d but got %d", want, got)
		}
	})

	t.Run("undefined", func(t *testing.T) {
		got := getIntOrDefault("TEST_ENV_VAR", 456)
		want := 456
		if got != want {
			t.Errorf("expected %d but got %d", want, got)
		}
	})

}

func TestGetEnvVarAsIntegerFail(t *testing.T) {

	t.Run("defined valid", func(t *testing.T) {
		os.Setenv("TEST_ENV_VAR", "123")
		defer os.Unsetenv("TEST_ENV_VAR")

		got, err := getIntOrFail("TEST_ENV_VAR")
		if err != nil {
			t.Fatal("unexpected error:", err)
		}

		want := 123
		if got != want {
			t.Errorf("expected %d but got %d", want, got)
		}
	})

	t.Run("defined invalid", func(t *testing.T) {
		os.Setenv("TEST_ENV_VAR", "abc")
		defer os.Unsetenv("TEST_ENV_VAR")

		_, err := getIntOrFail("TEST_ENV_VAR")
		if err == nil {
			t.Fatal("expected error, none thrown")
		}
	})

	t.Run("undefined", func(t *testing.T) {
		_, err := getIntOrFail("TEST_ENV_VAR")
		if err == nil {
			t.Fatal("expected error, none thrown")
		}
	})
}
