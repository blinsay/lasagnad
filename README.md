## lasagnad

`lasagnad` is short for Lasagna Dad. it's a bot for slack, and it's bad.

#### building and running

Building and running `lasagnad` requires a working `go` toolchain. Run
`make all build` to get you a `lasagnad` executable.

Once you have an `lasagnad`, the next step is to add your auth token in your
config file or in an environment variable.

A lasagna config files is located in `~/.config/lasagnad/config.ini`. See
`example_config.ini` for all of the possible things to configure. Any setting
in the config file can be set as an environment variable by prefixing it with
`GARF_` (e.g. `GARF_DEBUG=true`) or as a command line flag (e.g. `lasagnad
--debug`).
