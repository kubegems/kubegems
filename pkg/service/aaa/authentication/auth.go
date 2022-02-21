package auth

import (
	"context"
	"crypto/rsa"
	"errors"
	"fmt"
	"io/ioutil"
	"time"

	jwt "github.com/appleboy/gin-jwt/v2"
	jwtgo "github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"kubegems.io/pkg/apis/gems"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/service/aaa"
	"kubegems.io/pkg/service/handlers"
	"kubegems.io/pkg/service/models"
	"kubegems.io/pkg/utils"
	"kubegems.io/pkg/utils/database"
	"kubegems.io/pkg/utils/redis"
	"kubegems.io/pkg/utils/system"
)

var userCacheExirpreMinute = 10

const (
	identityKey = "username"
)

type loginForm struct {
	Username string `form:"username" json:"username" binding:"required"`
	Password string `form:"password" json:"password" binding:"required"`
}

func dbauth(db *gorm.DB, form loginForm) (interface{}, error) {
	var user models.User
	if err := db.First(&user, "username = ?", form.Username).Error; err != nil {
		log.Warnf("dbauth failed for user %s: %v", form.Username, err)
		return nil, errors.New("用户名或者密码错误")
	}
	if err := utils.ValidatePassword(form.Password, user.Password); err != nil {
		return nil, errors.New("用户名或者密码错误")
	}

	if user.IsActive == nil || !*user.IsActive {
		return nil, errors.New("用户暂未激活")
	}

	now := time.Now()
	user.LastLoginAt = &now
	db.Save(&user)
	return &user, nil
}

func getAuthenticator(db *gorm.DB, uif aaa.UserInterface) func(c *gin.Context) (interface{}, error) {
	return func(c *gin.Context) (interface{}, error) {
		var loginVals loginForm
		/*
			var success bool
			begin := time.Now()
		*/
		// TODO: 记录用户登录的操作

		if err := c.ShouldBind(&loginVals); err != nil {
			return "", errors.New("用户名或者密码错误")
		}
		user, err := dbauth(db, loginVals)
		if err != nil {
			return nil, err
		}
		u := user.(*models.User)
		uif.SetContextUser(c, u)
		return u, nil
	}
}

func getIdentityHandler(db *gorm.DB, redis *redis.Client, uif aaa.UserInterface) func(*gin.Context) interface{} {
	return func(c *gin.Context) interface{} {
		claims := jwt.ExtractClaims(c)
		username := claims[identityKey].(string)
		u := models.User{}
		cacheKey := getUserInfoCacheKey(username)
		if err := redis.Get(context.Background(), cacheKey).Scan(&u); err != nil {
			log.Debugf("get userinfo cache failed for user %v, will get from database: %v", cacheKey, err)
			if err := db.Preload("SystemRole").First(&u, "username = ?", username).Error; err != nil {
				log.Warnf("failed to find user %s %v", username, err)
				return ""
			} else {
				uif.SetContextUser(c, &u)
				refreshUserInfoCache(redis, u, userCacheExirpreMinute)
			}
		} else {
			uif.SetContextUser(c, &u)
		}
		return username
	}
}

func getUserInfoCacheKey(username string) string {
	return fmt.Sprintf("userinfo_cahce_%s", username)
}

func refreshUserInfoCache(redis *redis.Client, u models.User, timeout int) error {
	_, err := redis.SetEX(context.TODO(), getUserInfoCacheKey(u.Username), u, time.Duration(timeout)*time.Minute).Result()
	if err != nil {
		log.Warnf("failed to fresh userinfo cache for user %v: %v", u.Username, err)
	}
	return err
}

func getPayloadFunc() func(data interface{}) jwt.MapClaims {
	return func(data interface{}) jwt.MapClaims {
		if v, ok := data.(*models.User); ok {
			return jwt.MapClaims{
				identityKey: v.Username,
				"iat":       time.Now().Unix(),
			}
		}
		return jwt.MapClaims{}
	}
}

func unauthorized(c *gin.Context, code int, message string) {
	handlers.Response(c, code, message, nil)
}

func loginResponse(c *gin.Context, _ int, token string, expire time.Time) {
	handlers.OK(c, gin.H{
		"token":  token,
		"expire": expire.Format(time.RFC3339),
	})
}

func initPrivateKey(keyfile string) (*rsa.PrivateKey, error) {
	keyData, err := ioutil.ReadFile(keyfile)
	if err != nil {
		return nil, err
	}
	key, err := jwtgo.ParseRSAPrivateKeyFromPEM(keyData)
	if err != nil {
		return nil, err
	}
	return key, nil
}

type Middleware struct {
	Database   *database.Database
	PrivateKey *rsa.PrivateKey
	jwt.GinJWTMiddleware
}

func NewAuthMiddleware(system *system.SystemOptions, database *database.Database,
	redis *redis.Client, uif aaa.UserInterface) (*Middleware, error) {
	db := database.DB()
	authMiddleware, err := jwt.New(&jwt.GinJWTMiddleware{
		Realm:            gems.GroupName,
		Key:              []byte{},
		SigningAlgorithm: "RS256",
		Timeout:          system.JwtExpire,                   // 有效时间
		MaxRefresh:       system.JwtExpire,                   // 最大刷新时间
		IdentityKey:      identityKey,                        // ID 字段，这里用的是Username
		PayloadFunc:      getPayloadFunc(),                   // 获取payload
		IdentityHandler:  getIdentityHandler(db, redis, uif), // ID处理
		Authenticator:    getAuthenticator(db, uif),          // 登录认证
		Authorizator:     nil,                                // 权限认证
		Unauthorized:     unauthorized,                       // 未认证
		LoginResponse:    loginResponse,
		TokenLookup:      "header: Authorization, query: token, cookie: jwt",
		TokenHeadName:    "Bearer",
		TimeFunc:         time.Now,
		PrivKeyFile:      system.JWTKey,
		PubKeyFile:       system.JWTCert,
	})
	if err != nil {
		return nil, err
	}
	key, err := initPrivateKey(system.JWTKey)
	if err != nil {
		return nil, err
	}
	midware := &Middleware{
		PrivateKey:       key,
		Database:         database,
		GinJWTMiddleware: *authMiddleware,
	}
	return midware, nil
}
