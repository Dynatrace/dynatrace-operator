package metadata

import (
	"github.com/pkg/errors"
)

type Cleaner interface {
	ListDeletedTenantConfigs() ([]TenantConfig, error)
	PurgeTenantConfig(tenantConfig *TenantConfig) error

	ListDeletedCodeModules() ([]CodeModule, error)
	PurgeCodeModule(codeModule *CodeModule) error

	ListDeletedAppMounts() ([]AppMount, error)
	PurgeAppMount(appMount *AppMount) error

	ListDeletedOSMounts() ([]OSMount, error)
	PurgeOSMount(osMount *OSMount) error
}

var _ Cleaner = &GormConn{}

func (conn *GormConn) ListDeletedTenantConfigs() ([]TenantConfig, error) {
	var tenantConfigs []TenantConfig

	result := conn.db.WithContext(conn.ctx).Unscoped().Where("deleted_at is not ?", nil).Find(&tenantConfigs)
	if result.Error != nil {
		return nil, result.Error
	}

	return tenantConfigs, nil
}

func (conn *GormConn) PurgeTenantConfig(tenantConfig *TenantConfig) error {
	if (tenantConfig == nil || *tenantConfig == TenantConfig{}) {
		return errors.New("Can't delete an empty TenantConfig")
	}

	return conn.db.WithContext(conn.ctx).Unscoped().Delete(&TenantConfig{}, tenantConfig).Error
}

func (conn *GormConn) ListDeletedCodeModules() ([]CodeModule, error) {
	var codeModules []CodeModule

	result := conn.db.WithContext(conn.ctx).Unscoped().Where("deleted_at is not ?", nil).Find(&codeModules)
	if result.Error != nil {
		return nil, result.Error
	}

	return codeModules, nil
}

func (conn *GormConn) PurgeCodeModule(codeModule *CodeModule) error {
	if (codeModule == nil || *codeModule == CodeModule{}) {
		return errors.New("Can't delete an empty CodeModule")
	}

	return conn.db.WithContext(conn.ctx).Unscoped().Delete(&CodeModule{}, codeModule).Error
}

func (conn *GormConn) ListDeletedAppMounts() ([]AppMount, error) {
	var appMounts []AppMount

	result := conn.db.WithContext(conn.ctx).Unscoped().Where("deleted_at is not ?", nil).Preload("VolumeMeta").Preload("CodeModule").Find(&appMounts)
	if result.Error != nil {
		return nil, result.Error
	}

	return appMounts, nil
}

func (conn *GormConn) PurgeAppMount(appMount *AppMount) error {
	if (appMount == nil || *appMount == AppMount{}) {
		return errors.New("Can't delete an empty AppMount")
	}

	err := conn.db.WithContext(conn.ctx).Unscoped().Delete(&AppMount{}, appMount).Error
	if err != nil {
		return errors.New("couldn't purge app mount, err: " + err.Error())
	}

	return conn.db.WithContext(conn.ctx).Unscoped().Delete(&VolumeMeta{}, appMount.VolumeMeta).Error
}

func (conn *GormConn) ListDeletedOSMounts() ([]OSMount, error) {
	var osMounts []OSMount

	result := conn.db.WithContext(conn.ctx).Unscoped().Where("deleted_at is not ?", nil).Preload("VolumeMeta").Preload("TenantConfig").Find(&osMounts)
	if result.Error != nil {
		return nil, result.Error
	}

	return osMounts, nil
}

func (conn *GormConn) PurgeOSMount(osMount *OSMount) error {
	if (osMount == nil || *osMount == OSMount{}) {
		return errors.New("Can't delete an empty OSMount")
	}

	err := conn.db.WithContext(conn.ctx).Unscoped().Delete(&OSMount{}, osMount).Error
	if err != nil {
		return errors.New("couldn't purge an os mount, err: " + err.Error())
	}

	return conn.db.WithContext(conn.ctx).Unscoped().Delete(&VolumeMeta{}, osMount.VolumeMeta).Error
}
