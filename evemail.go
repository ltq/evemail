// Copyright 2012 EVALGO Community Developers.
// See the LICENSE file at the top-level directory of this distribution
// and at http://opensource.org/licenses/bsd-license.php

package evemail

import (
	"fmt"
	"github.com/evalgo/evapi"
	"github.com/evalgo/evapplication"
	"github.com/evalgo/everror"
	"github.com/evalgo/evlog"
	"github.com/evalgo/evmail"
	"github.com/evalgo/evmessage"
	"github.com/evalgo/evmonitor"
	"github.com/evalgo/evxml"
	"io/ioutil"
	"net/http"
	"net/rpc"
)

var RpcObjects map[string]evmessage.HttpRpcInterface = nil

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
	FeatureName string                      `xml:"FeatureName"`
	SrvName     string                      `xml:"ServiceName"`
	URLS        []string                    `xml:"URLS"`
	Theme       *evapplication.FeatureTheme `xml:"FeatureTheme"`
	Templates   []*FeatureTemplate          `xml:"FeatureTemplates>FeatureTemplate"`
	Redirect    *Redirect                   `xml:"Redirect"`
}

func NewFeatureConfig() *FeatureConfig {
	config := new(FeatureConfig)
	config.URLS = make([]string, 0)
	config.Theme = evapplication.NewFeatureTheme()
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
		return nil, everror.NewFromError(err)
	}
	return xml, nil
}

func (config *FeatureConfig) ConfigString() (string, error) {
	xml, err := config.ConfigBytes()
	if err != nil {
		return "", everror.NewFromError(err)
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
	return "", everror.New("given template <" + typeT + "> was not found!")
}

type Feature struct {
	Config  interface{}
	Context *evapplication.Context
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
		return nil, everror.NewFromError(err)
	}
	err = evxml.FromXml(config, xml)
	if err != nil {
		return nil, everror.NewFromError(err)
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

func CreateFeature(context *evapplication.Context) (*Feature, error) {
	configPath, err := evapi.PackageConfigPath("config.xml", context.Name, "evalgo/evemail")
	if err != nil {
		return nil, everror.NewFromError(err)
	}
	config, err := Config(configPath)
	if err != nil {
		return nil, everror.NewFromError(err)
	}

	feature := NewFeature()
	feature.Context = context
	feature.Config = config
	return feature, nil
}

func (httpFeature *Feature) Initialize() error {
	//initialize rpc objects
	RpcObjects = make(map[string]evmessage.HttpRpcInterface, 0)
	evlog.Println("initialize: adding evmail.NewEmail object...")
	RpcObjects["evmail"] = evmail.NewEmail()
	// register all evapplication message gob objects
	gobObjects := evapplication.NewGobRegisteredObjects()
	gobObjects.RegisterAll()
	return nil
}

func (httpFeature *Feature) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	evlog.Println("starting ServeHTTP ---------------------------------------------- ...")
	evlog.Println(r.Method+":", r.URL.Path)
	w.Header().Add("Content-Type", "text/xml; charset=utf-8")
	evlog.Println("running create messages(req,res)...")
	reqMsg, resMsg, rpcFuncName, err := RpcObjects["evmail"].HttpCreateRpcMessage(w, r)
	if err != nil {
		resMsg := evmessage.RpcServiceInitializeErrorMessage()
		resMsg.Body("errors").(*evmessage.Errors).Append(everror.NewFromError(err))
		evlog.Println("error:", everror.NewFromError(err))
		resXml, _ := resMsg.ToXmlString()
		fmt.Fprintf(w, "%s", resXml)
		return
	}
	connectors := evmessage.NewConnectors()
	connectorsConf := new(evmessage.ConnectorsConf)
	evlog.Println("search connectors file...")
	connPath, err := evapi.PackageConfigPath("connectors.xml", httpFeature.Context.Name, "evalgo/evemail")
	if err != nil {
		resMsg.Body("errors").(*evmessage.Errors).Append(everror.NewFromError(err))
		evlog.Println("error:", everror.NewFromError(err))
		resXml, _ := resMsg.ToXmlString()
		fmt.Fprintf(w, "%s", resXml)
		return
	}
	evlog.Println("read connectors from file...")
	err = evxml.FromXmlFile(connectorsConf, connPath)
	if err != nil {
		resMsg.Body("errors").(*evmessage.Errors).Append(everror.NewFromError(err))
		evlog.Println("error:", everror.NewFromError(err))
		resXml, _ := resMsg.ToXmlString()
		fmt.Fprintf(w, "%s", resXml)
		return
	}
	evlog.Println("load all connectors into the connectors object from connnectors conf...")
	for _, conn := range connectorsConf.Connectors {
		evlog.Println("appending connector:" + conn.Id() + "...")
		connectors.Append(conn)
	}
	evlog.Println("append connectors to request message...")
	reqMsg.AppendToBody(connectors)

	// get connection data for the service from the monitor
	ip, port, err := evmonitor.RpcRequestInfo("evemail-rpc", connectorsConf.Connectors)
	if err != nil {
		resMsg.Body("errors").(*evmessage.Errors).Append(everror.NewFromError(err))
		evlog.Println("error:", everror.NewFromError(err))
		resXml, _ := resMsg.ToXmlString()
		fmt.Fprintf(w, "%s", resXml)
		return
	}

	evlog.Println("create rpc client...")
	client, err := rpc.DialHTTP("tcp", ip+":"+port)
	defer client.Close()
	evlog.Println("call rpc service...")
	err = client.Call(rpcFuncName, reqMsg, resMsg)
	evlog.Println("remove connectors...")
	resMsg.Remove(connectors.EVName())
	if err != nil {
		resMsg.Body("errors").(*evmessage.Errors).Append(everror.NewFromError(err))
		evlog.Println("error:", everror.NewFromError(err))
		resXml, _ := resMsg.ToXmlString()
		fmt.Fprintf(w, "%s", resXml)
		return
	}
	evlog.Println("running response handle...")
	response, err := RpcObjects["evmail"].HttpRpcHandleResponse(w, r, resMsg)
	if err != nil {
		resMsg.Body("errors").(*evmessage.Errors).Append(everror.NewFromError(err))
		evlog.Println("error:", everror.NewFromError(err))
		resXml, _ := resMsg.ToXmlString()
		fmt.Fprintf(w, "%s", resXml)
		return
	}
	fmt.Fprintf(w, "%s", response)
	evlog.Println("ServeHTTP finished successfull...")
}
