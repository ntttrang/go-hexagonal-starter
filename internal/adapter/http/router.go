package httpadapter

import (
	"log/slog"
	"net/http"
	"net/http/pprof"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

	"github.com/nttttranggo-hexagonal-starter/internal/adapter/http/middleware"
	"github.com/nttttranggo-hexagonal-starter/internal/domain"
	"github.com/nttttranggo-hexagonal-starter/internal/platform/metrics"
)

// Dependencies groups handlers and shared deps for the router.
type Dependencies struct {
	Log         *slog.Logger
	Metrics     *metrics.Metrics
	Tokens      domain.TokenIssuer
	Auth        *AuthHandler
	Users       *UserHandler
	Health      *HealthHandler
	Env         string
	ServiceName string
}

// NewRouter builds the Gin engine with all routes and middleware.
func NewRouter(deps Dependencies) *gin.Engine {
	if deps.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	serviceName := deps.ServiceName
	if serviceName == "" {
		serviceName = "go-hexagonal-starter"
	}

	r := gin.New()
	r.Use(middleware.RequestID())
	r.Use(otelgin.Middleware(serviceName,
		otelgin.WithFilter(func(r *http.Request) bool {
			path := r.URL.Path
			return path != "/metrics" && path != "/healthz" && path != "/readyz" &&
				!strings.HasPrefix(path, "/debug/pprof")
		}),
	))
	r.Use(middleware.Recovery(deps.Log))
	r.Use(middleware.RequestLogger(deps.Log))
	r.Use(middleware.Metrics(deps.Metrics))

	r.GET("/healthz", deps.Health.Liveness)
	r.GET("/readyz", deps.Health.Readiness)
	r.GET("/metrics", gin.WrapH(promhttp.HandlerFor(deps.Metrics.Registry, promhttp.HandlerOpts{})))
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	if deps.Env != "production" {
		registerPprof(r)
		if deps.Log != nil {
			deps.Log.Info("pprof enabled", "path", "/debug/pprof/")
		}
	}

	v1 := r.Group("/api/v1")
	{
		auth := v1.Group("/auth")
		{
			auth.POST("/register", deps.Auth.Register)
			auth.POST("/login", deps.Auth.Login)
		}

		users := v1.Group("/users")
		users.Use(middleware.Auth(deps.Tokens))
		{
			users.GET("", deps.Users.List)
			users.GET("/:id", deps.Users.Get)
			users.PUT("/:id", deps.Users.Update)
			users.DELETE("/:id", deps.Users.Delete)
		}
	}

	return r
}

func registerPprof(r *gin.Engine) {
	dbg := r.Group("/debug/pprof")
	{
		dbg.GET("/", gin.WrapF(pprof.Index))
		dbg.GET("/cmdline", gin.WrapF(pprof.Cmdline))
		dbg.GET("/profile", gin.WrapF(pprof.Profile))
		dbg.POST("/symbol", gin.WrapF(pprof.Symbol))
		dbg.GET("/symbol", gin.WrapF(pprof.Symbol))
		dbg.GET("/trace", gin.WrapF(pprof.Trace))
		dbg.GET("/allocs", gin.WrapH(pprof.Handler("allocs")))
		dbg.GET("/block", gin.WrapH(pprof.Handler("block")))
		dbg.GET("/goroutine", gin.WrapH(pprof.Handler("goroutine")))
		dbg.GET("/heap", gin.WrapH(pprof.Handler("heap")))
		dbg.GET("/mutex", gin.WrapH(pprof.Handler("mutex")))
		dbg.GET("/threadcreate", gin.WrapH(pprof.Handler("threadcreate")))
	}
}
