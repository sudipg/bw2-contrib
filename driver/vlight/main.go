package main

import (
	"fmt"
	"github.com/immesys/spawnpoint/spawnable"
	bw2 "gopkg.in/immesys/bw2bind.v5"
	"time"
)

const (
	PONUM = "2.1.1.1"
)

type Reading struct {
	Time int64
	State bool
	Brightness int
}

func (r *Reading) ToMsgPackPO() (bo bw2.PayloadObject) {
	po, err := bw2.CreateMsgPackPayloadObject(bw2.FromDotForm(PONUM), r)
	if err != nil {
		panic(err)
	}
	return po
}

func main() {
	bwClient := bw2.ConnectOrExit("")

	params := spawnable.GetParamsOrExit()
	bwClient.OverrideAutoChainTo(true)
	bwClient.SetEntityFromEnvironOrExit()

	baseuri := params.MustString("svc_base_uri")
	poll_interval := params.MustString("poll_interval")
	pollInt, err := time.ParseDuration(poll_interval)
	if err != nil {
		panic(err)
	}

	service := bwClient.RegisterService(baseuri, "s.vlight")
	iface := service.RegisterInterface("vlight", "i.xbos.light")

	params.MergeMetadata(bwClient)

	v := NewVlight()

	iface.SubscribeSlot("state", func(msg *bw2.SimpleMessage) {
		po := msg.GetOnePODF(PONUM)
		if po == nil {
			fmt.Println("Received actuation command without valid PO, dropping")
			return
		} else if len(po.GetContents()) < 1 {
			fmt.Println("Received actuation command with invalid PO, dropping")
			return
		}

		msgpo, err := bw2.LoadMsgPackPayloadObject(po.GetPONum(), po.GetContents())
		if err != nil {
			fmt.Println(err)
			return
		}

		var data Info
		err = msgpo.ValueInto(&data)
		if err != nil {
			fmt.Println(err)
			return
		}

		v.ActuateLight(data.State)
		v.SetBrightness(data.Brightness)
	})

	for {
		info := v.GetStatus()
		timestamp := time.Now().UnixNano()
		msg := Reading {
			Time: timestamp,
			State: info.State,
			Brightness: info.Brightness,
		}

		iface.PublishSignal("info", msg.ToMsgPackPO())
		time.Sleep(pollInt)
	}
}
