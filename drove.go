package drovedns

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	FETCH_APP_TIMEOUT    time.Duration = time.Duration(5) * time.Second
	FETCH_EVENTS_TIMEOUT time.Duration = time.Duration(5) * time.Second
	PING_TIMEOUT         time.Duration = time.Duration(5) * time.Second
)

type IDroveClient interface {
	FetchApps() (*DroveAppsResponse, error)
	FetchRecentEvents(syncPoint *CurrSyncPoint) (*DroveEventSummary, error)
	PollEvents(callback func(event *DroveEventSummary))
}
type DroveClient struct {
	EndpointMutex sync.RWMutex
	Endpoint      []EndpointStatus
	Leader        *LeaderController
	AuthConfig    *DroveAuthConfig
	client        *http.Client
}

func NewDroveClient(config DroveConfig) DroveClient {
	controllerEndpoints := strings.Split(config.Endpoint, ",")
	endpoints := make([]EndpointStatus, len(controllerEndpoints))
	for i, e := range controllerEndpoints {
		endpoints[i] = EndpointStatus{e, true, ""}
	}
	tr := &http.Transport{MaxIdleConnsPerHost: 10, TLSClientConfig: &tls.Config{InsecureSkipVerify: config.SkipSSL}}
	httpClient := &http.Client{
		Timeout:   0 * time.Second,
		Transport: tr,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	return DroveClient{Endpoint: endpoints, AuthConfig: &config.AuthConfig, client: httpClient}
}

func (c *DroveClient) Init() error {
	c.updateHealth()
	c.endpointHealth()
	_, err := c.endpoint()
	return err
}
func (c *DroveClient) getRequest(path string, timeout time.Duration, obj any) error {
	host, err := c.endpoint()
	if err != nil {
		return err
	}
	endpoint := host + path
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return err
	}

	setHeaders(*c.AuthConfig, req)
	resp, err := c.client.Do(req)
	if err != nil {
		DroveApiRequests.WithLabelValues("err", "GET", host).Inc()
		return err
	}
	DroveApiRequests.WithLabelValues(strconv.Itoa(resp.StatusCode), "GET", host).Inc()
	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)

	err = decoder.Decode(obj)
	if err != nil {
		return err
	}
	return nil
}

func (c *DroveClient) FetchApps() (*DroveAppsResponse, error) {

	jsonapps := &DroveAppsResponse{}
	err := c.getRequest("/apis/v1/endpoints", FETCH_APP_TIMEOUT, jsonapps)
	return jsonapps, err

}

func (c *DroveClient) FetchRecentEvents(syncPoint *CurrSyncPoint) (*DroveEventSummary, error) {

	var newEventsApiResponse = DroveEventsApiResponse{}
	err := c.getRequest("/apis/v1/cluster/events/summary?lastSyncTime="+fmt.Sprint(syncPoint.LastSyncTime), FETCH_EVENTS_TIMEOUT, &newEventsApiResponse)
	if err != nil {
		return nil, err
	}

	log.Debugf("events response %+v", newEventsApiResponse)
	if newEventsApiResponse.Status != "SUCCESS" {
		return nil, errors.New("Events api call failed. Message: " + newEventsApiResponse.Message)
	}

	syncPoint.LastSyncTime = newEventsApiResponse.EventSummary.LastSyncTime
	return &(newEventsApiResponse.EventSummary), nil
}

func (c *DroveClient) PollEvents(callback func(event *DroveEventSummary)) {
	go func() {

		syncData := CurrSyncPoint{}
		refreshInterval := 2

		ticker := time.NewTicker(time.Duration(refreshInterval) * time.Second)
		for range ticker.C {
			func() {
				log.Debugf("Syncing... at %d", time.Now().UnixMilli())
				syncData.Lock()
				defer syncData.Unlock()
				eventSummary, err := c.FetchRecentEvents(&syncData)
				if err != nil {
					log.Errorf("unable to sync events from drove %s", err.Error())
				} else {
					callback(eventSummary)
				}
			}()
		}
	}()
}

func setHeaders(config DroveAuthConfig, req *http.Request) {
	req.Header.Set("Accept", "application/json")
	if config.User != "" {
		req.SetBasicAuth(config.User, config.Pass)
	}
	if config.AccessToken != "" {
		req.Header.Add("Authorization", config.AccessToken)
	}
}

func leaderController(endpoint string) (*LeaderController, error) {
	if endpoint == "" {
		return nil, fmt.Errorf("Empty leader endpoint")
	}

	parsedUrl, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}

	host, port, splitErr := net.SplitHostPort(parsedUrl.Host)
	if splitErr != nil {
		return nil, splitErr
	}

	iPort, _ := strconv.Atoi(port)
	return &LeaderController{
		Endpoint: endpoint,
		Host:     host,
		Port:     int32(iPort),
	}, nil
}

func (c *DroveClient) endpoint() (string, error) {
	c.EndpointMutex.RLock()
	defer c.EndpointMutex.RUnlock()
	var err error = nil
	if c.Leader == nil || c.Leader.Endpoint == "" {
		return "", errors.New("all endpoints are down")
	}
	return c.Leader.Endpoint, err
}

func (c *DroveClient) refreshLeaderData() {
	var endpoint string
	for _, es := range c.Endpoint {
		DroveControllerHealth.WithLabelValues(es.Endpoint).Set(boolToDouble(es.Healthy))
		if es.Healthy {
			endpoint = es.Endpoint
		}
	}

	if c.Leader == nil || endpoint != c.Leader.Endpoint {
		log.Infof("Looks like master shifted. Will resync app new [%s] old[%+v]", endpoint, c.Leader)
		newLeader, err := leaderController(endpoint)
		if err != nil {
			log.Errorf("Leader struct generation failed %+v", err)
			return
		}
		c.EndpointMutex.Lock()
		defer c.EndpointMutex.Unlock()
		c.Leader = newLeader
		log.Infof("New leader being set leader %+v", c.Leader)
	}
}

func (c *DroveClient) endpointHealth() {
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		for {
			select {
			case <-ticker.C:
				shouldReturn := c.updateHealth()
				if shouldReturn {
					return
				}
			}
		}
	}()
}

func (c *DroveClient) updateHealth() bool {
	log.Debugf("Updating health  %+v", c.Endpoint)
	for i, es := range c.Endpoint {
		ctx, cancel := context.WithTimeout(context.Background(), PING_TIMEOUT)
		defer cancel()
		req, err := http.NewRequestWithContext(ctx, "GET", es.Endpoint+"/apis/v1/ping", nil)
		if err != nil {
			log.Errorf("an error occurred creating endpoint health request %s %s", es.Endpoint, err.Error())
			c.Endpoint[i].Healthy = false
			c.Endpoint[i].Message = err.Error()
			continue
		}
		setHeaders(*c.AuthConfig, req)
		resp, err := c.client.Do(req)
		if err != nil {
			log.Errorf("endpoint is down %s %s", es.Endpoint, err)
			c.Endpoint[i].Healthy = false
			c.Endpoint[i].Message = err.Error()
			continue
		}
		resp.Body.Close()

		if resp.StatusCode != 200 {
			if resp.StatusCode != 400 {
				log.Errorf("Unknown responsecode from drove %d %+v", resp.StatusCode, resp)
			}
			c.Endpoint[i].Healthy = false
			c.Endpoint[i].Message = resp.Status
			continue
		}
		c.Endpoint[i].Healthy = true
		c.Endpoint[i].Message = "OK"
		log.Debugf("Endpoint is healthy host %s", es.Endpoint)
	}
	c.refreshLeaderData()
	return false
}
