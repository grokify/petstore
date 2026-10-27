package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

const testDataSources = `{"MY8MICRO": {"driver": "mysql", "hostname": "db.example.com",
  "port": 3306, "user": "testuser", "password": "your-password", "database": "petstore"}}`

func writeDataSources(t *testing.T, mode os.FileMode, body string) string {
	t.Helper()
	dir := t.TempDir()
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := root.Close(); err != nil {
			t.Error(err)
		}
	}()
	const name = "datasources.json"
	if err := root.WriteFile(name, []byte(body), mode); err != nil {
		t.Fatal(err)
	}
	if err := root.Chmod(name, mode); err != nil {
		t.Fatal(err)
	}
	return filepath.Join(dir, name)
}

func TestLoadDataSourceSet(t *testing.T) {
	t.Setenv(DataSourcesEnv, writeDataSources(t, 0o600, testDataSources))
	dss, err := LoadDataSourceSet()
	if err != nil {
		t.Fatal(err)
	}
	ds, err := dss.GetDataSource("MY8MICRO")
	if err != nil {
		t.Fatal(err)
	}
	if ds.Hostname != "db.example.com" || ds.Port != 3306 || ds.Database != "petstore" || ds.Driver != "mysql" {
		t.Fatalf("unexpected data source: %+v", ds)
	}
	if _, err := dss.GetDataSource("missing"); err == nil {
		t.Fatal("want an error for an unknown name")
	}
}

func TestLoadDataSourceSetUnset(t *testing.T) {
	t.Setenv(DataSourcesEnv, "")
	if _, err := LoadDataSourceSet(); !errors.Is(err, ErrDataSourcesUnset) {
		t.Fatalf("got %v, want ErrDataSourcesUnset", err)
	}
}

func TestReadDataSourceSetBadJSON(t *testing.T) {
	if _, err := ReadDataSourceSet(writeDataSources(t, 0o600, "{not json")); err == nil {
		t.Fatal("want an error for invalid JSON")
	}
}

func TestReadDataSourceSetMissingFile(t *testing.T) {
	if _, err := ReadDataSourceSet(filepath.Join(t.TempDir(), "absent.json")); err == nil {
		t.Fatal("want an error for a missing file")
	}
}

func TestReadDataSourceSetRefusesSharedFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not enforce file modes")
	}
	if _, err := ReadDataSourceSet(writeDataSources(t, 0o644, testDataSources)); err == nil {
		t.Fatal("want an error for a file other users can read")
	}
}

func TestExampleFileIsValid(t *testing.T) {
	b, err := os.ReadFile("datasources.example.json")
	if err != nil {
		t.Fatal(err)
	}
	dss, err := ReadDataSourceSet(writeDataSources(t, 0o600, string(b)))
	if err != nil {
		t.Fatal(err)
	}
	if len(dss.Keys()) == 0 {
		t.Fatal("the example holds no data sources")
	}
}
