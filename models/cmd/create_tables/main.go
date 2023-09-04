package main

import (
	"fmt"
	"log"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/grokify/mogo/fmt/fmtutil"
	"github.com/grokify/petstore/models"
	"github.com/grokify/petstore/models/cmd"
	flags "github.com/jessevdk/go-flags"
)

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

	fmtutil.PrintJSON(ds)
	db, err := ds.Open()
	if err != nil {
		panic(err)
	}

	err = models.CreatePetTable(db)
	if err != nil {
		panic(err)
	}

	fmt.Println("DONE")
}
