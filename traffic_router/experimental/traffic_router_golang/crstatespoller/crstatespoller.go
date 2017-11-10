package crstatespoller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/apache/incubator-trafficcontrol/traffic_router/experimental/traffic_router_golang/availableservers"
	"github.com/apache/incubator-trafficcontrol/traffic_router/experimental/traffic_router_golang/crconfig"
	"github.com/apache/incubator-trafficcontrol/traffic_router/experimental/traffic_router_golang/crstates"
	"github.com/apache/incubator-trafficcontrol/traffic_router/experimental/traffic_router_golang/fetch"

	"github.com/apache/incubator-trafficcontrol/lib/go-tc"
)

func updateAvailableServers(crcThs crconfig.Ths, crsThs crstates.Ths, as availableservers.AvailableServers) {
	newAS := map[tc.DeliveryServiceName]map[tc.CacheGroupName][]tc.CacheName{}
	crc := crcThs.Get()
	crs := crsThs.Get()
	for serverNameStr, server := range crc.ContentServers {
		serverName := tc.CacheName(serverNameStr)
		if !crs.Caches[serverName].IsAvailable {
			continue
		}
		if server.CacheGroup == nil {
			fmt.Println("ERROR updateAvailableServers CRConfig server " + serverNameStr + " cachegroup is nil")
			continue
		}
		cgName := tc.CacheGroupName(*server.CacheGroup)
		for dsNameStr, _ := range server.DeliveryServices {
			dsName := tc.DeliveryServiceName(dsNameStr)
			if newAS[dsName] == nil {
				newAS[dsName] = map[tc.CacheGroupName][]tc.CacheName{}
			}
			newAS[dsName][cgName] = append(newAS[dsName][cgName], serverName)
		}
	}
	fmt.Println("updateAvailableServers setting new ", newAS)
	as.Set(newAS)
}

// TODO implement HTTP poller
func Start(fetcher fetch.Fetcher, interval time.Duration, crc crconfig.Ths) (crstates.Ths, availableservers.AvailableServers, error) {
	thsCrs := crstates.NewThs()
	availableServers := availableservers.New()
	prevBts := []byte{}
	prevCrs := (*tc.CRStates)(nil)

	get := func() {
		newBts, err := fetcher.Fetch()
		if err != nil {
			fmt.Println("ERROR CRStates read error: " + err.Error())
			return
		}

		if bytes.Equal(newBts, prevBts) {
			fmt.Println("INFO CRStates unchanged.")
			return
		}

		fmt.Println("INFO CRStates changed.")
		crs := &tc.CRStates{}
		if err := json.Unmarshal(newBts, crs); err != nil {
			fmt.Println("ERROR CRStates unmarshalling: " + err.Error())
			return
		}

		thsCrs.Set(crs)
		prevBts = newBts
		prevCrs = crs

		updateAvailableServers(crc, thsCrs, availableServers) // TODO update AvailableServers when CRStates OR CRConfig is update, via channel and manager goroutine?

		fmt.Println("INFO CRStates set new")
		// TODO update AvailableServers
	}

	get()

	go func() {
		for {
			time.Sleep(interval)
			get()
		}
	}()
	return thsCrs, availableServers, nil
}
