package main

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/sirupsen/logrus"
	"github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/config"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"net/http"
	"strconv"
)

var db *gorm.DB

type Flight struct {
	FlightID    int
	Name        string
	Destination string
}

func main() {
	cfg := &config.Configuration{
		ServiceName: "flight-service",
		Sampler: &config.SamplerConfig{
			Type:  "const",
			Param: 1,
		},
		Reporter: &config.ReporterConfig{
			LogSpans:           true,
			LocalAgentHostPort: "localhost:16686", // TODO: Change localhost to jaeger
		},
	}

	tracer, closer, err := cfg.NewTracer(config.Logger(jaeger.StdLogger))
	if err != nil {
		logrus.Fatalf("could not init tracing: %v\n", err)
	}
	defer func() {
		err := closer.Close()
		if err != nil {
			logrus.Error("error closing tracing: %v", err)
		}
	}()
	opentracing.SetGlobalTracer(tracer)

	db, err = gorm.Open(mysql.Open("user:password@tcp(localhost:3306)/flight?charset=utf8mb4&parseTime=True&loc=Local"), &gorm.Config{}) // TODO: change localhost to flight-db
	if err != nil {
		logrus.Fatalf("could not connect to database: %v", err)
	}

	r := gin.Default()
	r.GET("/api/flight-service/flight-v1/:ID", func(c *gin.Context) {
		ID, err := strconv.Atoi(c.Param("ID"))
		if err != nil {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		spanCtx, _ := tracer.Extract(
			opentracing.HTTPHeaders,
			opentracing.HTTPHeadersCarrier(c.Request.Header),
		)
		span := tracer.StartSpan("flight-service GET /:id", ext.RPCServerOption(spanCtx))
		defer span.Finish()

		ctx := opentracing.ContextWithSpan(c, span)
		flight := getFlight(ID, ctx)
		
		c.JSON(200, flight)
	})

	r.GET("/api/flight-service/flight-v1", func(c *gin.Context) {

	})

	err = r.Run(":8080")
	if err != nil {
		logrus.Fatalf("could not start http server: %v", err)
	}
	logrus.Info("server started on port 8080")
}

func getFlight(ID int, ctx context.Context) Flight {
	span, _ := opentracing.StartSpanFromContext(ctx, "flight-service MySQL GET Flight")
	defer span.Finish()

	flight := Flight{}
	db.Find(&flight, ID)
	return flight
}
