package drovedns

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRaceCondidtion(t *testing.T) {

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

	mux.HandleFunc("/apis/v1/cluster/events/summary", func(rw http.ResponseWriter, req *http.Request) {
		// Test request parameters
		// Send response to be tested
		// rw.Write([]byte(`OK`))
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)

		fmt.Fprint(rw, `{"status": "SUCCESS", "message": "ok", "data":{"eventsCount":{"APP_STATE_CHANGE": 1}, "lastSyncTime": 1}}`)
	})
	// Close the server when test finishes
	defer server.Close()

	// Use Client & URL from our local test server
	client := NewDroveClient(DroveConfig{Endpoint: server.URL, AuthConfig: DroveAuthConfig{AccessToken: ""}})
	client.Init()
	underTest := newDroveEndpoints(&client)
	go func() {
		for i := 0; i < 100; i++ {
			go func() {
				apps := underTest.getApps()
				if apps == nil {
					return
				}
				for i, _ := range apps.Apps {
					t.Logf("%+v", apps.Apps[i].Hosts)
				}
			}()
		}
	}()

	go func() {
		for i := 0; i < 100; i++ {
			go func() {
				underTest.setApps(&DroveAppsResponse{})

			}()
		}
	}()
	time.Sleep(1)
}
