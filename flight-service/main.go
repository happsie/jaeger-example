package main

import (
	"github.com/gin-gonic/gin"
	"github.com/opentracing/opentracing-go"
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
	cfg := config.Configuration{
		ServiceName: "flight-service",
		Sampler:     &config.SamplerConfig{
			Type:  jaeger.SamplerTypeConst,
			Param: 1,
		},
		Reporter:    &config.ReporterConfig{
			LogSpans: true,
		},
	}

	tracer, closer, err := cfg.NewTracer(config.Logger(jaeger.StdLogger))
	if err != nil {
		logrus.Fatalf("could not init tracing: %v\n", err)
	}
	opentracing.SetGlobalTracer(tracer)
	defer closer.Close()

	db, err = gorm.Open(mysql.Open("root:secret@tcp(flight-db:3306)/flights?charset=utf8mb4&parseTime=True&loc=Local"), &gorm.Config{})
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
		span := tracer.StartSpan("flight-service GET /:id")
		defer span.Finish()

		flight := getFlight(ID, span)
		
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

func getFlight(ID int, parent opentracing.Span) Flight {
	tracer := opentracing.GlobalTracer()
	span := tracer.StartSpan(
		"flight-service MySQL GET Flight",
		opentracing.ChildOf(parent.Context()),
	)
	defer span.Finish()

	flight := Flight{}
	db.Find(&flight, ID)
	return flight
}
