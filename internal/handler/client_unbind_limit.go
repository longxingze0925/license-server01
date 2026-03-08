package handler

import (
	"errors"
	"license-server/internal/model"

	"gorm.io/gorm"
)

const clientUnbindLimitExceededMessage = "解绑次数已用完，请联系管理员"

var errClientUnbindLimitExceeded = errors.New("client unbind limit exceeded")

func increaseLicenseUnbindUsed(tx *gorm.DB, licenseID, tenantID, appID string) error {
	result := tx.Model(&model.License{}).
		Where("id = ? AND tenant_id = ? AND app_id = ? AND unbind_used < unbind_limit", licenseID, tenantID, appID).
		UpdateColumn("unbind_used", gorm.Expr("unbind_used + 1"))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errClientUnbindLimitExceeded
	}
	return nil
}

func increaseSubscriptionUnbindUsed(tx *gorm.DB, subscriptionID, tenantID, appID string) error {
	result := tx.Model(&model.Subscription{}).
		Where("id = ? AND tenant_id = ? AND app_id = ? AND unbind_used < unbind_limit", subscriptionID, tenantID, appID).
		UpdateColumn("unbind_used", gorm.Expr("unbind_used + 1"))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errClientUnbindLimitExceeded
	}
	return nil
}
