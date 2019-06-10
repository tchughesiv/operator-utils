package openshift

import (
	openapi_v2 "github.com/googleapis/gnostic/OpenAPIv2"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/rest"
	"testing"
)

type FakeDiscoverer struct {
	info               PlatformInfo
	serverInfo         *version.Info
	groupList          *v1.APIGroupList
	doc                *openapi_v2.Document
	client             rest.Interface
	ServerVersionError error
	ServerGroupsError  error
	OpenAPISchemaError error
}

func (d FakeDiscoverer) ServerVersion() (*version.Info, error) {
	if d.ServerVersionError != nil {
		return nil, d.ServerVersionError
	}
	return d.serverInfo, nil
}

func (d FakeDiscoverer) ServerGroups() (*v1.APIGroupList, error) {
	if d.ServerGroupsError != nil {
		return nil, d.ServerGroupsError
	}
	return d.groupList, nil
}

func (d FakeDiscoverer) OpenAPISchema() (*openapi_v2.Document, error) {
	if d.OpenAPISchemaError != nil {
		return nil, d.OpenAPISchemaError
	}
	return d.doc, nil
}

func (d FakeDiscoverer) RESTClient() rest.Interface {
	return d.client
}

func TestK8SBasedPlatformVersioner_GetPlatformInfo(t *testing.T) {

	pv := K8SBasedPlatformVersioner{}

	cases := []struct {
		discoverer   Discoverer
		config       *rest.Config
		expectedInfo PlatformInfo
		expectedErr  error
	}{
		// CASE 1
		// trigger error in client.ServerVersion(), only Name present on Info
		{
			discoverer: FakeDiscoverer{
				ServerVersionError: errors.New("oops"),
			},
			config:       nil,
			expectedInfo: PlatformInfo{Name: Kubernetes},
			expectedErr:  ErrK8SVersionFetch,
		},
		// CASE 2
		// trigger error in client.ServerGroups(), K8S major/minor now present from ServerVersion() result
		{
			discoverer: FakeDiscoverer{
				ServerGroupsError: errors.New("oops"),
				serverInfo: &version.Info{
					Major: "1",
					Minor: "2",
				},
			},
			config:       nil,
			expectedInfo: PlatformInfo{Name: Kubernetes, K8SVersion: "1.2"},
			expectedErr:  ErrServerGroupsFetch,
		},
		// CASE 3
		// trigger no errors, simulate K8S platform (no OCP route present)
		{
			discoverer: FakeDiscoverer{
				serverInfo: &version.Info{
					Major: "1",
					Minor: "2",
				},
				groupList: &v1.APIGroupList{
					TypeMeta: v1.TypeMeta{},
					Groups:   []v1.APIGroup{},
				},
			},
			config:       nil,
			expectedInfo: PlatformInfo{Name: Kubernetes, K8SVersion: "1.2"},
			expectedErr:  nil,
		},
		// CASE 4
		// trigger error in OpenAPISchema, info should now be OCP with K8S major/minor
		{
			discoverer: FakeDiscoverer{
				OpenAPISchemaError: errors.New("oops"),
				serverInfo: &version.Info{
					Major: "1",
					Minor: "2",
				},
				groupList: &v1.APIGroupList{
					TypeMeta: v1.TypeMeta{},
					Groups: []v1.APIGroup{
						{
							Name: "route.openshift.io",
						},
					},
				},
			},
			config:       nil,
			expectedInfo: PlatformInfo{Name: OpenShift, K8SVersion: "1.2"},
			expectedErr:  ErrOpenAPISchemaFetch,
		},
		// CASE 5
		// trigger no error, let OCP version start with "3.1", info should now reflect this
		{
			discoverer: FakeDiscoverer{
				serverInfo: &version.Info{
					Major: "1",
					Minor: "2",
				},
				groupList: &v1.APIGroupList{
					TypeMeta: v1.TypeMeta{},
					Groups: []v1.APIGroup{
						{
							Name: "route.openshift.io",
						},
					},
				},
				doc: &openapi_v2.Document{
					Info: &openapi_v2.Info{
						Version: "v3.11.42",
					},
				},
			},
			config:       nil,
			expectedInfo: PlatformInfo{Name: OpenShift, K8SVersion: "1.2", OCPVersion: "v3.11.42"},
			expectedErr:  nil,
		},
	}

	for _, c := range cases {
		info, err := pv.GetPlatformInfo(c.discoverer, c.config)
		assert.Equal(t, c.expectedInfo, info, "mismatch in returned PlatformInfo")
		assert.Equal(t, c.expectedErr, err, "mismatch in returned error")
	}
}
