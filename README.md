# SuperDaemon

SuperDaemon is a fork of [Pterodactyl Wings](https://github.com/pterodactyl/wings). It provides the same HTTP API and SFTP access to manage game-server containers, but is developed independently.

SuperDaemon provides an HTTP API allowing you to interface directly with running server
instances, fetch server logs, generate backups, and control all aspects of the server lifecycle.

In addition, SuperDaemon ships with a built-in SFTP server allowing your system to remain free of
Pterodactyl-specific dependencies, and allowing users to authenticate with the same credentials they
would normally use to access the Panel.

## Migrating from Wings

SuperDaemon is a drop-in replacement for Wings in most setups. The binary name is `superdaemon`
instead of `wings`, but the configuration layout and paths intentionally stay compatible:

| Wings                 | SuperDaemon                                         |
|-----------------------|-----------------------------------------------------|
| `wings`               | `superdaemon`                                       |
| `/etc/pterodactyl/config.yml` | `/etc/pterodactyl/config.yml` (same location) |
| `/var/lib/pterodactyl`      | `/var/lib/pterodactyl`                      |
| `/var/log/pterodactyl`      | `/var/log/pterodactyl`                      |

To migrate from an existing Wings installation:

1. Download the `superdaemon_linux_amd64` or `superdaemon_linux_arm64` release asset for your platform.
2. Stop the running `wings` service.
3. Replace the binary path in your systemd unit (or container image) with `superdaemon`.
4. Start the `superdaemon` service. Your existing `config.yml` and server data continue to work unchanged.

## Releases

Pre-built binaries are attached to each GitHub Release. The release names use the SuperDaemon binary name:

- `superdaemon_linux_amd64`
- `superdaemon_linux_arm64`

## Documentation

Upstream Wings documentation is a good starting point because SuperDaemon keeps the same
configuration format and API surface:

* [Wings Documentation](https://pterodactyl.io/wings/1.0/installing.html)
* [Community Guides](https://pterodactyl.io/community/about.html)

For SuperDaemon-specific issues, please use this repository.

## Reporting Issues

Please use the [MrSchnirschuh/SuperDaemon](https://github.com/MrSchnirschuh/SuperDaemon) repository
to report issues or make feature requests for SuperDaemon.

Security issues should be reported privately to the repository maintainer.
