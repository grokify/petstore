package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
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
	b, err := readPrivateFile(path)
	if err != nil {
		return datasource.DataSourceSet{}, err
	}
	m := map[string]datasource.DataSource{}
	if err := json.Unmarshal(b, &m); err != nil {
		return datasource.DataSourceSet{}, fmt.Errorf("%s: %w", path, err)
	}
	return datasource.DataSourceSet{Data: m}, nil
}

// readPrivateFile reads path through one open handle, checking the mode of
// that handle, so the file cannot change between the check and the read. The
// file is opened inside its own directory, so a symbolic link cannot lead out
// of it.
func readPrivateFile(path string) (b []byte, err error) {
	root, err := os.OpenRoot(filepath.Dir(path))
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, root.Close()) }()
	f, err := root.Open(filepath.Base(path))
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, f.Close()) }()
	if runtime.GOOS != "windows" {
		info, statErr := f.Stat()
		if statErr != nil {
			return nil, statErr
		}
		if info.Mode().Perm()&0o077 != 0 {
			return nil, fmt.Errorf("%s is readable by other users (mode %04o); run chmod 600 on it", path, info.Mode().Perm())
		}
	}
	return io.ReadAll(f)
}
