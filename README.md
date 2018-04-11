# collectd-json-influxdb-proxy

<!-- markdownlint-disable MD013 -->
[![Build Status](https://secure.travis-ci.org/dex4er/collectd-json-influxdb-proxy.svg)](http://travis-ci.org/dex4er/collectd-json-influxdb-proxy)
<!-- markdownlint-enable MD013 -->

Translate collectd JSON HTTP request to Influx Data line protocol

## Requirements

* Go >= 1.9
* [Glide](https://github.com/Masterminds/glide)

## Compilation

```console
glide install
go build .
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

## Example request

```console
curl -H "Content-Type: application/json" -X POST -d '[{"values":  [1901474177],"dstypes":["counter"],"dsnames":["value"],"time":1280959128,"interval":10,"host":"leeloo.octo.it","plugin":"cpu","plugin_instance": "0","type":"cpu", "type_instance":"idle"}]' http://localhost:5826/
curl http://localhost:8080/debug/vars
```



## Copyright

(c) 2018 Piotr Roszatycki <mailto:piotr.roszatycki@gmail.com> MIT

Based on previous work
<https://github.com/dex4er/perl-collectd-json-influxdb-proxy>
