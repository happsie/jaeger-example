package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/config"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var db *gorm.DB

type Flight struct {
	FlightID    int
	Name        string
	Destination string
	Passengers   []Passenger `gorm:"-"`
}

type Passenger struct {
	PassengerID int
	FlightID    int
	Firstname   string
	Surname     string
}

func main() {
	connectDb()
	_, err := initializeTracer()
	if err != nil {
		logrus.Fatalf("could not initialize tracing: %v", err)
	}

	r := gin.Default()
	r.GET("/api/flight-service/flight-v1/:ID", getFlight)

	err = r.Run(":8080")
	if err != nil {
		logrus.Fatalf("could not start http server: %v", err)
	}
	logrus.Info("server started on port 8080")
}

func getFlight(c *gin.Context) {
	ID, err := strconv.Atoi(c.Param("ID"))
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	tracer := opentracing.GlobalTracer()
	parentSpan := tracer.StartSpan("flight-service GET /:ID")
	defer parentSpan.Finish()

	ctx := opentracing.ContextWithSpan(c, parentSpan)

	selectSpan, ctx := opentracing.StartSpanFromContext(ctx, "flight-service: MySQL Select Flight")
	flight := Flight{}
	db.Find(&flight, ID)
	selectSpan.Finish()

	flight.Passengers = findPassengers(ID, ctx)
	c.JSON(200, flight)
}

func findPassengers(flightID int, ctx context.Context) []Passenger {
	url := fmt.Sprintf("http://passenger-service:8080/api/passenger-service/passenger-v1?flightId=%d", flightID)
	span, ctx := opentracing.StartSpanFromContext(ctx, "flight-service: passenger-service GET /api/passenger-service/passenger-v1" + url)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logrus.Errorf("failed to initialize request: %v", err)
		return []Passenger{}
	}
	err = span.Tracer().Inject(
		span.Context(),
		opentracing.HTTPHeaders,
		opentracing.HTTPHeadersCarrier(req.Header),
	)
	if err != nil {
		logrus.Warn("Error while injecting headers: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logrus.Errorf("failed to get passengers: %v", err)
		return []Passenger{}
	}
	span.Finish()

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logrus.Errorf("could not read response body: %v", err)
		return []Passenger{}
	}
	var passengers []Passenger
	err = json.Unmarshal(body, &passengers)
	if err != nil {
		logrus.Errorf("could not unmarshal response body: %v", err)
		return []Passenger{}
	}
	return passengers
}

func initializeTracer() (io.Closer, error) {
	cfg := config.Configuration{
		ServiceName: "flight-service",
		Sampler: &config.SamplerConfig{
			Type:  jaeger.SamplerTypeConst,
			Param: 1,
		},
		Reporter: &config.ReporterConfig{
			LogSpans:           true,
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
	con, err := gorm.Open(mysql.Open("root:secret@tcp(flight-db:3306)/flights?charset=utf8mb4&parseTime=True&loc=Local"), &gorm.Config{})
	// Resilient reconnect to db. Starting with docker-compose needs to wait for mysql too accept connection
	if err != nil && strings.Contains(err.Error(), "connection refused") {
		if retries >= 10 {
			logrus.Fatalf("could not connectDb to database: %v", err)
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