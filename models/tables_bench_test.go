package models

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/grokify/cryptocharacters"
	"github.com/grokify/petstore/models/cmd"
)

// benchDataSourceEnv names the data source, in the file that
// cmd.DataSourcesEnv points to, that BenchmarkPet inserts into.
const benchDataSourceEnv = "PETSTORE_BENCH_DATASOURCE"

// BenchmarkPet inserts pets into a MySQL database. It is skipped unless
// PETSTORE_DATASOURCES names a data sources file and
// PETSTORE_BENCH_DATASOURCE names one of its entries:
//
//	PETSTORE_DATASOURCES=~/.config/petstore/datasources.json \
//	PETSTORE_BENCH_DATASOURCE=MY8MICRO \
//	go test ./models -bench Pet -benchtime 1000x
func BenchmarkPet(b *testing.B) {
	name := os.Getenv(benchDataSourceEnv)
	if name == "" {
		b.Skipf("%s is not set", benchDataSourceEnv)
	}
	dss, err := cmd.LoadDataSourceSet()
	if errors.Is(err, cmd.ErrDataSourcesUnset) {
		b.Skip(err)
	}
	if err != nil {
		b.Fatal(err)
	}
	ds, err := dss.GetDataSource(name)
	if err != nil {
		b.Fatal(err)
	}
	db, err := ds.Open()
	if err != nil {
		b.Fatal(err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			b.Error(err)
		}
	}()

	const insert = `
		INSERT INTO pet (name, status, creation_time, update_time)
		VALUES (?,?,?,?)`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dt := time.Now()
		if _, err := db.Exec(insert, cryptocharacters.NameX(uint(i)), "available", dt, dt); err != nil {
			b.Fatal(err)
		}
	}
}
