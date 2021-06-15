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
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	JAEGER_HOST_PORT = os.Getenv("JAEGER_HOST_PORT")
	SERVICE_PORT = os.Getenv("SERVICE_PORT")
	MYSQL_HOST = os.Getenv("MYSQL_HOST")
	MYSQL_PORT = os.Getenv("MYSQL_PORT")
)

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
	db := connect()
	closer, err := initTracer()
	if err != nil {
		logrus.Fatalf("could not initialize tracing: %v", err)
	}
	defer closer.Close()

	r := gin.Default()
	r.GET("/api/flight-service/flight-v1/:ID", getFlight(db))

	err = r.Run(fmt.Sprintf(":%s", SERVICE_PORT))
	if err != nil {
		logrus.Fatalf("could not start http server: %v", err)
	}
	logrus.Info("server started on port 8080")
}

func getFlight(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ID, err := strconv.Atoi(c.Param("ID"))
		if err != nil {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		tracer := opentracing.GlobalTracer()
		parentSpan := tracer.StartSpan("flight-service GET /:ID")
		parentSpan.SetTag("flight id", ID)
		defer parentSpan.Finish()

		ctx := opentracing.ContextWithSpan(c, parentSpan)

		selectSpan, ctx := opentracing.StartSpanFromContext(ctx, "flight-service: MySQL Select Flight")
		selectSpan.SetTag("database type", db.Name())
		selectSpan.SetTag("database host", MYSQL_HOST)
		selectSpan.SetTag("database port", MYSQL_PORT)
		selectSpan.SetTag("database schema", "passengers")
		selectSpan.SetTag("flight id", ID)
		flight := Flight{}
		db.Find(&flight, ID)
		selectSpan.Finish()

		passengers, _ := findPassengers(ID, ctx)
		flight.Passengers = passengers
		c.JSON(200, flight)
	}
}

func findPassengers(flightID int, ctx context.Context) ([]Passenger, error) {
	url := fmt.Sprintf("http://passenger-service:8080/api/passenger-service/passenger-v1?flightId=%d", flightID)
	span, ctx := opentracing.StartSpanFromContext(ctx, "flight-service: passenger-service GET /api/passenger-service/passenger-v1")
	span.SetTag("flight id", flightID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logrus.Errorf("failed to initialize request: %v", err)
		span.SetTag("error", true)
		return []Passenger{}, err
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
		span.SetTag("error", true)
		logrus.Errorf("failed to get passengers: %v", err)
		return []Passenger{}, err
	}
	span.Finish()

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		span.SetTag("error", true)
		logrus.Errorf("could not read response body: %v", err)
		return []Passenger{}, err
	}
	var passengers []Passenger
	err = json.Unmarshal(body, &passengers)
	if err != nil {
		span.SetTag("error", true)
		logrus.Errorf("could not unmarshal response body: %v", err)
		return []Passenger{}, err
	}
	return passengers, nil
}

func initTracer() (io.Closer, error) {
	cfg := config.Configuration{
		ServiceName: "flight-service",
		Sampler: &config.SamplerConfig{
			Type:  jaeger.SamplerTypeConst,
			Param: 1,
		},
		Reporter: &config.ReporterConfig{
			LogSpans:           true,
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
	dsn := fmt.Sprintf("root:secret@tcp(%s:%s)/flights?charset=utf8mb4&parseTime=True&loc=Local", MYSQL_HOST, MYSQL_PORT)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil && strings.Contains(err.Error(), "connection refused") {
		if retries >= 10 {
			logrus.Fatalf("could not connectDb to database: %v", err)
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