package drovedns

import (
	"sync"
	"time"
)

type DroveEndpoints struct {
	appsMutex   *sync.RWMutex
	AppsDB      *DroveAppsResponse
	DroveClient IDroveClient
	AppsByVhost map[string]DroveApp
}

func (dr *DroveEndpoints) setApps(appDB *DroveAppsResponse) {
	var appsByVhost map[string]DroveApp = make(map[string]DroveApp)
	if appDB != nil {
		for _, app := range appDB.Apps {
			appsByVhost[app.Vhost+"."] = app
		}
	}
	dr.appsMutex.Lock()
	dr.AppsDB = appDB
	dr.AppsByVhost = appsByVhost
	dr.appsMutex.Unlock()
}

func (dr *DroveEndpoints) getApps() *DroveAppsResponse {
	dr.appsMutex.RLock()
	defer dr.appsMutex.RUnlock()
	if dr.AppsDB == nil {
		return nil
	}
	return dr.AppsDB
}

func (dr *DroveEndpoints) searchApps(questionName string) *DroveApp {
	dr.appsMutex.RLock()
	defer dr.appsMutex.RUnlock()
	if dr.AppsByVhost == nil {
		return nil
	}
	if app, ok := dr.AppsByVhost[questionName]; ok {
		return &app
	}
	return nil
}

func newDroveEndpoints(client IDroveClient) *DroveEndpoints {
	endpoints := DroveEndpoints{DroveClient: client, appsMutex: &sync.RWMutex{}}
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
				log.Errorf("Error refreshing nodes data")
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
