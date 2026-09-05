# go-depend

go-depend is a command line tool that calculates software package metrics for Go packages.

For more background information see [fundamentals.md](./doc/fundamentals.md).

## Metrics

go-depend can calculate the following metrics for a Go package:
- number of outgoing dependencies (to the Go standard library, to other external libraries)
- instability (I)
- abstractness (A)
- distance from the main sequence (D = | A + I - 1 |)

## Invariants

go-depend can check for a Go package if the following invariants are violated:
- stable dependencies: unstable packages should rely on stable packages
- stable abstractions: stable packages should be abstract
- there should be no packages in either the zone of pain or the zone of uselessness

## Dependency Graph

go-depend can produce dependency graphs with different focus using [mermaid](https://github.com/mermaid-js/mermaid) for visualization:
- starting from the `main` package for the whole application, covering only packages from the module
- starting from an arbitrary package, going in both directions (incoming -> selected package -> outgoing)
- for an arbitrary external package, including transitive dependencies, covering only incoming dependencies and packages from the module

## Reality Check with Git

go-depend can evaluate the git history of a Go module to find out, which packages were changed when and how often:
- constant change over time -> the package is considered "unstable"
- high change rate only for a certain amount of time -> the package is considered "stable"

The reality check reports the perceived instability in a column next to the calculated instability. It does not calculate a difference between the two values: the instability counts imports, the perceived instability counts months. Both values are between 0 and 1, but their difference has no meaning. Compare the two columns yourself.

## Usage

```shell
go-depend scan [pattern ...] [flags]
```
Scan Go packages matching the given patterns (defaults to `./...` - the whole module) and outputs a table sorted by the package name, containing the metrics and markers for violations of any invariant.

```shell
go-depend graph [pattern] [flags]
```
Produce a dependency graph for the selected package (defaults to `.` - the package of the current working dir) and write it either to stdout or to a file (`--output filename`).

Patterns use the same syntax as `go list` / `go build`: `./...` (recursive), `./internal/score` (single package), `github.com/x/y` (import path). Multiple patterns are accepted (e.g. `./internal/foo ./internal/bar`).

### Patterns in a workspace

The root directory of a Go workspace is no module. Therefore the pattern `./...` matches no package if you use it in the root directory of a workspace. The Go tools show this error message:

```
directory prefix . does not contain modules listed in go.work or their selected dependencies
```

Give one pattern for each module of the workspace:

```shell
go-depend scan ./a/... ./b/...
```

### Two-layer selection

go-depend uses two independent mechanisms to narrow what gets analyzed:
1. **Patterns** select which Go *packages* are scanned. Operates at the package/directory level.
2. **`--exclude`** filters files within matched packages via regex. Cannot exclude whole packages from being tested, only from the final report.

They are designed to work together:

```shell
# Scan all packages, but skip generated protobuf files inside them
go-depend scan --exclude '\.pb\.go$'

# Narrow to specific packages, then exclude files within them
go-depend scan ./internal/scan ./internal/coverage --exclude 'mock_'
```

### Flags

#### Flags for `scan`

| Flag | Default | Description |
| --- | --- | --- |
| `--exclude regex` | none | Excludes all files with a path that matches the regular expression. go-depend does not read these files. Their declarations and their imports have no effect on the metrics. Give the flag more than one time for more than one expression. |
| `--history` | off | Reads the git history and calculates the perceived instability of each package. The table shows it in an additional column. |
| `--since date` | the full history | Reads only the part of the git history after the given date. Use this flag only together with `--history`. |
| `--max-distance value` | `0.5` | Marks a package as a violation if its distance from the main sequence is more than this value. |
| `--format table\|json\|csv` | `table` | Selects the output format. |
| `--fail-on-violation` | off | Makes go-depend exit with the code 1 if it finds one violation or more. |

go-depend exits with the code 0 if it finds no error. It exits with the code 1 if you give the flag `--fail-on-violation` and it finds one violation or more. It exits with the code 2 if an error occurs.

#### Flags for `graph`

| Flag | Default | Description |
| --- | --- | --- |
| `--exclude regex` | none | Excludes all files with a path that matches the regular expression. |
| `--incoming` | see below | Includes the packages that depend on the selected package. |
| `--outgoing` | see below | Includes the packages that the selected package depends on. |
| `--depth n` | `1` | Limits the graph to `n` steps from the selected package. The value 0 removes the limit. |
| `--output filename` | stdout | Writes the graph to the given file. |

If you give neither `--incoming` nor `--outgoing`, go-depend uses both directions.

Use the flags `--incoming`, `--outgoing` and `--depth` together to select the focus of the graph:

```shell
# the whole application, from the main package
go-depend graph . --outgoing --depth 0

# the direct neighbours of one package (the default)
go-depend graph ./metrics

# all packages of the module that depend on an external package
go-depend graph github.com/spf13/cobra --incoming --depth 0
```


## License

This software is published under the [MIT License](https://www.tldrlegal.com/l/mit).

Copyright [Florian Thienel](http://thecodingflow.com/)
