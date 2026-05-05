package optionalscope

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/token"
	"k8s.io/utils/ptr"
)

func IsAvailable(dk *dynakube.DynaKube, scope string) bool {
	switch scope {
	case token.ScopeSettingsRead:
		return dk.Status.APIToken.AvailableOptionalScopes.SettingsRead != nil && *dk.Status.APIToken.AvailableOptionalScopes.SettingsRead
	case token.ScopeSettingsWrite:
		return dk.Status.APIToken.AvailableOptionalScopes.SettingsWrite != nil && *dk.Status.APIToken.AvailableOptionalScopes.SettingsWrite
	}

	return false
}

func SetMissing(dk *dynakube.DynaKube, scope string) {
	switch scope {
	case token.ScopeSettingsRead:
		dk.Status.APIToken.AvailableOptionalScopes.SettingsRead = ptr.To(false)
	case token.ScopeSettingsWrite:
		dk.Status.APIToken.AvailableOptionalScopes.SettingsWrite = ptr.To(false)
	}
}

func SetAvailable(dk *dynakube.DynaKube, scope string) {
	switch scope {
	case token.ScopeSettingsRead:
		dk.Status.APIToken.AvailableOptionalScopes.SettingsRead = ptr.To(true)
	case token.ScopeSettingsWrite:
		dk.Status.APIToken.AvailableOptionalScopes.SettingsWrite = ptr.To(true)
	}
}
