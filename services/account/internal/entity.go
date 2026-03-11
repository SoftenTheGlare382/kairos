package account

import (
	"time"

	"gorm.io/gorm"
)

// Account 账户实体（Token 仅存于 Redis，不存 MySQL）
type Account struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Username  string         `gorm:"size:191;uniqueIndex:idx_account_username_deleted" json:"username"` // size 避免 MySQL LONGTEXT 无法建唯一索引
	Password    string         `gorm:"size:255" json:"-"`                                                 // bcrypt 密文约 60 字符
	LastLoginAt *time.Time     `json:"last_login_at,omitempty"`                                            // 最近登录时间
	LastLogoutAt *time.Time    `json:"last_logout_at,omitempty"`                                           // 最近登出时间
	DeletedAt   gorm.DeletedAt `gorm:"index,uniqueIndex:idx_account_username_deleted" json:"-"`            // 软删除
	CreatedAt   time.Time      `json:"-"`
	UpdatedAt   time.Time      `json:"-"`
}

// CreateAccountRequest 注册请求
type CreateAccountRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// RenameRequest 重命名请求
type RenameRequest struct {
	NewUsername string `json:"new_username"`
}

// FindByIDRequest 按 ID 查询请求
type FindByIDRequest struct {
	ID uint `json:"id"`
}

// FindByUsernameRequest 按用户名查询请求
type FindByUsernameRequest struct {
	Username string `json:"username"`
}

// ChangePasswordRequest 修改密码请求
type ChangePasswordRequest struct {
	Username    string `json:"username"`
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// ValidateRequest 内部校验 Token 请求（Gateway 调用）
type ValidateRequest struct {
	Token string `json:"token"`
}

// ValidateResponse 内部校验 Token 响应
type ValidateResponse struct {
	AccountID uint   `json:"account_id"`
	Username  string `json:"username"`
}
