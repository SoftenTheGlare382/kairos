package account

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// Repository 账户仓储
type Repository struct {
	db *gorm.DB
}

// NewRepository 创建仓储
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// Create 创建账户
func (r *Repository) Create(ctx context.Context, account *Account) error {
	return r.db.WithContext(ctx).Create(account).Error
}

// Rename 重命名
func (r *Repository) Rename(ctx context.Context, id uint, newUsername string) error {
	result := r.db.WithContext(ctx).Model(&Account{}).Where("id = ?", id).Update("username", newUsername)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// UpdatePassword 更新密码（bcrypt 密文）
func (r *Repository) UpdatePassword(ctx context.Context, id uint, hashedPassword string) error {
	return r.db.WithContext(ctx).Model(&Account{}).Where("id = ?", id).Update("password", hashedPassword).Error
}

// UpdateLastLoginAt 更新最近登录时间
func (r *Repository) UpdateLastLoginAt(ctx context.Context, id uint, t time.Time) error {
	return r.db.WithContext(ctx).Model(&Account{}).Where("id = ?", id).Update("last_login_at", t).Error
}

// UpdateLastLogoutAt 更新最近登出时间
func (r *Repository) UpdateLastLogoutAt(ctx context.Context, id uint, t time.Time) error {
	return r.db.WithContext(ctx).Model(&Account{}).Where("id = ?", id).Update("last_logout_at", t).Error
}

// Delete 软删除账户
func (r *Repository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&Account{}, id).Error
}

// FindByID 按 ID 查询
func (r *Repository) FindByID(ctx context.Context, id uint) (*Account, error) {
	var account Account
	if err := r.db.WithContext(ctx).First(&account, id).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

// FindByUsername 按用户名查询
func (r *Repository) FindByUsername(ctx context.Context, username string) (*Account, error) {
	var account Account
	if err := r.db.WithContext(ctx).Where("username = ?", username).First(&account).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

// FindByIDs 批量按 ID 查询（供 Social 等下游服务补全用户信息）
func (r *Repository) FindByIDs(ctx context.Context, ids []uint) ([]Account, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var list []Account
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

