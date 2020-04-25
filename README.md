# profefe - Collect profiling data for long-term analysis

[![Build Status](https://travis-ci.org/profefe/profefe.svg?branch=master)](https://travis-ci.org/profefe/profefe)
[![Go Report Card](https://goreportcard.com/badge/github.com/profefe/profefe)](https://goreportcard.com/report/github.com/profefe/profefe)
[![Docker Pulls](https://img.shields.io/docker/pulls/profefe/profefe.svg)][hub.docker]
[![MIT licensed](https://img.shields.io/badge/license-MIT-blue.svg)](https://raw.githubusercontent.com/profefe/profefe/master/LICENSE)

*profefe* continuously collects profiling data from a running Go application and provides an API for querying
the profiling samples base on metadata associated with the application.

## Why Continuous Profiling?

The blog-post "[Continuous Profiling and Go](https://medium.com/@tvii/continuous-profiling-and-go-6c0ab4d2504b)" describes
the motivation behind the project:

> With the increase in momentum around the term “observability” over the last few years, there is a common misconception
> amongst the developers, that observability is exclusively about _metrics_, _logs_ and _tracing_ (a.k.a. “three pillars of observability”)
> [..] With metrics and tracing, we can see the system on a macro-level. Logs only cover the known parts of the system.
> Performance profiling is yet another signal that uncovers the micro-level of a system and continuous profiling allows
> observing how the components of the application or the infrastructure it runs in, influence the overall system.

## How does it work?

See [Design Docs](DESIGN.md) documentation.

## Quickstart

To build and start profefe collector, run:

```
> make
> ./BUILD/profefe -addr=locahost:10100 -storage-type=badger -badger.dir=/tmp/profefe-data

2019-06-06T00:07:58.499+0200    info    profefe/main.go:86    server is running    {"addr": ":10100"}
```

`./BUILD/profefe -help` will print the list of available options.

---

You can build a docker image with the collector, by running the command:

```
> make docker-image
```

The documentation about running profefe in docker can be found in [contrib/docker/README.md](./contrib/docker/README.md).

---

The project includes a fork of [Google Stackdriver Profiler's example application][5], modified to use *profefe agent*,
that sends profiling data to profefe collector.

Run the following command in a separate terminal window, to start the example:

```
> go run ./examples/hotapp/main.go
```

After a brief period, the application will start sending CPU profiles to running profefe collector

```
send profile: http://localhost:10100/api/0/profiles?service=hotapp-service&labels=version=1.0.0&type=cpu
send profile: http://localhost:10100/api/0/profiles?service=hotapp-service&labels=version=1.0.0&type=cpu
send profile: http://localhost:10100/api/0/profiles?service=hotapp-service&labels=version=1.0.0&type=cpu
```

To query stored profiling data, make an HTTP call to profefe collector API ([_see documentation for collector's HTTP API_](#http-api)):

```
> go tool pprof 'http://localhost:10100/api/0/profiles/merge?service=hotapp-service&type=cpu&from=2019-05-30T11:49:00&to=2019-05-30T12:49:00&labels=version=1.0.0'

Fetching profile over HTTP from http://localhost:10100/api/0/profiles...
Saved profile in /Users/varankinv/pprof/pprof.samples.cpu.001.pb.gz
Type: cpu
Entering interactive mode (type "help" for commands, "o" for options)
(pprof) top
Showing nodes accounting for 43080ms, 99.15% of 43450ms total
Dropped 53 nodes (cum <= 217.25ms)
Showing top 10 nodes out of 12
      flat  flat%   sum%        cum   cum%
   42220ms 97.17% 97.17%    42220ms 97.17%  main.load
     860ms  1.98% 99.15%      860ms  1.98%  runtime.nanotime
         0     0% 99.15%    21050ms 48.45%  main.bar
         0     0% 99.15%    21170ms 48.72%  main.baz
         0     0% 99.15%    42250ms 97.24%  main.busyloop
         0     0% 99.15%    21010ms 48.35%  main.foo1
         0     0% 99.15%    21240ms 48.88%  main.foo2
         0     0% 99.15%    42250ms 97.24%  main.main
         0     0% 99.15%    42250ms 97.24%  runtime.main
         0     0% 99.15%     1020ms  2.35%  runtime.mstart
```

Note, above we requested all profiling data associated with the given meta-information (service and time period),
as a single *merged* profile.

profefe includes an experimental tool, that allows importing existing pprof data into the collector.
While the profefe collector is running:

```
> ./scripts/pprof_import.sh --service service1 --label region=europe-west3 --label host=backend1 --type cpu -- path/to/cpu.prof

uploading service1-cpu-backend1-20190313-0948Z.prof...OK
```

## HTTP API

### Save pprof-formatted profile

```
POST /api/0/profiles?service=<service>&type=[cpu|heap|...]&labels=<key=value,key=value>
body pprof.pb.gz

< 200 OK
<
{
  "code": 200,
  "body": {
    "id": <id>,
    "type": <type>,
    ···
  }
}
```

- `service` — service name (string)
- `type` — profile type ("cpu", "heap", "block", "mutex", "goroutine", "threadcreate", or "other")
- `labels` — a set of key-value pairs, e.g. "region=europe-west3,dc=fra,ip=1.2.3.4,version=1.0" (Optional)

#### Save runtime execution traces (experimental)

Go's [runtime traces](https://golang.org/pkg/runtime/trace/) are a special case of profiling data, that can be stored
and queried with profefe.

Currently, profefe doesn't support extracting the timestamp of when the trace was created. Client may provide
this information via `created_at` parameter, see below.

```
POST /api/0/profiles?service=<service>&type=trace&created_at=<created_at>&labels=<key=value,key=value>
body trace.out

< 200 OK
<
{
  "code": 200,
  "body": {
    "id": <id>,
    "type": "trace",
    ···
  }
}
```

- `service` — service name (string)
- `type` — profile type ("trace")
- `created_at` — trace profile creation time, e.g. "2006-01-02T15:04:05" (defaults to server's current time)
- `labels` — a set of key-value pairs, e.g. "region=europe-west3,dc=fra,ip=1.2.3.4,version=1.0" (Optional)

### Query saved meta information

```
GET /api/0/profiles?service=<service>&type=<type>&from=<created_from>&to=<created_to>&labels=<key=value,key=value>

< 200 OK
<
{
  "code": 200,
  "body": [
    {
      "id": <id>,
      "type": <type>
    },
    ···
  ]
}
```

- `service` — service name
- `type` — profile type ("cpu", "heap", "block", "mutex", "goroutine", "threadcreate", "trace", "other")
- `created_from`, `created_to` — a time window between which profiling data was collected, e.g. "from=2006-01-02T15:04:05"
- `labels` — a set of key-value pairs

### Query saved profiling data returning it as a single merged profile

```
GET /api/0/profiles/merge?service=<service>&type=<type>&from=<created_from>&to=<created_to>&labels=<key=value,key=value>

< 200 OK
< pprof.pb.gz
```

Request parameters are the same as for querying meta information.

*Note, merging runtime traces is not supported.*

### Return individual profiling data

```
GET /api/0/profiles/<id>

< 200 OK
< pprof.pb.gz
```

- `id` - id of stored profile, returned with the request for meta information above

#### Merge a set of individual profiles into a single profile

```
GET /api/0/profiles/<id1>+<id2>+...

< 200 OK
< pprof.pb.gz
```

- `id1`, `id2` - ids of stored profiles

*Note, merging is possible only for profiles of the same type.*

### Get services for which profiling data is stored

```
GET /api/0/services

< 200 OK
<
{
  "code": 200,
  "body": [
    <service1>,
    ···
  ]
}
```

### Get profefe server version

```
GET /api/0/version

< 200 OK
<
{
  "code": 200,
  "body": {
    "version": <version>,
    "commit": <git revision>,
    "build_time": <build timestamp>"
  }
}
```

## FAQ

### Does continuous profiling affect the performance of the production?

Profiling always comes with some costs. Go collects sampling-based profiling data and for the most of the applications
the real overhead is small enough (refer to "[Can I profile my production services](https://golang.org/doc/diagnostics.html#profiling)"
from Go's Diagnostics documentation).

To reduce the costs, users can adjust the frequency of profiles collection, e.g. collect 10 seconds of CPU profiles every 5 minutes.

[profefe-agent](https://godoc.org/github.com/profefe/profefe/agent) tries to reduce the overhead further by adding a small
jiggling in-between the profiles collections. This distributes the total profiling overhead, making sure that not all instances
of the application's cluster are being profiled at the same time.

### Can I use profefe with non-Go projects?

profefe collects [pprof-formatted](https://github.com/google/pprof/blob/master/README.md) profiling data. The format is used by Go profiler,
but thrid-party profilers for other programming languages support of the format. For example, [`google/pprof-nodejs`](https://github.com/google/pprof-nodejs) for Node.js,
[`tikv/pprof-rs`](https://github.com/tikv/pprof-rs) for Rust, [`arnaud-lb/php-memory-profiler`](https://github.com/arnaud-lb/php-memory-profiler) for PHP, etc.

Integrating those is the subject of building a transport layer between the profiler and profefe.

## Further reading

While the topic of continuous profiling in the production is a bit unrepresented in the public internet, some
research or commercial projects are already exist

- [Stackdriver profiler](https://cloud.google.com/profiler/)
- [Google-Wide Profiling: A Continuous Profiling Infrastructure for Data Centers](https://ai.google/research/pubs/pub36575) (paper)
- [StackImpact](https://stackimpact.com/docs/go-profiling/)
- [conprof](https://github.com/conprof/conprof)
- [Opsian - Continuous Profiling for JVM](https://opsian.com) (provides on-premises plan for enterprise customers)
- [Liveprof - Continuous Profiling for PHP](https://habr.com/ru/company/badoo/blog/436364/) (RUS)
- [FlameScope](https://github.com/Netflix/flamescope)

*The project is still in its early state. Feedback and contribution are very welcome.*

## License

MIT

[hub.docker]: https://hub.docker.com/r/profefe/profefe
[3]: https://stackimpact.com/
[5]: https://github.com/GoogleCloudPlatform/golang-samples/tree/master/profiler/hotapp
[pprof]: https://github.com/google/pprof/
