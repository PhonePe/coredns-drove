package drovedns

import (
	"context"
	"encoding/json"
	"net"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
)

type MockDroveClient struct{}

func (*MockDroveClient) FetchApps() (*DroveAppsResponse, error) {
	apps := &DroveAppsResponse{}
	json.Unmarshal([]byte(`{"status": "ok", "message": "ok", "data":[{"appId": "PS", "vhost": "example.com", "tags": {}, "hosts":[{"host": "host", "port": 1234, "portType": "http"}]}]}`), apps)
	return apps, nil
}

func (*MockDroveClient) FetchRecentEvents(sync *CurrSyncPoint) (*DroveEventSummary, error) {
	eventCount := make(map[string]interface{})
	eventCount["APP_STATE_CHANGE"] = 1
	return &DroveEventSummary{eventCount, 1}, nil
}

func (*MockDroveClient) PollEvents(callback func(event *DroveEventSummary)) {

}

type MockResponseWriter struct {
	dns.ResponseWriter
	validator   func(ms *dns.Msg)
	callCounter int
}

func (w *MockResponseWriter) WriteMsg(res *dns.Msg) error {
	w.callCounter += 1
	w.validator(res)
	return nil

}

func TestServeDNSNotReady(t *testing.T) {
	handler := DroveHandler{DroveEndpoints: &DroveEndpoints{DroveClient: &MockDroveClient{}}}

	writer := &MockResponseWriter{
		validator: func(res *dns.Msg) {
			assert.Equal(t, 1, len(res.Answer), "One Answer should be returned")
			assert.Equal(t, 0, len(res.Extra), "Additional should be empty")
			assert.Equal(t, "host.", res.Answer[0].(*dns.SRV).Target, "'host' should be the target")
			assert.Equal(t, uint16(1234), res.Answer[0].(*dns.SRV).Port, "1234 should be the port")
		}}
	code, err := handler.ServeDNS(context.Background(), writer, &dns.Msg{Question: []dns.Question{dns.Question{Name: "example.com.", Qtype: dns.TypeSRV, Qclass: dns.ClassINET}}})
	assert.NotNil(t, err, "Error should be returned")
	assert.Equal(t, dns.RcodeServerFailure, code, "Failure error code should be returned")
	assert.Equal(t, 0, writer.callCounter, "Message would not be written")

}
func TestServeDNSAnswer(t *testing.T) {
	handler := NewDroveHandler(&MockDroveClient{})
	for !handler.Ready() {
		time.Sleep(1)
	}
	writer := &MockResponseWriter{
		validator: func(res *dns.Msg) {
			assert.Equal(t, 1, len(res.Answer), "One Answer should be returned")
			assert.Equal(t, 0, len(res.Extra), "Additional should be empty")
			assert.Equal(t, "host.", res.Answer[0].(*dns.SRV).Target, "'host' should be the target")
			assert.Equal(t, uint16(1234), res.Answer[0].(*dns.SRV).Port, "1234 should be the port")
		}}
	code, err := handler.ServeDNS(context.Background(), writer, &dns.Msg{Question: []dns.Question{dns.Question{Name: "example.com.", Qtype: dns.TypeSRV, Qclass: dns.ClassINET}}})
	assert.Nil(t, err, "Error should be nil")
	assert.Equal(t, dns.RcodeSuccess, code, "SuccessCode should be returned")
	assert.Less(t, 0, writer.callCounter, "Message should be written")

}

func TestServeDNSAdditional(t *testing.T) {
	handler := NewDroveHandler(&MockDroveClient{})
	for !handler.Ready() {
		time.Sleep(1)
	}
	writer := &MockResponseWriter{
		validator: func(res *dns.Msg) {
			assert.Equal(t, 0, len(res.Answer))
			assert.Equal(t, 1, len(res.Extra))
			assert.Equal(t, "host.", res.Extra[0].(*dns.SRV).Target)
			assert.Equal(t, uint16(1234), res.Extra[0].(*dns.SRV).Port)
		}}
	code, err := handler.ServeDNS(context.Background(), writer, &dns.Msg{Question: []dns.Question{dns.Question{Name: "example.com.", Qtype: dns.TypeA, Qclass: dns.ClassINET}}})
	assert.Nil(t, err, "Error should be nil")
	assert.Equal(t, dns.RcodeSuccess, code, "SuccessCode should be returned")
	assert.Less(t, 0, writer.callCounter, "Message should be written")

}

func TestServeDNSNoMatchingApp(t *testing.T) {
	handler := NewDroveHandler(&MockDroveClient{})
	for !handler.Ready() {
		time.Sleep(1)
	}
	writer := &MockResponseWriter{
		validator: func(res *dns.Msg) {
			assert.Equal(t, 0, len(res.Answer))
			assert.Equal(t, 1, len(res.Extra))
			assert.Equal(t, "host.", res.Extra[0].(*dns.SRV).Target)
			assert.Equal(t, uint16(1234), res.Extra[0].(*dns.SRV).Port)
		}}
	code, err := handler.ServeDNS(context.Background(), writer, &dns.Msg{Question: []dns.Question{dns.Question{Name: "example2.com.", Qtype: dns.TypeA, Qclass: dns.ClassINET}}})
	assert.NotNil(t, err, "Error should be returned")
	assert.Equal(t, dns.RcodeServerFailure, code, "Failure error code should be returned")
	assert.Equal(t, 0, writer.callCounter, "Message should not be Written")

}

type MockHandler struct {
	callCounter int
}

func (h *MockHandler) ServeDNS(c context.Context, rw dns.ResponseWriter, r *dns.Msg) (int, error) {
	h.callCounter += 1
	r.Answer = append(r.Answer, &dns.A{
		A: net.IPv4(1, 1, 1, 1),
	})
	return 0, nil
}
func (h *MockHandler) Name() string {
	return "MOCK"
}
func TestServeDNSForwarding(t *testing.T) {
	handler := NewDroveHandler(&MockDroveClient{})
	mockNextHandler := MockHandler{}
	handler.Next = &mockNextHandler
	for !handler.Ready() {
		time.Sleep(1)
	}
	writer := &MockResponseWriter{
		validator: func(res *dns.Msg) {
			assert.Equal(t, 0, len(res.Answer))
			assert.Equal(t, 1, len(res.Extra))
			assert.Equal(t, "host.", res.Extra[0].(*dns.SRV).Target)
			assert.Equal(t, uint16(1234), res.Extra[0].(*dns.SRV).Port)
		}}
	handler.ServeDNS(context.Background(), writer, &dns.Msg{Question: []dns.Question{dns.Question{Name: "example.com.", Qtype: dns.TypeA, Qclass: dns.ClassINET}}})

	assert.Equal(t, 1, mockNextHandler.callCounter, "Next handler should be called")
}
