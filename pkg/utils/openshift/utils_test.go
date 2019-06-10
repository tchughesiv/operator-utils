package openshift

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/rest"
	"testing"
)

type FakePlatformVersioner struct {
	Info PlatformInfo
	Err  error
}

func (pv FakePlatformVersioner) GetPlatformInfo(d Discoverer, cfg *rest.Config) (PlatformInfo, error) {
	if pv.Err != nil {
		return pv.Info, pv.Err
	}
	return pv.Info, nil
}

func TestDetectOpenShift(t *testing.T) {

	ocpInfo := PlatformInfo{
		Name:       OpenShift,
		OCPVersion: "1.2.3",
		K8SVersion: "4.5.6",
		OS:         "foo/bar",
	}
	k8sInfo := ocpInfo
	k8sInfo.Name = Kubernetes

	cases := []struct {
		pv           FakePlatformVersioner
		cfg          *rest.Config
		expectedBool bool
		expectedErr  error
	}{
		{
			pv: FakePlatformVersioner{
				Info: ocpInfo,
				Err:  nil,
			},
			cfg:          nil,
			expectedBool: true,
			expectedErr:  nil,
		},
		{
			pv: FakePlatformVersioner{
				Info: k8sInfo,
				Err:  nil,
			},
			cfg:          nil,
			expectedBool: false,
			expectedErr:  nil,
		},
		{
			pv: FakePlatformVersioner{
				Info: ocpInfo,
				Err:  errors.New("uh oh"),
			},
			cfg:          nil,
			expectedBool: false,
			expectedErr:  ErrInfoFetch,
		},
	}

	for _, c := range cases {
		IsOpenShift, err := DetectOpenShift(c.pv, c.cfg)
		assert.Equal(t, c.expectedBool, IsOpenShift, "mismatch in returned boolean result")
		assert.Equal(t, c.expectedErr, err, "mismatch in returned error")
	}
}
