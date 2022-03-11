package authsource

import (
	"crypto/tls"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-ldap/ldap/v3"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/service/handlers"
	"kubegems.io/pkg/service/models"
	"kubegems.io/pkg/utils"
)

// ListAuthSourceSimple list authsource simple
// @Tags AuthSource
// @Summary AuthSource列表 (no auth)
// @Description AuthSource列表 (no auth)
// @Accept json
// @Produce json
// @Success 200 {object} handlers.ResponseStruct{Data=[]models.AuthSource} "AuthSource"
// @Router /v1/system/authsource [get]
// @Security JWT
func (h *AuthSourceHandler) ListAuthSourceSimple(c *gin.Context) {
	list := []models.AuthSourceSimple{}
	if err := h.GetDB().Find(&list).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, list)
}

// ListAuthSource list authsource
// @Tags AuthSource
// @Summary AuthSource列表
// @Description AuthSource列表
// @Accept json
// @Produce json
// @Param page query int false "page"
// @Param size query int false "page"
// @Success 200 {object} handlers.ResponseStruct{Data=handlers.PageData{List=[]models.AuthSource}} "AuthSource"
// @Router /v1/authsource [get]
// @Security JWT
func (h *AuthSourceHandler) ListAuthSource(c *gin.Context) {
	var list []models.AuthSource
	query, err := handlers.GetQuery(c, nil)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	cond := &handlers.PageQueryCond{
		Model: "AuthSource",
	}
	total, page, size, err := query.PageList(h.GetDB(), cond, &list)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, handlers.Page(total, list, int64(page), int64(size)))
}

// Create create authsource
// @Tags AuthSource
// @Summary create AuthSource
// @Description create AuthSource  oauth(authURL,tokenURL,userInfoURL,redirectURL,appID,appSecret,scopes) ldap(basedn,ldapaddr,binduser,password)
// @Accept json
// @Produce json
// @Param param body models.AuthSource true "表单"
// @Success 200 {object} handlers.ResponseStruct{Data=models.AuthSource} "AuthSource"
// @Router /v1/authsource [post]
// @Security JWT
func (h *AuthSourceHandler) Create(c *gin.Context) {
	var source models.AuthSource
	ctx := c.Request.Context()
	if err := c.BindJSON(&source); err != nil {
		handlers.NotOK(c, err)
		return
	}
	source.Enabled = true
	if err := validateAuthConfig(&source); err != nil {
		handlers.NotOK(c, err)
		return
	}
	if err := h.GetDB().WithContext(ctx).Save(&source).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.Created(c, source)
}

// Create modify authsource
// @Tags AuthSource
// @Summary modify AuthSource
// @Description modify AuthSource
// @Accept json
// @Produce json
// @Param param body models.AuthSource true "表单"
// @Param source_id path uint true "source_id"
// @Success 200 {object} handlers.ResponseStruct{Data=models.AuthSource} "AuthSource"
// @Router /v1/authsource/{source_id} [put]
// @Security JWT
func (h *AuthSourceHandler) Modify(c *gin.Context) {
	var (
		source models.AuthSource
		newOne models.AuthSource
	)
	ctx := c.Request.Context()
	pk := utils.ToUint(c.Param("source_id"))
	if err := h.GetDB().WithContext(ctx).First(&source, pk).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	if err := c.BindJSON(&newOne); err != nil {
		handlers.NotOK(c, err)
		return
	}
	if err := validateAuthConfig(&newOne); err != nil {
		handlers.NotOK(c, err)
		return
	}
	source.Config = newOne.Config
	now := time.Now()
	source.UpdatedAt = &now
	if err := h.GetDB().Save(source).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, source)

}

// Create delete authsource
// @Tags AuthSource
// @Summary delete AuthSource
// @Description delete AuthSource
// @Accept json
// @Produce json
// @Param source_id path uint true "source_id"
// @Success 200 {object} handlers.ResponseStruct{Data=object} "AuthSource"
// @Router /v1/authsource/{source_id} [delete]
// @Security JWT
func (h *AuthSourceHandler) Delete(c *gin.Context) {
	var source models.AuthSource
	pk := utils.ToUint(c.Param("source_id"))
	h.GetDB().Delete(&source, pk)
	handlers.NoContent(c, nil)
}

func validateAuthConfig(source *models.AuthSource) error {
	errs := []string{}
	if source.Kind == "LDAP" {
		if source.Config.BaseDN == "" {
			errs = append(errs, "basedn can't empty")
		}
		if !basednIsValid(source.Config.BaseDN) {
			errs = append(errs, "basedn is not valid")
		}
		if source.Config.BindUsername == "" {
			errs = append(errs, "binduser can't empty")
		}
		if source.Config.BindPassword == "" {
			errs = append(errs, "password can't empty")
		}
		if source.Config.LdapAddr == "" {
			errs = append(errs, "ldapaddr can't empty")
		}
		if err := filterIsValid(source.Config.Filter); err != nil {
			errs = append(errs, fmt.Sprintf("filter format error: %v", err))
		}
		if err := validateLdapConfig(source.Config); err != nil {
			errs = append(errs, fmt.Sprintf("test ldap conn error : %v", err))
		}
	}
	if source.Kind == "OAUTH" {
		if source.Config.AppID == "" {
			errs = append(errs, "appID can't empty")
		}
		if source.Config.AppSecret == "" {
			errs = append(errs, "appSecret can't empty")
		}
		if source.Config.AuthURL == "" {
			errs = append(errs, "authURL can't empty")
		}
		if source.Config.RedirectURL == "" {
			errs = append(errs, "redirectURL can't empty")
		}
		if source.Config.UserInfoURL == "" {
			errs = append(errs, "userInfoURL can't empty")
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf(strings.Join(errs, ";"))
	}
	return nil
}

func validateLdapConfig(cfg models.AuthSourceConfig) error {
	req := ldap.NewSimpleBindRequest(cfg.BindUsername, cfg.BindPassword, nil)
	var (
		ldapConn *ldap.Conn
		err      error
	)
	ldap.DefaultTimeout = time.Second * 5
	if strings.HasPrefix(cfg.LdapAddr, "ldap") {
		ldapConn, err = ldap.DialURL(
			cfg.LdapAddr,
			ldap.DialWithDialer(&net.Dialer{Timeout: time.Second * 5}),
		)
	} else {
		ldapConn, err = ldap.Dial("tcp", cfg.LdapAddr)
	}
	if err != nil {
		log.Info("validate ldap config failed", "ldapaddr", cfg.LdapAddr, "error", err.Error())
		return err
	}
	ldapConn.SetTimeout(time.Second * 5)
	defer ldapConn.Close()
	if cfg.EnableTLS {
		if err := ldapConn.StartTLS(&tls.Config{InsecureSkipVerify: true}); err != nil {
			return err
		}
	}
	_, err = ldapConn.SimpleBind(req)
	return err
}

func basednIsValid(basedn string) bool {
	seps := strings.Split(basedn, ",")
	for _, sep := range seps {
		kvs := strings.Split(sep, "=")
		if len(kvs) != 2 {
			return false
		}
		if len(kvs[0]) == 0 || len(kvs[1]) == 0 {
			return false
		}
	}
	return true
}

func filterIsValid(filter string) error {
	if filter == "" {
		return nil
	}
	_, err := ldap.CompileFilter(filter)
	if err != nil {
		log.Debugf("ldap filter format error: %s, %v", filter, err)
		return err
	}
	return nil
}
