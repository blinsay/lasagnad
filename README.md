## lasagnad

`lasagnad` is short for Lasagna Dad. it's a bot for pinning images in your
free-tier slack so you don't run out of file space, and it's bad.

#### design

lasagna dad stores images in S3. you can `!pin` images under a name and `!show`
an image with a name. if you `!pin` multiple images with the same name, you'll
see a random one whenever you `!show` that name.

#### building and running

Building and running `lasagnad` requires a working `go` toolchain. Run
`make all build` to get you a `lasagnad` executable.

Once you have an `lasagnad`, the next step is to add your auth token in your
config file or in an environment variable.

A lasagna config files is located in `~/.config/lasagnad/config.ini`. See
`example_config.ini` for all of the possible things to configure. Global config
options (options that don't occur under a subheading) can be passed as command
line flags (e.g. `lasagnad --debug`).
