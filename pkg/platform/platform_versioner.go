package platform

import (
	"encoding/json"
	"errors"
	openapi_v2 "github.com/googleapis/gnostic/OpenAPIv2"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var (
	log                     = logf.Log.WithName("utils")
	ErrInfoFetch            = errors.New("error fetching PlatformInfo")
	ClusterVersionApiPath   = "apis/config.openshift.io/v1/clusterversions/version"
	ErrDefaultingArgs       = errors.New("error while defaulting non-provided args for PlatformInfo fetch")
	ErrRestConfigFetch      = errors.New("error fetching REST config to replace passed nil")
	ErrDiscoveryClientFetch = errors.New("error getting new DiscoveryClient")
	ErrK8SVersionFetch      = errors.New("error fetching K8S version from ServerVersion")
	ErrServerGroupsFetch    = errors.New("error fetching ServerGroups from discovery client")
	ErrOpenAPISchemaFetch   = errors.New("error fetching OpenAPISchema")
	ErrClusterVersionFetch  = errors.New("error fetching OCP4 version via API")
	ErrVersionMarshalling   = errors.New("error unmarshalling version API result to ClusterVersionInfo")
)

type PlatformVersioner interface {
	GetPlatformInfo(discoverer Discoverer, cfg *rest.Config) (PlatformInfo, error)
}

type Discoverer interface {
	ServerVersion() (*version.Info, error)
	ServerGroups() (*v1.APIGroupList, error)
	OpenAPISchema() (*openapi_v2.Document, error)
	RESTClient() rest.Interface
}

type K8SBasedPlatformVersioner struct{}

// deal with cfg coming from legacy method signature and allow injection for client testing
func (K8SBasedPlatformVersioner) DefaultArgs(client Discoverer, cfg *rest.Config) (Discoverer, *rest.Config, error) {
	if cfg == nil {
		var err error
		cfg, err = config.GetConfig()
		if err != nil {
			log.Error(err, ErrRestConfigFetch.Error())
			return nil, nil, ErrRestConfigFetch
		}
	}
	if client == nil {
		var err error
		client, err = discovery.NewDiscoveryClientForConfig(cfg)
		if err != nil {
			log.Error(err, ErrDiscoveryClientFetch.Error())
			return nil, nil, ErrDiscoveryClientFetch
		}
	}
	return client, cfg, nil
}

func (pv K8SBasedPlatformVersioner) GetPlatformInfo(client Discoverer, cfg *rest.Config) (PlatformInfo, error) {
	log.Info("detecting platform version...")
	info := PlatformInfo{Name: Kubernetes}

	var err error
	client, cfg, err = pv.DefaultArgs(client, cfg)
	if err != nil {
		log.Error(err, ErrDefaultingArgs.Error())
		return info, ErrDefaultingArgs
	}

	k8sVersion, err := client.ServerVersion()
	if err != nil {
		log.Error(err, ErrK8SVersionFetch.Error())
		return info, ErrK8SVersionFetch
	}
	info.K8SVersion = k8sVersion.Major + "." + k8sVersion.Minor
	info.OS = k8sVersion.Platform

	apiList, err := client.ServerGroups()
	if err != nil {
		log.Error(err, ErrServerGroupsFetch.Error())
		return info, ErrServerGroupsFetch
	}

	for _, v := range apiList.Groups {
		if v.Name == "route.openshift.io" {

			info.Name = OpenShift
			log.Info("route.openshift.io found in apis, platform is OpenShift")

			doc, err := client.OpenAPISchema()
			if err != nil {
				log.Error(err, ErrOpenAPISchemaFetch.Error())
				return info, ErrOpenAPISchemaFetch
			}

			switch doc.Info.Version[:4] {
			case "v3.1":
				info.OCPVersion = doc.Info.Version

			// OCP4 returns K8S major/minor from old API endpoint [bugzilla-1658957]
			case "v1.1":
				// sooo much interfacing for testing...
				body, err := client.RESTClient().Get().AbsPath(ClusterVersionApiPath).Do().Raw()

				if err != nil {
					log.Error(err, ErrClusterVersionFetch.Error())
					return info, ErrClusterVersionFetch
				}

				var cvi PlatformClusterInfo
				err = json.Unmarshal(body, &cvi)
				if err != nil {
					log.Error(err, ErrVersionMarshalling.Error())
					return info, ErrVersionMarshalling
				}
				info.OCPVersion = cvi.Status.Desired.Version
			}
		}
	}

	log.Info(info.String())
	return info, nil
}

func DetectOpenShift(pv PlatformVersioner, cfg *rest.Config) (bool, error) {

	if pv == nil {
		pv = K8SBasedPlatformVersioner{}
	}
	info, err := pv.GetPlatformInfo(nil, cfg)
	if err != nil {
		log.Error(err, ErrInfoFetch.Error()+", returning false")
		return false, ErrInfoFetch
	}
	return info.Name == OpenShift, nil
}
