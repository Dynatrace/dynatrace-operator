package kspm

func (kspm *Kspm) SetName(name string) {
	kspm.name = name
}

func (kspm *Kspm) IsEnabled() bool {
	return kspm.Enabled
}

func (kspm *Kspm) GetTokenSecretName() string {
	return kspm.name + "-" + TokenSecretKey
}
