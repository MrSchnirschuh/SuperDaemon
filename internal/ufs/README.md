# Filesystem

This package provides a filesystem abstraction used by SuperDaemon. It wraps
path handling, file operations, and directory traversal with the server-root
semantics required by the daemon.

## Licensing

Most code in this package is licensed under `MIT` with some exceptions.

The following files are licensed under `BSD-3-Clause` due to them being copied
verbatim or derived from [Go](https://go.dev)'s source code.

- [`file_posix.go`](./file_posix.go)
- [`mkdir_unix.go`](./mkdir_unix.go)
- [`path_unix.go`](./path_unix.go)
- [`removeall_unix.go`](./removeall_unix.go)
- [`stat_unix.go`](./stat_unix.go)
- [`walk.go`](./walk.go)

These changes are not associated with nor endorsed by The Go Authors.
