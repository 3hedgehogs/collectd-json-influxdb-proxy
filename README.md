# collectd-json-influxdb-proxy

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
