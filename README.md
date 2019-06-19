# zap-syslog

[![GoDoc](https://godoc.org/github.com/imperfectgo/zap-syslog?status.svg)](https://godoc.org/github.com/imperfectgo/zap-syslog)

Syslog ([RFC5424]) encoder & syncer for [zap](https://github.com/uber-go/zap).

This repo is forked from ([imperfectgo/zap-syslog]). Changes are as follows:

* make BOM optional via configuration
* put proc-id into square brackets

[RFC5424]: https://tools.ietf.org/html/rfc5424
[imperfectgo/zap-syslog]: https://godoc.org/github.com/imperfectgo/zap-syslog
