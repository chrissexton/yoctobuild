# yoctoBuild

yoctoBuild is a bare-bones build service for your personal projects. Since many
hosted build services do not work outside of GitHub and BitBucket, this was
made to work with Gogs for light usage where hosting Jenkins would be absurd.

yoctoBuild is unaware of git, Mercurial, scss, etc. It depends on a bash
executable and proper configuration.

## Usage

Run `yoctobuild` in its directory or copy the badges and a config file to a
working directory. Provide a `-secret` on startup, and then set your Gogs/Git
hooks/etc to send a request to `/projects/<your project>/build?secret=<your
secret>`.

The config file consists of a map of projects and their steps. The build server
does nothing but create a working directory for the build so the build steps
must check out code, perform tests, and manage any dependencies. The config
file should only be modified by owners of the server it lives on as it may
execute arbitrary commands on your behalf.

This build service is very simple. It does not watch for branches or anything
of the like. You must configure a project for each individual item that you
wish to watch.

Badges may be served by embedding `/projects/<your project>/badge` as an image
in your page.
