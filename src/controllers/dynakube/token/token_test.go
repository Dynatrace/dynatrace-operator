package token

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestTokens(t *testing.T) {
	t.Run("set api token scopes", testSetApiTokenScopes)
}

func testSetApiTokenScopes(t *testing.T) {
	t.Run("empty dynakube", func(t *testing.T) {
		tokens := Tokens{
			dtclient.DynatraceApiToken: {},
		}
		tokens = tokens.setScopes(dynatracev1beta1.DynaKube{})

		assert.Equal(t,
			[]string{
				dtclient.TokenScopeInstallerDownload,
				dtclient.TokenScopeDataExport,
			},
			tokens.ApiToken().RequiredScopes)
	})
	t.Run("disabled host requests", func(t *testing.T) {
		tokens := Tokens{
			dtclient.DynatraceApiToken: {},
		}
		tokens = tokens.setScopes(dynatracev1beta1.DynaKube{
			ObjectMeta: v1.ObjectMeta{
				Annotations: map[string]string{
					dynatracev1beta1.AnnotationFeatureHostsRequests: "false",
				},
			},
		})

		assert.Equal(t,
			[]string{
				dtclient.TokenScopeInstallerDownload,
			},
			tokens.ApiToken().RequiredScopes)
	})
}
