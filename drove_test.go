package drovedns

import (
	"fmt"
	"net/http"
	"net/http/httptest"
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
	client := NewDroveClient(server.URL, DroveAuthConfig{AccessToken: ""})
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
	client := NewDroveClient(fmt.Sprintf("%s", server.URL), DroveAuthConfig{AccessToken: ""})
	client.Init()
	assert.Nil(t, client.Leader)

	client = NewDroveClient("http://random.blah.endpoint.non-existent", DroveAuthConfig{AccessToken: ""})
	client.Init()
	assert.Nil(t, client.Leader)

	client = NewDroveClient(fmt.Sprintf("%s,%s", server.URL, server2.URL), DroveAuthConfig{AccessToken: ""})
	client.Init()
	assert.NotNil(t, client.Leader)
	assert.Equal(t, server2.URL, client.Leader.Endpoint)
	time.Sleep(2 * time.Second)
	assert.NotNil(t, client.Leader)
	assert.Equal(t, server2.URL, client.Leader.Endpoint)

}

func TestLeaderFailover(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	status1, status2 := 200, 400
	mux.HandleFunc("/apis/v1/ping", func(rw http.ResponseWriter, req *http.Request) {
		// Test request parameters
		// Send response to be tested
		// rw.Write([]byte(`OK`))
		rw.WriteHeader(status1)
	})

	mux2 := http.NewServeMux()
	server2 := httptest.NewServer(mux2)
	mux2.HandleFunc("/apis/v1/ping", func(rw http.ResponseWriter, req *http.Request) {
		// Test request parameters
		// Send response to be tested
		// rw.Write([]byte(`OK`))
		rw.WriteHeader(status2)
	})

	client := NewDroveClient(fmt.Sprintf("%s,%s", server.URL, server2.URL), DroveAuthConfig{AccessToken: ""})
	client.Init()
	assert.NotNil(t, client.Leader)
	assert.Equal(t, server.URL, client.Leader.Endpoint)
	status1, status2 = 400, 200
	time.Sleep(4 * time.Second)
	assert.NotNil(t, client.Leader)
	assert.Equal(t, server2.URL, client.Leader.Endpoint)
}
