package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/grokify/cryptocharacters"
	"github.com/grokify/gomysql"
	"github.com/grokify/mogo/fmt/fmtutil"
	"github.com/grokify/mogo/log/logutil"
	"github.com/grokify/petstore/models"
	"github.com/grokify/petstore/models/cmd"
	flags "github.com/jessevdk/go-flags"
)

type TestRun struct {
	Nickname     string
	Host         string
	Version      string
	Iterations   int
	StartCount   int
	StartTime    time.Time
	EndTime      time.Time
	Duration     time.Duration
	DurationMins float64
}

type Options struct {
	TestEnvironment string `short:"t" long:"test-scenario" description:"Test Scenario" required:"true"`
	Iterations      uint   `short:"i" long:"iterations" description:"Test Iterations"`
}

func main() {
	opts := Options{}
	_, err := flags.Parse(&opts)
	if err != nil {
		log.Fatal(err)
	}

	dss := cmd.GetDataSourceSet()
	ds, err := dss.GetDataSource(opts.TestEnvironment)
	if err != nil {
		fmt.Printf("Test Environments are: (%s)\n", strings.Join(dss.Keys(), ","))
		log.Fatal(err)
	}
	if opts.Iterations == 0 {
		opts.Iterations = 10
	}

	tr := TestRun{
		Nickname:   opts.TestEnvironment,
		Iterations: int(opts.Iterations),
	}

	tr.Host = ds.Hostname

	db, err := ds.Open()
	logutil.FatalErr(err)

	if 1 == 0 {
		err = models.CreatePetTable(db)
		if err != nil {
			panic(err)
		}
	}

	ver, err := gomysql.QueryVersion(db)
	logutil.FatalErr(err)
	tr.Version = ver

	count, err := gomysql.GetInt(db, "SELECT COUNT(*) FROM pet")
	logutil.FatalErr(err)
	tr.StartCount = count
	tr.StartTime = time.Now()

	fmtutil.PrintJSON(tr)

	for i := 0; i < tr.Iterations; i++ {
		name := cryptocharacters.NameX(uint(i))

		tx := time.Now()

		sqlStatement := `
		INSERT INTO pet (name, status, creation_time, update_time)
		VALUES (?,?,?,?)`

		vars := []any{name, "available", tx, tx}

		_, err = db.Exec(sqlStatement, vars...)
		if err != nil {
			panic(err)
		}
	}
	tr.EndTime = time.Now()
	tr.Duration = tr.EndTime.Sub(tr.StartTime)
	tr.DurationMins = tr.Duration.Minutes()

	fmtutil.PrintJSON(tr)

	fmt.Println("DONE")
}
