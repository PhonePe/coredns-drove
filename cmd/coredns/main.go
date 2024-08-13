package main

//go:generate go run directives_generate.go
//go:generate go run owners_generate.go

import (
	"github.com/coredns/coredns/core/dnsserver"
	_ "github.com/coredns/coredns/core/plugin" // Plug in CoreDNS.
	"github.com/coredns/coredns/coremain"
	_ "phonepe.com/drovedns"
)

var directives = []string{
	"log",
	"drove",
	"forward",
	"ready",
	"whoami",
	"cache",
	"startup",
	"shutdown",
}

func init() {
	dnsserver.Directives = directives
}
func main() {
	coremain.Run()
}
