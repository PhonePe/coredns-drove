package drovedns

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAppFetch(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	mux.HandleFunc("/apis/v1/ping", func(rw http.ResponseWriter, req *http.Request) {
		// Test request parameters
		// Send response to be tested
		// rw.Write([]byte(`OK`))
		rw.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("/apis/v1/endpoints", func(rw http.ResponseWriter, req *http.Request) {
		// Test request parameters
		// Send response to be tested
		// rw.Write([]byte(`OK`))
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)

		fmt.Fprint(rw, `{"status": "ok", "message": "ok", "data":[{"appId": "PS", "vhost": "ps.blah", "tags": {}, "hosts":[{"host": "host", "port": 1234, "portType": "http"}]}]}`)
	})
	// Close the server when test finishes
	defer server.Close()

	// Use Client & URL from our local test server
	client := NewDroveClient(DroveConfig{Endpoint: server.URL, AuthConfig: DroveAuthConfig{AccessToken: ""}})
	client.Init()
	assert.NotNil(t, client.Leader)

	apps, err := client.FetchApps()
	assert.Nil(t, err)
	assert.Equal(t, len(apps.Apps), 1)
	assert.Equal(t, len(apps.Apps[0].Hosts), 1)
}

func TestLeaderElection(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	mux.HandleFunc("/apis/v1/ping", func(rw http.ResponseWriter, req *http.Request) {
		// Test request parameters
		// Send response to be tested
		// rw.Write([]byte(`OK`))
		rw.WriteHeader(http.StatusForbidden)
	})

	mux2 := http.NewServeMux()
	server2 := httptest.NewServer(mux2)
	mux2.HandleFunc("/apis/v1/ping", func(rw http.ResponseWriter, req *http.Request) {
		// Test request parameters
		// Send response to be tested
		// rw.Write([]byte(`OK`))
		rw.WriteHeader(http.StatusOK)
	})

	// Use Client & URL from our local test server
	client := NewDroveClient(DroveConfig{Endpoint: server.URL, AuthConfig: DroveAuthConfig{AccessToken: ""}})
	client.Init()
	assert.Nil(t, client.Leader)

	client1 := NewDroveClient(DroveConfig{Endpoint: "http://random.blah.endpoint.non-existent", AuthConfig: DroveAuthConfig{AccessToken: ""}})
	client1.Init()
	assert.Nil(t, client1.Leader)

	client2 := NewDroveClient(DroveConfig{Endpoint: fmt.Sprintf("%s,%s", server.URL, server2.URL), AuthConfig: DroveAuthConfig{AccessToken: ""}})
	client2.Init()
	assert.NotNil(t, client2.Leader)
	assert.Equal(t, server2.URL, client2.Leader.Endpoint)
	time.Sleep(2 * time.Second)
	assert.NotNil(t, client2.Leader)
	assert.Equal(t, server2.URL, client2.Leader.Endpoint)

}

func TestLeaderFailover(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	var status1, status2 atomic.Int64
	status1.Store(200)
	status2.Store(400)

	mux.HandleFunc("/apis/v1/ping", func(rw http.ResponseWriter, req *http.Request) {
		// Test request parameters
		// Send response to be tested
		// rw.Write([]byte(`OK`))
		rw.WriteHeader(int(status1.Load()))
	})

	mux2 := http.NewServeMux()
	server2 := httptest.NewServer(mux2)
	mux2.HandleFunc("/apis/v1/ping", func(rw http.ResponseWriter, req *http.Request) {
		// Test request parameters
		// Send response to be tested
		// rw.Write([]byte(`OK`))
		rw.WriteHeader(int(status2.Load()))
	})

	client := NewDroveClient(DroveConfig{Endpoint: fmt.Sprintf("%s,%s", server.URL, server2.URL), AuthConfig: DroveAuthConfig{AccessToken: ""}})
	client.Init()
	assert.NotNil(t, client.Leader)
	endpoint, err := client.endpoint()
	assert.Equal(t, server.URL, endpoint)
	status1.Store(400)
	status2.Store(200)
	time.Sleep(4 * time.Second)
	endpoint, err = client.endpoint()
	assert.Nil(t, err)

	assert.Equal(t, server2.URL, endpoint)
}
