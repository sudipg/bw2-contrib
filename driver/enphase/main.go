package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/immesys/spawnpoint/spawnable"
	"github.com/satori/go.uuid"
	bw2 "gopkg.in/immesys/bw2bind.v5"
)

const rootUUIDStr = "d8b61708-2797-11e6-836b-0cc47a0f7eea"

type TimeseriesReading struct {
	UUID  string
	Time  int64
	Value uint64
}

func (tsr *TimeseriesReading) ToMsgPack() bw2.PayloadObject {
	po, err := bw2.CreateMsgPackPayloadObject(bw2.PONumTimeseriesReading, tsr)
	if err != nil {
		panic(err)
	} else {
		return po
	}
}

func main() {
	// As per the Enphase attribution requirement
	fmt.Println("Powered by Enphase Energy (http://enphase.com)")

	bwClient := bw2.ConnectOrExit("")
	bwClient.OverrideAutoChainTo(true)
	bwClient.SetEntityFromEnvironOrExit()

	params := spawnable.GetParamsOrExit()
	name := params.MustString("name")
	baseURI := params.MustString("svc_base_uri")
	if strings.HasSuffix(baseURI, "/") {
		baseURI = baseURI[:len(baseURI)-1]
	}
	userID := params.MustString("user_id")
	apiKey := params.MustString("api_key")
	sysName := params.MustString("system_name")

	intervalStr := params.MustString("poll_interval")
	pollInterval, err := time.ParseDuration(intervalStr)
	if err != nil {
		fmt.Println("Invalid Poll Interval Length:", pollInterval)
		os.Exit(1)
	}

	svc := bwClient.RegisterService(baseURI+name, "s.Enphase")
	iface := svc.RegisterInterface("enphase1", "i.meter")
	bwClient.SetMetadata(iface.SignalURI("CurrentPower"), "UnitofMeasure", "W")

	rootUUID := uuid.FromStringOrNil(rootUUIDStr)
	currentPowerUUID := uuid.NewV3(rootUUID, "CurrentPower")
	bwClient.SetMetadata(iface.SignalURI("EnergyLifetime"), "UnitofMeasure", "Wh")
	energyLifetimeUUID := uuid.NewV3(rootUUID, "EnergyLifetime")
	bwClient.SetMetadata(iface.SignalURI("EnergyToday"), "UnitofMeasure", "Wh")
	energyTodayUUID := uuid.NewV3(rootUUID, "EnergyToday")

	enphase, err := NewEnphase(apiKey, userID, sysName)
	if err != nil {
		fmt.Println("Failed to initialize Enphase instance:", err.Error())
		os.Exit(1)
	}
	summCh := enphase.PollSummary(pollInterval)
	for summary := range summCh {
		fmt.Printf("Summary: %+v\n", summary)

		currentPowerReading := TimeseriesReading{
			UUID:  currentPowerUUID.String(),
			Time:  time.Now().UnixNano(),
			Value: summary.CurrentPower,
		}
		iface.PublishSignal("CurrentPower", currentPowerReading.ToMsgPack())

		energyLifetimeReading := TimeseriesReading{
			UUID:  energyLifetimeUUID.String(),
			Time:  time.Now().UnixNano(),
			Value: summary.EnergyLifetime,
		}
		iface.PublishSignal("EnergyLifetime", energyLifetimeReading.ToMsgPack())

		energyTodayReading := TimeseriesReading{
			UUID:  energyTodayUUID.String(),
			Time:  time.Now().UnixNano(),
			Value: summary.EnergyToday,
		}
		iface.PublishSignal("EnergyToday", energyTodayReading.ToMsgPack())
	}
}