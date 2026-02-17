package router

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/psds-microservice/data-channel-service/api"
	"github.com/psds-microservice/data-channel-service/internal/handler"
	"github.com/psds-microservice/data-channel-service/pkg/constants"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func New(dataHandler *handler.DataHandler) http.Handler {
	r := gin.New()
	r.Use(gin.Recovery())

	r.GET(constants.PathHealth, handler.Health)
	r.GET(constants.PathReady, handler.Ready)
	r.GET(constants.PathSwagger, func(c *gin.Context) { c.Redirect(http.StatusFound, constants.PathSwagger+"/") })
	r.GET(constants.PathSwagger+"/*any", func(c *gin.Context) {
		if strings.TrimPrefix(c.Param("any"), "/") == "openapi.json" {
			c.Data(http.StatusOK, "application/json", api.OpenAPISpec)
			return
		}
		if strings.TrimPrefix(c.Param("any"), "/") == "" {
			c.Request.URL.Path = constants.PathSwagger + "/index.html"
			c.Request.RequestURI = constants.PathSwagger + "/index.html"
		}
		ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL("/swagger/openapi.json"))(c)
	})

	r.GET("/ws/data/:session_id/:user_id", dataHandler.ServeWS)
	r.GET("/data/:session_id/history", dataHandler.GetHistory)
	r.POST("/data/file", dataHandler.UploadFile)

	return r
}
