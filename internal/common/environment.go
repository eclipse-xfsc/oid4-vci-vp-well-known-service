package common

import (
	"github.com/eclipse-xfsc/microservice-core-go/pkg/logr"

	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/swaggo/swag/example/basic/docs"

	"github.com/eclipse-xfsc/oid4-vci-vp-well-known-service/config"
)

type Environment struct {
	logger     *logr.Logger
	config     *config.Config
	healthFunc func() bool
}

var env *Environment

func init() {
	env = new(Environment)
}

func GetEnvironment() *Environment {
	return env
}

func (e *Environment) IsHealthy() bool {
	return e.healthFunc()
}

func (e *Environment) SetHealthFunc(healthFunc func() bool) {
	e.healthFunc = healthFunc
}

func (e *Environment) SetConfig(config *config.Config) {
	e.config = config
}

func (e *Environment) GetConfig() *config.Config {
	return e.config
}

func (e *Environment) SetLogger(logger *logr.Logger) {
	e.logger = logger
}

func (e *Environment) GetLogger() *logr.Logger {
	return e.logger
}

// SetSwaggerBasePath sets the base path that will be used by swagger ui for requests url generation
func (e *Environment) SetSwaggerBasePath(path string) {
	docs.SwaggerInfo.BasePath = path + BasePath
}

// SwaggerOptions swagger config options. See github.com/swaggo/gin-swagger?tab=readme-ov-file#configuration
func (e *Environment) SwaggerOptions() []func(config *ginSwagger.Config) {
	return []func(config *ginSwagger.Config){
		ginSwagger.DefaultModelsExpandDepth(10),
	}
}
