# collectd-json-influxdb-proxy

<!-- markdownlint-disable MD013 -->
[![Build Status](https://secure.travis-ci.org/3hedgehogs/collectd-json-influxdb-proxy.svg)](http://travis-ci.org/3hedgehogs/collectd-json-influxdb-proxy)
<!-- markdownlint-enable MD013 -->

Translate collectd JSON HTTP request to Influx Data line protocol

## Requirements

* Go >= 1.9

## Compilation

```console
dep ensure
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
curl http://localhost:5826/debug/vars
```



## Copyright

(c) 2018 Serguei Poliakov <mailto:serguei.poliakov@gmail.com> MIT  
(c) 2018 Piotr Roszatycki <mailto:piotr.roszatycki@gmail.com> MIT

Forked from
<https://github.com/dex4er/collectd-json-influxdb-proxy/>
Based on
<https://github.com/dex4er/perl-collectd-json-influxdb-proxy>
