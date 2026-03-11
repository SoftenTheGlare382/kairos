package account

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

// Handler HTTP 处理器
type Handler struct {
	svc *Service
}

// NewHandler 创建处理器
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Register 注册
func (h *Handler) Register(c *gin.Context) {
	var req CreateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.svc.Create(c.Request.Context(), &Account{Username: req.Username, Password: req.Password}); err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			c.JSON(http.StatusConflict, gin.H{"error": "username already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "account created"})
}

// Login 登录
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	token, err := h.svc.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid username or password"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token})
}

// Logout 登出
func (h *Handler) Logout(c *gin.Context) {
	accountID := h.getAccountID(c)
	if accountID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "accountID not found"})
		return
	}
	if err := h.svc.Logout(c.Request.Context(), accountID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "account logged out"})
}

// Cancel 注销账户（软删除）
func (h *Handler) Cancel(c *gin.Context) {
	accountID := h.getAccountID(c)
	if accountID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "accountID not found"})
		return
	}
	if err := h.svc.Cancel(c.Request.Context(), accountID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "account cancelled"})
}

// Rename 重命名
func (h *Handler) Rename(c *gin.Context) {
	var req RenameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	accountID := h.getAccountID(c)
	if accountID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "accountID not found"})
		return
	}
	token, err := h.svc.Rename(c.Request.Context(), accountID, req.NewUsername)
	if err != nil {
		if errors.Is(err, ErrNewUsernameRequired) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, ErrUsernameTaken) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token})
}

// ChangePassword 修改密码
func (h *Handler) ChangePassword(c *gin.Context) {
	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.svc.ChangePassword(c.Request.Context(), req.Username, req.OldPassword, req.NewPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsuccessfully password changed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "successfully password changed"})
}

// FindByID 按 ID 查询
func (h *Handler) FindByID(c *gin.Context) {
	var req FindByIDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	account, err := h.svc.FindByID(c.Request.Context(), req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	resp := gin.H{"id": account.ID, "username": account.Username}
	if account.LastLoginAt != nil {
		resp["last_login_at"] = account.LastLoginAt
	}
	if account.LastLogoutAt != nil {
		resp["last_logout_at"] = account.LastLogoutAt
	}
	c.JSON(http.StatusOK, resp)
}

// FindByUsername 按用户名查询
func (h *Handler) FindByUsername(c *gin.Context) {
	var req FindByUsernameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	account, err := h.svc.FindByUsername(c.Request.Context(), req.Username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	resp := gin.H{"id": account.ID, "username": account.Username}
	if account.LastLoginAt != nil {
		resp["last_login_at"] = account.LastLoginAt
	}
	if account.LastLogoutAt != nil {
		resp["last_logout_at"] = account.LastLogoutAt
	}
	c.JSON(http.StatusOK, resp)
}

// Validate 内部 Token 校验（供 Gateway 调用）
func (h *Handler) Validate(c *gin.Context) {
	var req ValidateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Token == "" {
		req.Token = c.GetHeader("Authorization")
		if len(req.Token) > 7 && req.Token[:7] == "Bearer " {
			req.Token = req.Token[7:]
		}
	}
	if req.Token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token required"})
		return
	}
	accountID, username, err := h.svc.ValidateToken(c.Request.Context(), req.Token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"account_id": accountID, "username": username})
}

func (h *Handler) getAccountID(c *gin.Context) uint {
	v, ok := c.Get("accountID")
	if !ok {
		return 0
	}
	id, ok := v.(uint)
	if !ok {
		return 0
	}
	return id
}
