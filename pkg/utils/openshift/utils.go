package openshift

import (
	"errors"
	"k8s.io/client-go/rest"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var (
	log          = logf.Log.WithName("utils")
	ErrInfoFetch = errors.New("error fetching PlatformInfo")
)

// maintained for legacy method signature
func IsOpenShift(cfg *rest.Config) (bool, error) {
	return DetectOpenShift(nil, cfg)
}

// new logic endpoint allowing PlatformVersioner struct testing/injectability
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
