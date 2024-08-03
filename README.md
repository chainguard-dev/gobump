# GoBump
GoBump is a simple command-line tool written in Go that allows you to update the versions of your Go dependencies.

## Usage

```shell
gobump --packages=<package@version> ... --modroot=<path to go.mod>
```

### Flags

* `--packages`: A space-separated list of packages to update. Each package should be in the format `package@version`.
* `--modroot`: Path to the go.mod root. If not specified, it defaults to the current directory.
* `--replaces`: A space-separated list of packages to replace. Each entry should be in the format `old=new@version`.
* `--go-version`: set the go-version for 'go mod tidy' command, default to the go version in the build environment.
* `--show-diff`: Show the difference between the original and 'go.mod' files.
* `--tidy`:  Run 'go mod tidy' command.
* `--skip-initial-tidy`: Skip the initial 'go mod tidy' command before updating and replacing the packages.
* `--bump-file`: Specify the yaml file where to read the bump instructions from

## Example

### Using flags

```shell
gobump --packages="github.com/pkg/errors@v0.9.1 golang.org/x/mod@v0.4.2" --modroot=/path/to/your/project
```

This will update the versions of `github.com/pkg/errors` and `golang.org/x/mod` in your `go.mod` file.

### Using file

Create a file `bumps.yaml`

```yaml
packages:
  - name: github.com/pkg/errors
    version: v0.9.1
  - name: golang.org/x/mod
    version: v0.4.2
```

And it does the same as above flags. You can also specify `replace` and
`require` in the yaml fields. Some [examples](./pkg/update/testdata/).
**Note** Index field is not used.

## Requirements

Go 1.20 or later

## Installation
To install gobump, you can use go install:

```shell
go install github.com/chainguard-dev/gobump@latest
```

## Contributing
Contributions are welcome! Please submit a pull request on GitHub.

