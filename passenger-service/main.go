package main

import (
	"github.com/gin-gonic/gin"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/sirupsen/logrus"
	"github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/config"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var db *gorm.DB

type Passenger struct {
	PassengerID int
	FlightID int
	Firstname string
	Surname string
}

func main() {
	connectDb()
	_, err := initializeTracer()
	if err != nil {
		logrus.Fatalf("could not initialize tracing: %v", err)
	}

	r := gin.Default()
	r.GET("/api/passenger-service/passenger-v1", getPassengers)

	err = r.Run(":8080")
	if err != nil {
		logrus.Fatalf("could not start http server: %v", err)
	}
	logrus.Info("server started on port 8080")
}

func getPassengers(c *gin.Context) {
	flightID, err := strconv.Atoi(c.Query("flightId"))
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	tracer := opentracing.GlobalTracer()
	reqCtx, _ := tracer.Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(c.Request.Header))
	parentSpan := tracer.StartSpan("passenger-service GET /", ext.RPCServerOption(reqCtx))
	defer parentSpan.Finish()

	ctx := opentracing.ContextWithSpan(c, parentSpan, )

	selectSpan, ctx := opentracing.StartSpanFromContext(ctx, "passenger-service: MySQL Select Passengers")
	var passengers []Passenger
	db.Where("flight_id = ?", flightID).Find(&passengers)
	selectSpan.Finish()

	c.JSON(200, passengers)
}

func initializeTracer() (io.Closer, error) {
	cfg := config.Configuration{
		ServiceName: "passenger-service",
		Sampler:     &config.SamplerConfig{
			Type:  jaeger.SamplerTypeConst,
			Param: 1,
		},
		Reporter:    &config.ReporterConfig{
			LogSpans: true,
			LocalAgentHostPort: "jaeger:6831",
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
func connectDb() {
	con, err := gorm.Open(mysql.Open("root:secret@tcp(passenger-db:3306)/passengers?charset=utf8mb4&parseTime=True&loc=Local"), &gorm.Config{})
	if err != nil && strings.Contains(err.Error(), "connection refused") {
		if retries >= 10 {
			logrus.Fatalf("could not connect to database: %v", err)
		}
		retries++
		logrus.Warnf("connection to database failed. Retry %d", retries)
		time.Sleep(20 * time.Second)
		connectDb()
		return
	}
	if err != nil {
		logrus.Fatalf("could not connect to database: %v", err)
	}
	db = con
}