package middleware

import (
	"fmt"
	model "github.com/HFO4/cloudreve/models"
	"github.com/HFO4/cloudreve/pkg/serializer"
	"github.com/HFO4/cloudreve/pkg/util"
	"github.com/gin-gonic/gin"
)

// ShareAvailable 检查分享是否可用
func ShareAvailable() gin.HandlerFunc {
	return func(c *gin.Context) {
		var user *model.User
		if userCtx, ok := c.Get("user"); ok {
			user = userCtx.(*model.User)
		} else {
			user = model.NewAnonymousUser()
		}

		share := model.GetShareByHashID(c.Param("id"))

		if share == nil || !share.IsAvailable() {
			c.JSON(200, serializer.Err(serializer.CodeNotFound, "分享不存在或已被取消", nil))
			c.Abort()
			return
		}

		c.Set("user", user)
		c.Set("share", share)
		c.Next()
	}
}

// ShareCanPreview 检查分享是否可被预览
func ShareCanPreview() gin.HandlerFunc {
	return func(c *gin.Context) {
		if share, ok := c.Get("share"); ok {
			if share.(*model.Share).PreviewEnabled {
				c.Next()
				return
			}
			c.JSON(200, serializer.Err(serializer.CodeNoPermissionErr, "此分享无法预览",
				nil))
			c.Abort()
			return
		}
		c.Abort()
	}
}

// BeforeShareDownload 分享被下载前的检查
func BeforeShareDownload() gin.HandlerFunc {
	return func(c *gin.Context) {
		if shareCtx, ok := c.Get("share"); ok {
			if userCtx, ok := c.Get("user"); ok {
				share := shareCtx.(*model.Share)
				user := userCtx.(*model.User)

				// 检查用户是否可以下载此分享的文件
				err := share.CanBeDownloadBy(user)
				if err != nil {
					c.JSON(200, serializer.Err(serializer.CodeNoPermissionErr, err.Error(),
						nil))
					c.Abort()
					return
				}

				// 分享是否已解锁
				if share.Password != "" {
					sessionKey := fmt.Sprintf("share_unlock_%d", share.ID)
					unlocked := util.GetSession(c, sessionKey) != nil
					if !unlocked {
						c.JSON(200, serializer.Err(serializer.CodeNoPermissionErr,
							"无权访问此分享", nil))
						c.Abort()
						return
					}
				}

				// 对积分、下载次数进行更新
				err = share.DownloadBy(user, c)
				if err != nil {
					c.JSON(200, serializer.Err(serializer.CodeNoPermissionErr, err.Error(),
						nil))
					c.Abort()
					return
				}

				c.Next()
				return
			}
		}
		c.Abort()
	}
}