package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/sirupsen/logrus"
	"github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/config"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"io"
	"os"
	"strings"
	"time"
)

var (
	JAEGER_HOST_PORT = os.Getenv("JAEGER_HOST_PORT")
	SERVICE_PORT = os.Getenv("SERVICE_PORT")
	MYSQL_HOST = os.Getenv("MYSQL_HOST")
	MYSQL_PORT = os.Getenv("MYSQL_PORT")
)

type Passenger struct {
	PassengerID int
	FlightID int
	Firstname string
	Surname string
}

func main() {
	db := connect()
	closer, err := initTracer()
	if err != nil {
		logrus.Fatalf("could not initialize tracing: %v", err)
	}
	defer closer.Close()

	r := gin.Default()
	r.GET("/api/passenger-service/passenger-v1", getPassengers(db))

	err = r.Run(fmt.Sprintf(":%s", SERVICE_PORT))
	if err != nil {
		logrus.Fatalf("could not start http server: %v", err)
	}
	logrus.Info("server started on port 8080")
}

func getPassengers(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		flightID := c.Query("flightId")

		tracer := opentracing.GlobalTracer()
		reqCtx, _ := tracer.Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(c.Request.Header))
		parentSpan := tracer.StartSpan("passenger-service GET /", ext.RPCServerOption(reqCtx))
		defer parentSpan.Finish()

		ctx := opentracing.ContextWithSpan(c, parentSpan, )

		selectSpan, ctx := opentracing.StartSpanFromContext(ctx, "passenger-service: MySQL Select Passengers")
		var passengers []Passenger
		if flightID != "" {
			db.Where("flight_id = ?", flightID).Find(&passengers)
			c.JSON(200, passengers)
			selectSpan.Finish()
			return
		}
		db.Find(&passengers)
		selectSpan.Finish()

		c.JSON(200, passengers)
	}
}

func initTracer() (io.Closer, error) {
	cfg := config.Configuration{
		ServiceName: "passenger-service",
		Sampler:     &config.SamplerConfig{
			Type:  jaeger.SamplerTypeConst,
			Param: 1,
		},
		Reporter:    &config.ReporterConfig{
			LogSpans: true,
			LocalAgentHostPort: JAEGER_HOST_PORT,
		},
	}

	tracer, closer, err := cfg.NewTracer(config.Logger(jaeger.StdLogger))
	if err != nil {
		return nil, err
	}
	opentracing.SetGlobalTracer(tracer)
	return closer, err
}

var retries = 0
func connect() *gorm.DB {
	dsn := fmt.Sprintf("root:secret@tcp(%s:%s)/passengers?charset=utf8mb4&parseTime=True&loc=Local", MYSQL_HOST, MYSQL_PORT)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil && strings.Contains(err.Error(), "connection refused") {
		if retries >= 10 {
			logrus.Fatalf("could not connect to database: %v", err)
		}
		retries++
		logrus.Warnf("connection to database failed. Retry %d", retries)
		time.Sleep(20 * time.Second)
		return connect()
	}
	if err != nil {
		logrus.Fatalf("could not connect to database: %v", err)
	}
	return db
}