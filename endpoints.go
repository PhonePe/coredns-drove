package drovedns

import (
	"sync"
	"time"
)

type DroveEndpoints struct {
	appsMutext  *sync.RWMutex
	AppsDB      *DroveAppsResponse
	DroveClient IDroveClient
}

func (dr *DroveEndpoints) setApps(appDB *DroveAppsResponse) {
	dr.appsMutext.Lock()
	dr.AppsDB = appDB
	dr.appsMutext.Unlock()
}

func (dr *DroveEndpoints) getApps() DroveAppsResponse {
	dr.appsMutext.RLock()
	defer dr.appsMutext.RUnlock()
	if dr.AppsDB == nil {
		return DroveAppsResponse{}
	}
	return *dr.AppsDB
}

func (dr *DroveEndpoints) searchApps(questionName string) *DroveApp {
	dr.appsMutext.RLock()
	defer dr.appsMutext.RUnlock()
	for _, app := range dr.AppsDB.Apps {
		if app.Vhost+"." == questionName {
			return &app
		}
	}
	return nil
}

func newDroveEndpoints(client IDroveClient) *DroveEndpoints {
	endpoints := DroveEndpoints{DroveClient: client, appsMutext: &sync.RWMutex{}}
	ticker := time.NewTicker(10 * time.Second)
	done := make(chan bool)
	reload := make(chan bool)
	endpoints.DroveClient.PollEvents(func(eventSummary *DroveEventSummary) {
		if len(eventSummary.EventsCount) > 0 {
			if _, ok := eventSummary.EventsCount["APP_STATE_CHANGE"]; ok {
				log.Debugf("App State Change %+v", eventSummary.EventsCount["APP_STATE_CHANGE"])
				reload <- true
				return
			}
			if _, ok := eventSummary.EventsCount["INSTANCE_STATE_CHANGE"]; ok {
				log.Debugf("Instance State Change %+v", eventSummary.EventsCount["INSTANCE_STATE_CHANGE"])
				reload <- true
				return
			}
		}
	})
	go func() {
		var syncApp = func() {
			DroveQueryTotal.Inc()
			apps, err := endpoints.DroveClient.FetchApps()
			if err != nil {
				DroveQueryFailure.Inc()
				log.Errorf("Error refreshing nodes data %+v", endpoints.AppsDB)
				return
			}

			endpoints.setApps(apps)
		}
		syncApp()
		for {
			select {
			case <-done:
				return
			case <-reload:
				log.Debug("Refreshing Apps due to event change from drove")
				syncApp()
			case _ = <-ticker.C:
				log.Debug("Refreshing Apps data from drove")
				syncApp()
			}
		}
	}()
	return &endpoints
}
