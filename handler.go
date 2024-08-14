// Package example is a CoreDNS plugin that prints "example" to stdout on every packet received.
//
// It serves as an example CoreDNS plugin with numerous code comments.
package drovedns

import (
	"context"
	"fmt"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/request"
	"github.com/miekg/dns"
)

// Example is an example plugin to show how to write a plugin.
type DroveHandler struct {
	DroveEndpoints *DroveEndpoints
	Next           plugin.Handler
}

func NewDroveHandler(droveClient IDroveClient) *DroveHandler {
	return &DroveHandler{DroveEndpoints: newDroveEndpoints(droveClient)}

}
func (e *DroveHandler) Name() string { return "drove" }

func (e *DroveHandler) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {

	a := new(dns.Msg)
	if e.DroveEndpoints.getApps() == nil {
		return dns.RcodeServerFailure, fmt.Errorf("Drove DNS not ready")
	}
	if len(r.Question) == 0 {
		return plugin.NextOrFailure(e.Name(), e.Next, ctx, w, r)
	}
	app := e.DroveEndpoints.searchApps(r.Question[0].Name)
	if app != nil {

		a.SetReply(r)
		a.Authoritative = true

		state := request.Request{W: w, Req: r}

		srv := make([]dns.RR, len(app.Hosts))

		for i, h := range app.Hosts {
			srv[i] = &dns.SRV{Hdr: dns.RR_Header{Name: state.QName(), Rrtype: dns.TypeSRV, Class: state.QClass(), Ttl: 30},
				Port:     uint16(h.Port),
				Target:   h.Host + ".",
				Weight:   1,
				Priority: 1,
			}
		}

		if state.QType() == dns.TypeSRV {
			a.Answer = srv
		} else {
			a.Extra = srv
		}
	}

	if len(a.Answer) > 0 || len(a.Extra) > 0 {
		if e.Next != nil {
			return plugin.NextOrFailure(e.Name(), e.Next, ctx, &CombiningResponseWriter{w, a}, r)
		}
		w.WriteMsg(a)
		return dns.RcodeSuccess, nil

	}

	// Call next plugin (if any).
	return plugin.NextOrFailure(e.Name(), e.Next, ctx, w, r)
}

// Name implements the Handler interface.
type CombiningResponseWriter struct {
	dns.ResponseWriter
	answer *dns.Msg
}

func (w *CombiningResponseWriter) WriteMsg(res *dns.Msg) error {

	res.Answer = append(res.Answer, w.answer.Answer...)
	res.Extra = append(res.Extra, w.answer.Extra...)
	return w.ResponseWriter.WriteMsg(res)

}
