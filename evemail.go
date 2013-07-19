// Copyright 2012 EVALGO Community Developers.
// See the LICENSE file at the top-level directory of this distribution
// and at http://opensource.org/licenses/bsd-license.php

package evemail

import (
	"errors"
	"fmt"
	"github.com/evalgo/evapi"
	"github.com/evalgo/evapplication"
	"github.com/evalgo/evmail"
	"github.com/evalgo/evmessage"
	"github.com/evalgo/evmonitor"
	"github.com/evalgo/evxml"
	"io/ioutil"
	"log"
	"net/http"
	"net/rpc"
)

var RpcObjects map[string]evmessage.EVMessageHttpRpcInterface = nil

const (
	formDefaultMaxMemory = 32 << 20 // 32 MB
)

type FeatureTemplate struct {
	Name string `xml:"Name"`
	Path string `xml:"Path"`
}

func NewFeatureTemplate() *FeatureTemplate {
	tmpl := new(FeatureTemplate)
	tmpl.Name = ""
	tmpl.Path = ""
	return tmpl
}

type FeatureConfig struct {
	FeatureName string                                   `xml:"FeatureName"`
	SrvName     string                                   `xml:"ServiceName"`
	URLS        []string                                 `xml:"URLS"`
	Theme       *evapplication.EVApplicationFeatureTheme `xml:"FeatureTheme"`
	Templates   []*FeatureTemplate                       `xml:"FeatureTemplates>FeatureTemplate"`
	Redirect    *Redirect                                `xml:"Redirect"`
}

func NewFeatureConfig() *FeatureConfig {
	config := new(FeatureConfig)
	config.URLS = make([]string, 0)
	config.Theme = evapplication.NewEVApplicationFeatureTheme()
	config.Templates = make([]*FeatureTemplate, 0)
	config.Redirect = NewRedirect()
	return config
}

func (config *FeatureConfig) Name() string {
	return config.FeatureName
}

func (config *FeatureConfig) ServiceName() string {
	return config.SrvName
}

func (config *FeatureConfig) ConfigBytes() ([]byte, error) {
	xml, err := evxml.ToXml(config)
	if err != nil {
		return nil, err
	}
	return xml, nil
}

func (config *FeatureConfig) ConfigString() (string, error) {
	xml, err := config.ConfigBytes()
	if err != nil {
		return "", err
	}
	return string(xml[0:]), nil
}

func (config *FeatureConfig) Urls() []string {
	return config.URLS
}

func (config *FeatureConfig) Template(typeT string) (string, error) {
	for _, tmpl := range config.Templates {
		if tmpl.Name == typeT {
			return tmpl.Path, nil
		}
	}
	return "", errors.New("given template <" + typeT + "> was not found!")
}

type Feature struct {
	Config  interface{}
	Context *evapplication.EVApplicationContext
}

func NewFeature() *Feature {
	feature := new(Feature)
	feature.Config = nil
	feature.Context = nil
	return feature
}

type Redirect struct {
	Enabled    bool   `xml:"Enabled"`
	Url        string `xml:"Url"`
	StatusCode int    `xml:"StatusCode"`
}

func NewRedirect() *Redirect {
	redirect := new(Redirect)
	redirect.Enabled = false
	redirect.Url = ""
	redirect.StatusCode = http.StatusTemporaryRedirect
	return redirect
}

func Config(configPath string) (*FeatureConfig, error) {
	config := NewFeatureConfig()
	xml, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	err = evxml.FromXml(config, xml)
	if err != nil {
		return nil, err
	}
	return config, nil
}

func (httpFeature *Feature) URLS() []string {
	return httpFeature.Config.(*FeatureConfig).Urls()
}

func (httpFeature *Feature) StaticURLS() []string {
	return httpFeature.Config.(*FeatureConfig).Theme.StaticDirectory
}

func (httpFeature *Feature) ThemeRoot() string {
	return httpFeature.Config.(*FeatureConfig).Theme.Path + "/" + httpFeature.Config.(*FeatureConfig).Theme.Name
}

func (httpFeature *Feature) SetRegisteredHandlers(handlers map[string]interface{}) {
	httpFeature.Context.Handlers = handlers
}

func CreateFeature(context *evapplication.EVApplicationContext) (*Feature, error) {
	configPath, err := evapi.PackageConfigPath("config.xml", context.Name, "evalgo/evemail")
	if err != nil {
		return nil, err
	}
	config, err := Config(configPath)
	if err != nil {
		return nil, err
	}

	feature := NewFeature()
	feature.Context = context
	feature.Config = config
	return feature, nil
}

func (httpFeature *Feature) Initialize() error {
	//initialize rpc objects
	RpcObjects = make(map[string]evmessage.EVMessageHttpRpcInterface, 0)
	log.Println("initialize: adding evmail.NewEVMailEmail object...")
	RpcObjects["evmail"] = evmail.NewEVMailEmail()
	// register all evapplication message gob objects
	gobObjects := evapplication.NewEVApplicationGobRegisteredObjects()
	gobObjects.RegisterAll()
	return nil
}

func (httpFeature *Feature) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println("starting ServeHTTP ---------------------------------------------- ...")
	log.Println(r.Method+":", r.URL.Path)
	w.Header().Add("Content-Type", "text/xml; charset=utf-8")
	log.Println("running create messages(req,res)...")
	reqMsg, resMsg, rpcFuncName, err := RpcObjects["evmail"].EVMessageHttpCreateRpcMessage(w, r)
	if err != nil {
		resMsg := evmessage.EVMessageRpcServiceInitializeErrorMessage()
		resMsg.Body("errors").(*evmessage.EVMessageErrors).Append(err)
		log.Println("error:", err)
		resXml, _ := resMsg.ToXmlString()
		fmt.Fprintf(w, "%s", resXml)
		return
	}
	connectors := evmessage.NewEVMessageConnectors()
	connectorsConf := new(evmessage.EVMessageConnectorsConf)
	log.Println("search connectors file...")
	connPath, err := evapi.PackageConfigPath("connectors.xml", httpFeature.Context.Name, "evalgo/evemail")
	if err != nil {
		resMsg.Body("errors").(*evmessage.EVMessageErrors).Append(err)
		log.Println("error:", err)
		resXml, _ := resMsg.ToXmlString()
		fmt.Fprintf(w, "%s", resXml)
		return
	}
	log.Println("read connectors from file...")
	err = evxml.FromXmlFile(connectorsConf, connPath)
	if err != nil {
		resMsg.Body("errors").(*evmessage.EVMessageErrors).Append(err)
		log.Println("error:", err)
		resXml, _ := resMsg.ToXmlString()
		fmt.Fprintf(w, "%s", resXml)
		return
	}
	log.Println("load all connectors into the connectors object from connnectors conf...")
	for _, conn := range connectorsConf.Connectors {
		log.Println("appending connector:" + conn.Id() + "...")
		connectors.Append(conn)
	}
	log.Println("append connectors to request message...")
	reqMsg.AppendToBody(connectors)

	// get connection data for the service from the monitor
	ip, port, err := evmonitor.EVMonitorRpcRequestInfo("evemail-rpc", connectorsConf.Connectors)
	if err != nil {
		resMsg.Body("errors").(*evmessage.EVMessageErrors).Append(err)
		log.Println("error:", err)
		resXml, _ := resMsg.ToXmlString()
		fmt.Fprintf(w, "%s", resXml)
		return
	}

	log.Println("create rpc client...")
	client, err := rpc.DialHTTP("tcp", ip+":"+port)
	defer client.Close()
	log.Println("call rpc service...")
	err = client.Call(rpcFuncName, reqMsg, resMsg)
	log.Println("remove connectors...")
	resMsg.Remove(connectors.EVName())
	if err != nil {
		resMsg.Body("errors").(*evmessage.EVMessageErrors).Append(err)
		log.Println("error:", err)
		resXml, _ := resMsg.ToXmlString()
		fmt.Fprintf(w, "%s", resXml)
		return
	}
	log.Println("running response handle...")
	response, err := RpcObjects["evmail"].EVMessageHttpRpcHandleResponse(w, r, resMsg)
	if err != nil {
		resMsg.Body("errors").(*evmessage.EVMessageErrors).Append(err)
		log.Println("error:", err)
		resXml, _ := resMsg.ToXmlString()
		fmt.Fprintf(w, "%s", resXml)
		return
	}
	fmt.Fprintf(w, "%s", response)
	log.Println("ServeHTTP finished successfull...")
}
