package drovedns

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

// DroveApps struct for our apps nested with tasks.
type DroveApps struct {
	Status string `json:"status"`
	Apps   []struct {
		ID    string             `json:"appId"`
		Vhost string             `json:"vhost"`
		Tags  map[string]string  `json:"tags"`
		Hosts []DroveServiceHost `json:"hosts"`
	} `json:"data"`
	Message string `json:"message"`
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
