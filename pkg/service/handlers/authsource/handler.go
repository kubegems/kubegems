package authsource

import (
	"time"

	"github.com/gin-gonic/gin"
	"kubegems.io/pkg/service/handlers"
	"kubegems.io/pkg/service/models"
	"kubegems.io/pkg/utils"
)

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
// @Description create AuthSource
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
