package main

import (
	"fmt"
	"github.com/djumanoff/amqp"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	auth_lib "github.com/kirigaikabuto/recommendation-system-auth-lib"
	auth_lib_tkn "github.com/kirigaikabuto/recommendation-system-auth-lib/auth"
	setdata_common "github.com/kirigaikabuto/setdata-common"
	"github.com/urfave/cli"
	"os"
	"strconv"
)

var (
	configPath = ".env"
	version    = "0.0.1"
	amqpHost   = ""
	amqpPort   = 0
	flags      = []cli.Flag{
		&cli.StringFlag{
			Name:        "config, c",
			Usage:       "path to .env config file",
			Destination: &configPath,
		},
	}
)

func parseEnvFile() {
	// Parse config file (.env) if path to it specified and populate env vars
	if configPath != "" {
		godotenv.Overload(configPath)
	}
	amqpHost = os.Getenv("RABBIT_HOST")
	amqpPortStr := os.Getenv("RABBIT_PORT")
	amqpPort, _ = strconv.Atoi(amqpPortStr)
	if amqpPort == 0 {
		amqpPort = 5672
	}
	if amqpHost == "" {
		amqpHost = "localhost"
	}
}

func main() {
	app := cli.NewApp()
	app.Name = "recommendation system auth lib api"
	app.Description = ""
	app.Usage = "recommendation system auth lib api"
	app.UsageText = "recommendation system auth lib api"
	app.Version = version
	app.Flags = flags
	app.Action = run

	err := app.Run(os.Args)
	if err != nil {
		fmt.Println(err)
	}
}

func run(c *cli.Context) error {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	parseEnvFile()
	amqpConfig := amqp.Config{
		AMQPUrl: "amqp://" + amqpHost + ":" + strconv.Itoa(amqpPort),
	}
	sess := amqp.NewSession(amqpConfig)
	err := sess.Connect()
	if err != nil {
		return err
	}
	clt, err := sess.Client(amqp.ClientConfig{})
	if err != nil {
		return err
	}
	redisStore, err := auth_lib_tkn.NewTokenStore(auth_lib_tkn.RedisConfig{
		Host: "localhost",
		Port: "6379",
	})
	if err != nil {
		return err
	}
	mdw := auth_lib_tkn.NewMiddleware(redisStore)

	amqpRequests := auth_lib.NewAmqpRequests(clt)
	service := auth_lib.NewAuthLibService(amqpRequests, redisStore)
	httpEndpoints := auth_lib.NewHttpEndpoints(setdata_common.NewCommandHandler(service))

	gin.SetMode(gin.ReleaseMode)

	r := gin.Default()

	authGroup := r.Group("/auth")
	{
		authGroup.POST("/login", httpEndpoints.MakeLoginEndpoint())
		authGroup.POST("/register", httpEndpoints.MakeRegisterEndpoint())
	}

	scoreGroup := r.Group("/score", mdw.MakeMiddleware())
	{
		scoreGroup.POST("/", httpEndpoints.MakeCreateScoreEndpoint())
		scoreGroup.GET("/", httpEndpoints.MakeListScoreEndpoint())
	}

	moviesGroup := r.Group("/movies", mdw.MakeMiddleware())
	{
		moviesGroup.GET("/", httpEndpoints.MakeListMovies())
	}
	return r.Run()
}
