package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

type TestSubmitSequenceRequest struct {
	method   string
	path     string
	body     io.Reader
	verifier func(*testing.T, *httptest.ResponseRecorder)
}

type DatabaseStub struct {
	Connected bool
	Config    []Config
}

func (d *DatabaseStub) GetConfigById(id int) (*Config, error) {
	return &d.Config[id], nil
}

func (d *DatabaseStub) GetConfigByName(name string) (*Config, error) {
	for _, cfg := range d.Config {
		if cfg.Name == name {
			return &cfg, nil
		}
	}
	return nil, nil
}

func (d *DatabaseStub) GetConfigs() (*[]Config, error) {
	return &d.Config, nil
}

func (d *DatabaseStub) InsertConfig(cfg *Config) (int, error) {
	d.Config = append(d.Config, *cfg)
	return len(d.Config), nil
}

func (d *DatabaseStub) DeleteConfigByName(name string) error {
	for idx, cfg := range d.Config {
		if cfg.Name == name {
			d.Config = append(d.Config[0:idx], d.Config[idx+1:]...)
		}
	}
	return nil
}

func (d *DatabaseStub) UpdateConfigByName(name string, newCfg *Config) error {
	for _, cfg := range d.Config {
		if cfg.Name == name {
			cfg.Metadata = newCfg.Metadata
		}
	}
	return nil
}

func (d *DatabaseStub) IsConnected() bool {
	return d.Connected
}

func assertResponseBody(t *testing.T, got, want string) {
	t.Helper()

	gotTrim := strings.TrimSpace(got)

	if gotTrim != want {
		t.Errorf("expected %q but got %q", want, gotTrim)
	}
}

func assertResponseCode(t *testing.T, got, want int) {
	t.Helper()
	if got != want {
		t.Errorf("expected %d but got %d", want, got)
	}
}

func assertConfig(t *testing.T, got, want string) {
	t.Helper()

	var a, b Config

	if err := json.Unmarshal([]byte(got), &a); err != nil {
		t.Fatal("Unexpected error:", err)
	}
	if err := json.Unmarshal([]byte(want), &b); err != nil {
		t.Fatal("Unexpected error:", err)
	}

	if !cmp.Equal(a, b) {
		t.Errorf("Config received\n%s", cmp.Diff(b, a))
	}
}

func assertConfigs(t *testing.T, got, want string) {
	t.Helper()

	var a, b []Config

	if err := json.Unmarshal([]byte(got), &a); err != nil {
		t.Fatal("Unexpected error:", err)
	}
	if err := json.Unmarshal([]byte(want), &b); err != nil {
		t.Fatal("Unexpected error:", err)
	}

	if !cmp.Equal(a, b) {
		t.Errorf("Config received\n%s", cmp.Diff(b, a))
	}
}

func prepareRequest(t *testing.T, method, path string, body io.Reader) (*http.Request, *httptest.ResponseRecorder) {
	t.Helper()
	req, err := http.NewRequest(method, path, body)
	if err != nil {
		t.Fatal("Unexpected error:", err)
	}
	res := httptest.NewRecorder()
	return req, res
}

func createTable(t *testing.T, db *Database) {
	t.Helper()

	stmt := `
		CREATE TABLE configs (
			id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			name VARCHAR(255) NOT NULL,
			metadata TEXT NOT NULL,
			created_at DATETIME NOT NULL
		);
		CREATE INDEX idx_configs_created ON configs(created_at);
		`

	_, err := db.Exec(stmt)
	if err != nil {
		t.Fatal("Unexpected error:", err)
	}
}

func submitRequestInMem(t *testing.T, initFunction InitializerFunc, req *http.Request, res *httptest.ResponseRecorder) {
	t.Helper()

	if initFunction == nil {
		initFunction = func(db *Database) error {
			return nil
		}
	}

	memStore, cleanUp, err := NewMemDatabaseStore(initFunction)
	if err != nil {
		t.Fatal("Unexpected error:", err)
	}
	defer cleanUp()

	server, err := NewWebServer(memStore)
	if err != nil {
		t.Fatal("Unexpected error:", err)
	}

	server.Handler.ServeHTTP(res, req)
}

func submitSequenceRequestInMem(t *testing.T, initFunction InitializerFunc, testPairs *[]TestSubmitSequenceRequest) {
	t.Helper()

	if initFunction == nil {
		initFunction = func(db *Database) error {
			return nil
		}
	}

	memStore, cleanUp, err := NewMemDatabaseStore(initFunction)
	if err != nil {
		t.Fatal("Unexpected error:", err)
	}
	defer cleanUp()

	server, err := NewWebServer(memStore)
	if err != nil {
		t.Fatal("Unexpected error:", err)
	}

	for _, pair := range *testPairs {
		req, res := prepareRequest(t, pair.method, pair.path, pair.body)
		server.Handler.ServeHTTP(res, req)
		pair.verifier(t, res)
	}

}

func submitRequestStub(t *testing.T, store DatabaseStore, req *http.Request, res *httptest.ResponseRecorder) {
	t.Helper()

	server, err := NewWebServer(store)
	if err != nil {
		t.Fatal("Unexpected error:", err)
	}

	server.Handler.ServeHTTP(res, req)
}

func TestGetHealth(t *testing.T) {

	t.Run("valid", func(t *testing.T) {
		os.Setenv("SERVE_PORT", "8080")
		defer os.Unsetenv("SERVE_PORT")

		req, res := prepareRequest(t, http.MethodGet, "/healthz", nil)

		submitRequestInMem(t, nil, req, res)

		got := res.Body.String()
		want := "ok"

		assertResponseBody(t, got, want)
	})

	t.Run("failure", func(t *testing.T) {
		os.Setenv("SERVE_PORT", "8080")
		defer os.Unsetenv("SERVE_PORT")

		req, res := prepareRequest(t, http.MethodGet, "/healthz", nil)

		storeStub := &DatabaseStub{Connected: false}

		submitRequestStub(t, storeStub, req, res)

		assertResponseCode(t, res.Code, http.StatusInternalServerError)
	})

}

func TestGetConfigs(t *testing.T) {

	t.Run("valid", func(t *testing.T) {
		os.Setenv("SERVE_PORT", "8080")
		defer os.Unsetenv("SERVE_PORT")

		req, res := prepareRequest(t, http.MethodGet, "/configs", nil)

		cfgNew := &Config{
			Name: "test",
			Metadata: &Metadata{
				"monitoring": &Monitoring{
					Enabled: true,
				},
				"limits": &Limits{
					Cpu: Cpu{
						Enabled: true,
						Value:   "300m",
					},
				},
			},
		}

		initDB := func(db *Database) error {
			createTable(t, db)
			_, err := db.InsertConfig(cfgNew)
			return err
		}

		submitRequestInMem(t, initDB, req, res)

		assertResponseCode(t, res.Code, http.StatusOK)

		got := res.Body.String()
		want := `[{"id":1,"name":"test","metadata":{"limits":{"cpu":{"enabled":true,"value":"300m"}},"monitoring":{"enabled":true}}}]`

		assertConfigs(t, got, want)

		assertResponseBody(t, got, want)
	})

	t.Run("valid stub", func(t *testing.T) {
		os.Setenv("SERVE_PORT", "8080")
		defer os.Unsetenv("SERVE_PORT")

		req, res := prepareRequest(t, http.MethodGet, "/configs", nil)

		cfgNew := &Config{
			ID:   1,
			Name: "test",
			Metadata: &Metadata{
				"monitoring": &Monitoring{
					Enabled: true,
				},
				"limits": &Limits{
					Cpu: Cpu{
						Enabled: true,
						Value:   "300m",
					},
				},
			},
		}

		storeStub := &DatabaseStub{Connected: true}
		storeStub.InsertConfig(cfgNew)

		if len(storeStub.Config) != 1 {
			t.Errorf("expected Config len %d but got %d", 1, len(storeStub.Config))
		}

		submitRequestStub(t, storeStub, req, res)

		assertResponseCode(t, res.Code, http.StatusOK)

		got := res.Body.String()
		want := `[{"id":1,"name":"test","metadata":{"limits":{"cpu":{"enabled":true,"value":"300m"}},"monitoring":{"enabled":true}}}]`

		assertConfigs(t, got, want)

		assertResponseBody(t, got, want)
	})

}

func TestGetConfigsOne(t *testing.T) {

	t.Run("valid", func(t *testing.T) {
		os.Setenv("SERVE_PORT", "8080")
		defer os.Unsetenv("SERVE_PORT")

		req, res := prepareRequest(t, http.MethodGet, "/configs/abc", nil)

		cfgNew := &Config{
			Name: "abc",
			Metadata: &Metadata{
				"monitoring": &Monitoring{
					Enabled: true,
				},
				"limits": &Limits{
					Cpu: Cpu{
						Enabled: true,
						Value:   "300m",
					},
				},
			},
		}

		initDB := func(db *Database) error {
			createTable(t, db)
			_, err := db.InsertConfig(cfgNew)
			return err
		}

		submitRequestInMem(t, initDB, req, res)

		got := res.Body.String()
		want := `{"id":1,"name":"abc","metadata":{"limits":{"cpu":{"enabled":true,"value":"300m"}},"monitoring":{"enabled":true}}}`

		assertConfig(t, got, want)

		assertResponseBody(t, got, want)
	})

}

func TestGetDefault(t *testing.T) {

	t.Run("valid", func(t *testing.T) {
		os.Setenv("SERVE_PORT", "8080")
		defer os.Unsetenv("SERVE_PORT")

		req, res := prepareRequest(t, http.MethodGet, "/", nil)

		submitRequestInMem(t, nil, req, res)

		got := res.Body.String()
		want := "fresh-server - build dev"

		assertResponseBody(t, got, want)
	})

}

func TestPostConfigs(t *testing.T) {

	t.Run("valid", func(t *testing.T) {
		os.Setenv("SERVE_PORT", "8080")
		defer os.Unsetenv("SERVE_PORT")

		body := strings.NewReader(`{"id":1,"name":"test","metadata":{"limits":{"cpu":{"enabled":true,"value":"300m"}},"monitoring":{"enabled":true}}}`)

		req, res := prepareRequest(t, http.MethodPost, "/configs", body)

		req.Header.Set("Content-Type", "application/json")

		submitRequestInMem(t, nil, req, res)

		got := res.Body.String()
		want := "new configuration item has successfully been added"

		assertResponseCode(t, res.Code, http.StatusOK)

		assertResponseBody(t, got, want)
	})

	t.Run("valid stub", func(t *testing.T) {
		os.Setenv("SERVE_PORT", "8080")
		defer os.Unsetenv("SERVE_PORT")

		body := strings.NewReader(`{"id":1,"name":"test","metadata":{"limits":{"cpu":{"enabled":true,"value":"300m"}},"monitoring":{"enabled":true}}}`)

		req, res := prepareRequest(t, http.MethodPost, "/configs", body)

		storeStub := &DatabaseStub{Connected: true}

		submitRequestStub(t, storeStub, req, res)

		assertResponseCode(t, res.Code, http.StatusOK)

		got := res.Body.String()
		want := "new configuration item has successfully been added"

		assertResponseBody(t, got, want)

		if len(storeStub.Config) != 1 {
			t.Errorf("expected Config len %d but got %d", 1, len(storeStub.Config))
		}
	})

}

func TestGetSearch(t *testing.T) {

	t.Run("valid", func(t *testing.T) {
		os.Setenv("SERVE_PORT", "8080")
		defer os.Unsetenv("SERVE_PORT")

		req, res := prepareRequest(t, http.MethodGet, "/search", nil)

		submitRequestInMem(t, nil, req, res)

		got := res.Body.String()
		want := "Search GET!"

		assertResponseBody(t, got, want)
	})

}

func TestPatchConfigsOne(t *testing.T) {

	t.Run("valid", func(t *testing.T) {

		os.Setenv("SERVE_PORT", "8080")
		defer os.Unsetenv("SERVE_PORT")

		cfgNew := &Config{
			Name: "abc",
			Metadata: &Metadata{
				"monitoring": &Monitoring{
					Enabled: true,
				},
				"limits": &Limits{
					Cpu: Cpu{
						Enabled: true,
						Value:   "300m",
					},
				},
			},
		}

		initDB := func(db *Database) error {
			createTable(t, db)
			_, err := db.InsertConfig(cfgNew)
			return err
		}

		testPairs := []TestSubmitSequenceRequest{
			{
				method: http.MethodPatch,
				path:   "/configs/abc",
				body:   strings.NewReader(`{"id":1,"name":"abc","metadata":{"limits":{"cpu":{"enabled":false,"value":"300m"}},"monitoring":{"enabled":false}}}`),
				verifier: func(t *testing.T, res *httptest.ResponseRecorder) {
					got := res.Body.String()
					want := "new configuration item has successfully been updated"
					assertResponseCode(t, res.Code, http.StatusOK)
					assertResponseBody(t, got, want)
				},
			},
			{
				method: http.MethodGet,
				path:   "/configs/abc",
				body:   nil,
				verifier: func(t *testing.T, res *httptest.ResponseRecorder) {
					got := res.Body.String()
					want := `{"id":1,"name":"abc","metadata":{"limits":{"cpu":{"enabled":false,"value":"300m"}},"monitoring":{"enabled":false}}}`
					assertConfig(t, got, want)
					assertResponseBody(t, got, want)
				},
			},
		}

		submitSequenceRequestInMem(t, initDB, &testPairs)
	})

}

func TestDeleteConfigsOne(t *testing.T) {

	t.Run("valid", func(t *testing.T) {
		os.Setenv("SERVE_PORT", "8080")
		defer os.Unsetenv("SERVE_PORT")

		req, res := prepareRequest(t, http.MethodDelete, "/configs/abc", nil)

		cfgNew := &Config{
			Name: "abc",
			Metadata: &Metadata{
				"monitoring": &Monitoring{
					Enabled: true,
				},
				"limits": &Limits{
					Cpu: Cpu{
						Enabled: true,
						Value:   "300m",
					},
				},
			},
		}

		initDB := func(db *Database) error {
			createTable(t, db)
			_, err := db.InsertConfig(cfgNew)
			return err
		}

		submitRequestInMem(t, initDB, req, res)

		got := res.Body.String()
		want := "configuration item has successfully been erased"

		assertResponseCode(t, res.Code, http.StatusOK)

		assertResponseBody(t, got, want)
	})

	t.Run("valid stub", func(t *testing.T) {
		os.Setenv("SERVE_PORT", "8080")
		defer os.Unsetenv("SERVE_PORT")

		req, res := prepareRequest(t, http.MethodDelete, "/configs/test", nil)

		cfgNew := &Config{
			ID:   1,
			Name: "test",
			Metadata: &Metadata{
				"monitoring": &Monitoring{
					Enabled: true,
				},
				"limits": &Limits{
					Cpu: Cpu{
						Enabled: true,
						Value:   "300m",
					},
				},
			},
		}

		storeStub := &DatabaseStub{Connected: true}
		storeStub.InsertConfig(cfgNew)

		if len(storeStub.Config) != 1 {
			t.Errorf("expected Config len %d but got %d", 1, len(storeStub.Config))
		}

		submitRequestStub(t, storeStub, req, res)

		assertResponseCode(t, res.Code, http.StatusOK)

		got := res.Body.String()
		want := "configuration item has successfully been erased"

		assertResponseBody(t, got, want)

		if len(storeStub.Config) != 0 {
			t.Errorf("expected Config len %d but got %d", 0, len(storeStub.Config))
		}

	})

}
