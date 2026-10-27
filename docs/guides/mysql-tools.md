# MySQL Tools

`create_tables` and `create_pets` create the Petstore `pet` table in a MySQL
database and measure inserts. `main.java` is a [JBang](https://www.jbang.dev/)
example that connects to the same databases from Java. `BenchmarkPet` in the
`models` package inserts pets the same way as a Go benchmark.

## Data Sources

The tools read the databases they can use from a JSON file, named by the
`PETSTORE_DATASOURCES` environment variable. It maps a name to a data source;
the repository carries an example, `models/cmd/datasources.example.json`:

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

The file holds credentials, so keep it outside the repository, for example at
`~/.config/petstore/datasources.json`, and restrict it with `chmod 600`. The
tools refuse a file that other users can read. Windows does not enforce file
modes, so the check is skipped there.

## Run

```bash
export PETSTORE_DATASOURCES=~/.config/petstore/datasources.json
go run ./models/cmd/create_tables -t MY8MICRO
go run ./models/cmd/create_pets -t MY8MICRO -i 1000
```

If the name is not in the file, the tools list the names it holds.
`create_pets` prints the host, the server version and the row count before it
inserts, then the elapsed time. `create_tables` prints only the database and
host, never the password.

## Benchmark

`BenchmarkPet` needs a second variable for the entry to use, and is skipped
without it:

```bash
PETSTORE_BENCH_DATASOURCE=MY8MICRO go test ./models -bench Pet -benchtime 1000x
```

## Java Example

`models/cmd/create_pets/main.java` connects with the MySQL Connector/J 5.1
driver over TLS 1.2, which MySQL 5.7 servers accept. It takes the connection
details and the number of inserts as arguments:

```bash
jbang models/cmd/create_pets/main.java <host> <port> <database> <user> <password> <iterations>
```

The password is a command-line argument here, so other users on the machine can
see it in the process list. Use a throwaway database.
