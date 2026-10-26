package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"runtime"

	_ "github.com/go-sql-driver/mysql"
	"github.com/grokify/mogo/database/datasource"
)

// DataSourcesEnv names the environment variable that holds the path of the
// JSON file describing the databases the tools and benchmarks can use.
const DataSourcesEnv = "PETSTORE_DATASOURCES"

// ErrDataSourcesUnset is returned when DataSourcesEnv is not set.
var ErrDataSourcesUnset = errors.New(DataSourcesEnv + " is not set")

// LoadDataSourceSet reads the data sources from the file named by
// DataSourcesEnv. See ReadDataSourceSet for the format.
func LoadDataSourceSet() (datasource.DataSourceSet, error) {
	path := os.Getenv(DataSourcesEnv)
	if path == "" {
		return datasource.DataSourceSet{}, ErrDataSourcesUnset
	}
	return ReadDataSourceSet(path)
}

// ReadDataSourceSet reads a JSON object that maps a name to a data source:
//
//	{"MY8MICRO": {"driver": "mysql", "hostname": "db.example.com",
//	  "port": 3306, "user": "testuser", "password": "your-password",
//	  "database": "petstore"}}
//
// The file holds credentials, so it must not be readable by group or other
// users (mode 0600). Windows does not enforce file modes, so the check is
// skipped there.
func ReadDataSourceSet(path string) (datasource.DataSourceSet, error) {
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			return datasource.DataSourceSet{}, err
		}
		if info.Mode().Perm()&0o077 != 0 {
			return datasource.DataSourceSet{}, fmt.Errorf("%s is readable by other users (mode %04o); run chmod 600 on it", path, info.Mode().Perm())
		}
	}
	b, err := os.ReadFile(path) //nolint:gosec // G304: the path is the operator's own configuration
	if err != nil {
		return datasource.DataSourceSet{}, err
	}
	m := map[string]datasource.DataSource{}
	if err := json.Unmarshal(b, &m); err != nil {
		return datasource.DataSourceSet{}, fmt.Errorf("%s: %w", path, err)
	}
	return datasource.DataSourceSet{Data: m}, nil
}
