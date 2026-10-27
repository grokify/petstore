# MySQL Tools

`create_tables` and `create_pets` create the Petstore `pet` table in a MySQL
database and measure inserts, and `main.java` is a JBang example for the same
databases. They read the databases they can use from a JSON file named by
`PETSTORE_DATASOURCES`; see [`datasources.example.json`](datasources.example.json)
for the format.

The file holds credentials: keep it outside the repository and restrict it with
`chmod 600`. The tools refuse a file that other users can read.

For setup, the commands and the benchmark, see the
[MySQL Tools guide](../../docs/guides/mysql-tools.md).
