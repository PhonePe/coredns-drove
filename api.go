package drovedns

import (
	"fmt"
	"sync"
)

// Host struct
type Host struct {
	Host     string
	Port     int32
	PortType string
}

type HostGroup struct {
	Hosts []Host
	Tags  map[string]string
}

// App struct
type App struct {
	ID     string
	Vhost  string
	Hosts  []Host
	Tags   map[string]string
	Groups map[string]HostGroup
}

type DroveServiceHost struct {
	Host     string `json:"host"`
	Port     int32  `json:"port"`
	PortType string `json:"portType"`
}

// DroveAppsResponse struct for our apps nested with tasks.
type DroveAppsResponse struct {
	Status  string     `json:"status"`
	Apps    []DroveApp `json:"data"`
	Message string     `json:"message"`
}

type DroveApp struct {
	ID    string             `json:"appId"`
	Vhost string             `json:"vhost"`
	Tags  map[string]string  `json:"tags"`
	Hosts []DroveServiceHost `json:"hosts"`
}

type DroveEventSummary struct {
	EventsCount  map[string]interface{} `json:"eventsCount"`
	LastSyncTime int64                  `json:"lastSyncTime"`
}

type DroveEventsApiResponse struct {
	Status       string            `json:"status"`
	EventSummary DroveEventSummary `json:"data"`
	Message      string            `json:"message"`
}

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

type DroveAuthConfig struct {
	User        string
	Pass        string
	AccessToken string
}

func (dc DroveAuthConfig) Validate() error {
	if dc.User == "" && dc.Pass == "" && dc.AccessToken == "" {
		return fmt.Errorf("User-pass or AccessToken should be set")
	}
	if (dc.Pass != "" || dc.User != "") && dc.AccessToken != "" {
		return fmt.Errorf("Both user-pass and access token should not be set")
	}
	return nil
}

type DroveConfig struct {
	Endpoint   string
	AuthConfig DroveAuthConfig
	SkipSSL    bool
}

func (dc DroveConfig) Validate() error {
	if dc.Endpoint == "" {
		return fmt.Errorf("Endpoint Cant be empty")
	}
	return dc.AuthConfig.Validate()
}

func NewDroveConfig() DroveConfig {
	return DroveConfig{SkipSSL: false, AuthConfig: DroveAuthConfig{}}
}
