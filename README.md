# collectd-json-influxdb-proxy

<!-- markdownlint-disable MD013 -->
[![Build Status](https://secure.travis-ci.org/dex4er/collectd-json-influxdb-proxy.svg)](http://travis-ci.org/dex4er/collectd-json-influxdb-proxy)
<!-- markdownlint-enable MD013 -->

Translate collectd JSON HTTP request to Influx Data line protocol

## Requirements

* Go >= 1.9
* Glide

## Compilation

```console
glide up
go build *.go
```

## Configuration for collectd

```xml
<Plugin "write_http">
    <Node "collectd-json-influxdb-proxy">
       URL "http://localhost:5826/"
       Format "JSON"
       BufferSize 129024
       Timeout 5000
    </Node>
</Plugin>
```

## Running

```console
./collectd-json-influxdb-proxy
```

## Copyright

(c) 2018 Piotr Roszatycki <mailto:piotr.roszatycki@gmail.com> MIT

Based on previous work
<https://github.com/dex4er/perl-collectd-json-influxdb-proxy>
