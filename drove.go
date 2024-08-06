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

type LeaderController struct {
	Endpoint string
	Host     string
	Port     int32
}

type EndpointStatus struct {
	Endpoint string
	Healthy  bool
	Message  string
}

type CurrSyncPoint struct {
	sync.RWMutex
	LastSyncTime int64
}
type IDroveClient interface {
	FetchApps() (*DroveApps, error)
	FetchRecentEvents(syncPoint *CurrSyncPoint) (*DroveEventSummary, error)
	PollEvents(callback func(event *DroveEventSummary))
}
type DroveClient struct {
	Endpoint   []EndpointStatus
	Leader     *LeaderController
	AuthConfig *DroveAuthConfig
	client     *http.Client
}

type DroveAuthConfig struct {
	User        string
	Pass        string
	AccessToken string
}

func NewDroveClient(endpoint string, authConfig DroveAuthConfig) DroveClient {
	controllerEndpoints := strings.Split(endpoint, ",")
	endpoints := make([]EndpointStatus, len(controllerEndpoints))
	for i, e := range controllerEndpoints {
		endpoints[i] = EndpointStatus{e, true, ""}
	}
	tr := &http.Transport{MaxIdleConnsPerHost: 10, TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	httpClient := &http.Client{
		Timeout:   0 * time.Second,
		Transport: tr,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	return DroveClient{Endpoint: endpoints, AuthConfig: &authConfig, client: httpClient}
}

func (c *DroveClient) Init() error {
	c.updateHealth()
	c.endpointHealth()
	_, err := c.endpoint()
	return err
}
func (c *DroveClient) getRequest(endpoint string, timeout int, obj any) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return err
	}
	setHeaders(*c.AuthConfig, req)
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)

	err = decoder.Decode(obj)
	if err != nil {
		return err
	}
	return nil
}

func (c *DroveClient) FetchApps() (*DroveApps, error) {
	endpoint, err := c.endpoint()
	if err != nil {
		return nil, err
	}

	var timeout int = 5
	jsonapps := &DroveApps{}
	err = c.getRequest(endpoint+"/apis/v1/endpoints", timeout, jsonapps)
	return jsonapps, err

}

func (c *DroveClient) FetchRecentEvents(syncPoint *CurrSyncPoint) (*DroveEventSummary, error) {
	endpoint, err := c.endpoint()
	if err != nil {
		return nil, err
	}
	var newEventsApiResponse = DroveEventsApiResponse{}
	err = c.getRequest(endpoint+"/apis/v1/cluster/events/summary?lastSyncTime="+fmt.Sprint(syncPoint.LastSyncTime), 5, &newEventsApiResponse)
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
	var err error = nil
	if c.Leader == nil || c.Leader.Endpoint == "" {
		return "", errors.New("all endpoints are down")
	}
	return c.Leader.Endpoint, err
}

func (c *DroveClient) refreshLeaderData() {
	var endpoint string
	for _, es := range c.Endpoint {
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
				//logger.WithFields(logrus.Fields{
				//            "health": health,
				//}).Info("Reloading endpoint health")
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
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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
