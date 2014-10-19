# HouseMon

Real-time home monitoring and automation.

[![GoDoc][G]][D] [![License][B]][L]

The [homepage][H], [discussion forum][F], and [issue tracker][I] are at JeeLabs.

WCC specifics
-------------
- Compile jeeboot config: `coffee jeeboot.coffee`
- Compile setup: `coffee setup.coffee`
- Run commandline jeeboot with debugging: `go run main.go -logtostderr=true -vmodule="udpgw=4,utils=4,jeeboot=4" udpgw`

[G]: https://godoc.org/github.com/jcw/housemon?status.png
[D]: https://godoc.org/github.com/jcw/housemon
[B]: http://img.shields.io/badge/license-MIT-brightgreen.svg
[L]: http://opensource.org/licenses/MIT

[H]: http://jeelabs.net/projects/housemon/wiki
[F]: http://jeelabs.net/projects/cafe/boards/9
[I]: http://jeelabs.net/projects/development/issues
