// Copyright 2012 EVALGO Community Developers.
// See the LICENSE file at the top-level directory of this distribution
// and at http://opensource.org/licenses/bsd-license.php

package main

import (
	"github.com/evalgo/evapi"
	"github.com/evalgo/evapplication"
	"github.com/evalgo/evemail"
	"github.com/evalgo/everror"
	"github.com/evalgo/evlog"
	"net"
	"net/http"
	"net/rpc"
)

func main() {
	gobObjects := evapplication.NewGobRegisteredObjects()
	gobObjects.Append(evemail.NewFeature())
	gobObjects.RegisterAll()
	configPath, err := evapi.ConfigPath("config.xml", "evalgo/evemail")
	if err != nil {
		evlog.FatalError(everror.NewFromError(err))
	}
	evlog.Println("get config path:", configPath)
	config, err := evemail.Config(configPath)
	if err != nil {
		evlog.FatalError(everror.NewFromError(err))
	}
	feature := evemail.NewFeature()
	feature.Config = config
	rpc.Register(feature)
	rpc.HandleHTTP()
	var ip string = ""
	ip, err = evapi.HostIp()
	if err != nil {
		evlog.Println("warning:", everror.NewFromError(err))
	}
	evlog.Println("starting feature on  " + ip + ":7070...")
	l, e := net.Listen("tcp", ip+":7070")
	if e != nil {
		evlog.FatalError(everror.NewFromError(e))
	}
	http.Serve(l, nil)
}
