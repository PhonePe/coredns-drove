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
example:github.com/santanusinha/coredns
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
  user [USERNAME]
  pass [PASSWORD]
}
~~~
* `URL` - Comma seperated list of drove controllers 
* `TOKEN` - In case drove controllers are using bearer auth Complete Authorization header "Bearer ..."
* `user` `pass` - In case drove is using basic auth

## Ready

This plugin reports readiness to the ready plugin. It will be immediately ready.

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


## Also See

See the [manual](https://coredns.io/manual).
