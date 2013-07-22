// Copyright 2012 EVALGO Community Developers.
// See the LICENSE file at the top-level directory of this distribution
// and at http://opensource.org/licenses/bsd-license.php

package main

import (
	"github.com/evalgo/evapplication"
	"github.com/evalgo/evemail"
	"github.com/evalgo/everror"
	"github.com/evalgo/evlog"
	"net/http"
)

var MuxPatterns []string
var MuxFound bool

func Factory(featureName string, context *evapplication.Context) (evapplication.FeatureInterface, error) {
	switch featureName {

	case "evemail":
		evlog.Println("creating feature object: evemail")
		handler, err := evemail.CreateFeature(context)
		if err != nil {
			return nil, everror.NewFromError(err)
		}
		return handler, nil
	}
	return nil, everror.New("evemail.Factory(): the object call for <" + featureName + "> was not found!")
}

func main() {
	MuxPatterns = make([]string, 0)
	MuxFound = false
	context := evapplication.NewContext()
	context.Name = "evemail"
	handlObj, err := Factory("evemail", context)
	if err != nil {
		evlog.Println(everror.NewFromError(err))
	}
	evlog.Println("running Initialize()...")
	err = handlObj.Initialize()
	if err != nil {
		evlog.FatalError(everror.NewFromError(err))
	}
	// register feature urls
	for _, fUrl := range handlObj.URLS() {
		evlog.Println("register url", fUrl)
		http.Handle(fUrl, handlObj)
	}
	// register feature static urls
	for _, fUrl := range handlObj.StaticURLS() {
		for _, pattern := range MuxPatterns {
			if pattern == fUrl {
				MuxFound = true
			}
		}
		if !MuxFound {
			evlog.Println("register static url", fUrl)
			realRoot := handlObj.ThemeRoot() + "/" + fUrl + "/"
			http.Handle("/"+fUrl+"/", http.StripPrefix("/"+fUrl, http.FileServer(http.Dir(realRoot))))
			MuxPatterns = append(MuxPatterns, fUrl)
		} else {
			evlog.Println("could not register url:", fUrl, "because it is already registered!")
		}
		MuxFound = false
	}
	handlers := make(map[string]interface{}, 0)
	handlers["evemail"] = handlObj
	handlObj.SetRegisteredHandlers(handlers)
	evlog.Println("starting feature on 127.0.0.1:9090")
	http.ListenAndServe("127.0.0.1:9090", nil)
}
