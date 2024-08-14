# DroveDNS

## Name

*drovedns* - DNS answers for apps runing on drove

## Description

The plugins can be used to implement service discovery through dns for apps running on drove. Plugin answers with SRV record for container discovery, optionally forward plugin can be used to add A record of drove-gateway

## Compilation

This package will always be compiled as part of CoreDNS and not in a standalone way. It will require you to use `go get` or as a dependency on [plugin.cfg](https://github.com/coredns/coredns/blob/master/plugin.cfg).

The [manual](https://coredns.io/manual/toc/#what-is-coredns) will have more information about how to configure and extend the server with external plugins.

A simple way to consume this plugin, is by adding the following on [plugin.cfg](https://github.com/coredns/coredns/blob/master/plugin.cfg), and recompile it as [detailed on coredns.io](https://coredns.io/2017/07/25/compile-time-enabling-or-disabling-plugins/#build-with-compile-time-configuration-file).

~~~
drove:github.com/PhonePe/coredns-drove
~~~

Put this early in the plugin list, so that *drovedns* is executed before any of the other plugins.

After this you can compile coredns by:

``` sh
go generate
go build
```

Or you can instead use make:

``` sh
make
```

## Syntax

~~~ txt
drovedns {
  endpoint [URL]
  accesstoken [TOKEN]
  user_pass [USERNAME] [PASSWORD]
  skip_ssl_check
}
~~~
* `URL` - Comma seperated list of drove controllers 
* `TOKEN` - In case drove controllers are using bearer auth Complete Authorization header "Bearer ..."
* `user` `pass` - In case drove is using basic auth
* `skip_ssl_check` - To skip client side ssl certificate validation

## Ready

This plugin reports readiness to the ready plugin. It will be immediately ready.

## Metrics

If monitoring is enabled (via the *prometheus* plugin) then the following metrics are exported:

* `coredns_drove_controller_health{host}` - Exports the health of controller at any given point
The following are client level metrics to monitor apiserver request latency & status codes. `verb` identifies the apiserver [request type](https://kubernetes.io/docs/reference/using-api/api-concepts/#single-resource-api) and `host` denotes the apiserver endpoint.
* `coredns_drove_sync_total` - captures total app syncs from drove.
* `coredns_drove_sync_failure` - captures failed app syncs from drove.
* `coredns_drove_api_total{status_code, method, host}` - captures drove request grouped by `status_code`, `method` & `host`.



## Examples

In this configuration, we resolve queries through the plugin and enrich the answers with A record from servers listed in local resolv.conf

~~~ corefile
example.drove.gateway.com {
  drovedns {
    endpoint "http://drove-control001.example.com:8080,http://drove-control002.example.com:8080"
    accesstoken "Bearer foo"
  }
  forward . /etc/resolv.conf
}
~~~

## Docker
Docker image containing coredns compiled with the plugin are available on ghcr.

~~~ bash
docker run  -p1053:1053/udp -p1053:1053 \
    -e DROVE_ENDPOINT="https://drovecontrol001.exmaple.com:8080,https://drovecontrol002.exmaple.com:8080,https://drovecontrol003.exmaple.com:8080"  \
    -e DROVE_USERNAME="<USERNAME>" -e DROVE_PASSWORD="<PASSWORD>"  \
    -it ghcr.io/phonepe/coredns-drove:<VERSION>
~~~

Alternatively you can provide your own Corefile

~~~ bash
docker run  -p1053:1053/udp -p1053:1053 \
    -v /path/to/Corefile:/opt/Corefile  \
    -it ghcr.io/phonepe/coredns-drove:<VERSION>
~~~

## Also See

See the [manual](https://coredns.io/manual).
