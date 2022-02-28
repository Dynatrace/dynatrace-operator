package dtclient

type TenantInfo struct {
	UUID  string `json:"tenantUUID"`
	Token string `json:"tenantToken"`
}
