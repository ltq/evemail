// Copyright 2012 EVALGO Community Developers.
// See the LICENSE file at the top-level directory of this distribution
// and at http://opensource.org/licenses/bsd-license.php

package main

import (
	"github.com/evalgo/evapi"
	"github.com/evalgo/evapplication"
	"github.com/evalgo/evemail"
	"log"
	"net"
	"net/http"
	"net/rpc"
)

func main() {
	gobObjects := evapplication.NewEVApplicationGobRegisteredObjects()
	gobObjects.Append(evemail.NewFeature())
	gobObjects.RegisterAll()
	configPath, err := evapi.ConfigPath("config.xml", "github.com/evalgo/evemail")
	if err != nil {
		panic(err)
	}
	log.Println("get config path:", configPath)
	config, err := evemail.Config(configPath)
	if err != nil {
		panic(err)
	}
	feature := evemail.NewFeature()
	feature.Config = config
	rpc.Register(feature)
	rpc.HandleHTTP()
	var ip string = ""
	ip, err := evapi.EVApiHostIp()
	if err != nil {
		log.Println("warning:", err)
	}
	log.Println("starting feature on  " + ip + ":7070...")
	l, e := net.Listen("tcp", ip+":7070")
	if e != nil {
		panic(e)
	}
	http.Serve(l, nil)
}
