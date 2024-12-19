package validation

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/oneagent"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestImageFieldHasTenantImage(t *testing.T) {
	testTenantUrl := "https://beepboop.dev.dynatracelabs.com"

	t.Run("image fields are a malformed ref", func(t *testing.T) {
		expectedMessage := strings.Join([]string{
			fmt.Sprintf(errorUnparsableImageRef, "ActiveGate"),
			fmt.Sprintf(errorUnparsableImageRef, "OneAgent"),
		}, ";")

		assertDenied(t, []string{expectedMessage}, &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name: "dynakube",
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL: testTenantUrl + "/api",
				OneAgent: oneagent.Spec{
					ClassicFullStack: &oneagent.HostInjectSpec{
						Image: "BOOM",
					},
				},
				ActiveGate: activegate.Spec{
					CapabilityProperties: activegate.CapabilityProperties{
						Image: "BOOM",
					},
				},
			},
		})
	})

	t.Run("valid image fields", func(t *testing.T) {
		testRegistryUrl := "my.images.com"
		assertAllowed(t, &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name: "dynakube",
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL: testTenantUrl + "/api",
				OneAgent: oneagent.Spec{
					ClassicFullStack: &oneagent.HostInjectSpec{
						Image: testRegistryUrl + "/linux/oneagent:latest",
					},
				},
				ActiveGate: activegate.Spec{
					CapabilityProperties: activegate.CapabilityProperties{
						Image: testRegistryUrl + "/linux/activegate:latest",
					},
				},
			},
		})
	})

	t.Run("valid image fields - only OA", func(t *testing.T) {
		testRegistryUrl := "my.images.com"
		assertAllowed(t, &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name: "dynakube",
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL: testTenantUrl + "/api",
				OneAgent: oneagent.Spec{
					ClassicFullStack: &oneagent.HostInjectSpec{
						Image: testRegistryUrl + "/linux/oneagent:latest",
					},
				},
			},
		})
	})

	t.Run("ip:port", func(t *testing.T) {
		assertAllowed(t, &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name: "dynakube",
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL: testTenantUrl + "/api",
				OneAgent: oneagent.Spec{
					ClassicFullStack: &oneagent.HostInjectSpec{
						Image: "127.0.0.1:5000/test:tag",
					},
				},
			},
		})
	})
}

func TestCheckImageField(t *testing.T) {
	type testCase struct {
		title    string
		imageURI string
		isError  bool
	}

	testCases := []testCase{
		{
			title:    "uri without tag",
			imageURI: "my.images.com/test",
			isError:  false,
		},
		{
			title:    "uri with tag",
			imageURI: "my.images.com/test:tag",
			isError:  false,
		},
		{
			title:    "uri with protocol",
			imageURI: "https://my.images.com/test:tag",
			isError:  false,
		},
		{
			title:    "uri with port no protocol",
			imageURI: "my.images.com:5000/test:tag",
			isError:  false,
		},
		{
			title:    "uri with protocol:ip",
			imageURI: "https://127.0.0.1/test:tag",
			isError:  false,
		},
		{
			title:    "uri with ip:port no protocol",
			imageURI: "127.0.0.1:5000/test:tag",
			isError:  false,
		},
		{
			title:    "uri with ipv6:port",
			imageURI: "[1080:0:0:0:8:800:200C:417A]:8888/test",
			isError:  false,
		},
		{
			title:    "uri with ipv6:port:tag",
			imageURI: "[1080:0:0:0:8:800:200C:417A]:8888/test:tag",
			isError:  false,
		},
		{
			title:    "uri with protocol:ipv6:port",
			imageURI: "https://[1080:0:0:0:8:800:200C:417A]:8888/test",
			isError:  true, // the image parsing library will error
		},
		{
			title:    "uri with protocol:ip:port",
			imageURI: "https://127.0.0.1:5000/test:tag",
			isError:  true, // the image parsing library will error
		},
		{
			title:    "uri with protocol port",
			imageURI: "https://my.images.com:5000/test:tag",
			isError:  true, // the image parsing library will error
		},
		{
			title:    "some random URI",
			imageURI: "BOOM",
			isError:  true,
		},
	}

	for _, test := range testCases {
		t.Run(test.title, func(t *testing.T) {
			errMsg := checkImageField(test.imageURI, "test")
			if test.isError {
				require.NotEmpty(t, errMsg)
			} else {
				require.Empty(t, errMsg)
			}
		})
	}
}
