# MySQL Tools

`create_tables` and `create_pets` create the Petstore tables in a MySQL
database and measure inserts. `main.java` is a JBang example that connects to
the same databases from Java. `BenchmarkPet` in `models` inserts pets the same
way as a Go benchmark.

## Data Sources

The tools read the databases they can use from a JSON file, named by the
`PETSTORE_DATASOURCES` environment variable. It maps a name to a data source;
see [`datasources.example.json`](datasources.example.json):

```json
{
  "MY8MICRO": {
    "driver": "mysql",
    "hostname": "db.example.com",
    "port": 3306,
    "user": "testuser",
    "password": "your-password",
    "database": "petstore"
  }
}
```

The file holds credentials, so keep it outside the repository, for example
at `~/.config/petstore/datasources.json`, and restrict it with `chmod 600`.
The tools refuse a file that other users can read.

## Run

```bash
export PETSTORE_DATASOURCES=~/.config/petstore/datasources.json
go run ./models/cmd/create_tables -t MY8MICRO
go run ./models/cmd/create_pets -t MY8MICRO -i 1000
```

If the name is not in the file, the tools list the names it holds.

The benchmark needs a second variable for the entry to use, and is skipped
without it:

```bash
PETSTORE_BENCH_DATASOURCE=MY8MICRO go test ./models -bench Pet -benchtime 1000x
```
