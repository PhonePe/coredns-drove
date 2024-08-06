package drovedns

import (
	"sync"
	"time"
)

type DroveEndpoints struct {
	appsMutext  *sync.RWMutex
	AppsDB      *DroveApps
	DroveClient IDroveClient
}

func (dr *DroveEndpoints) setApps(appDB *DroveApps) {
	dr.appsMutext.Lock()
	dr.AppsDB = appDB
	dr.appsMutext.Unlock()
}

func (dr *DroveEndpoints) getApps() DroveApps {
	dr.appsMutext.RLock()
	defer dr.appsMutext.RUnlock()
	if dr.AppsDB == nil {
		return DroveApps{}
	}
	return *dr.AppsDB

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
			apps, err := endpoints.DroveClient.FetchApps()
			if err != nil {
				log.Errorf("Error refreshing nodes data %+v", endpoints.AppsDB)
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
