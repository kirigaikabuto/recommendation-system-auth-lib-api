package main

import (
	"fmt"
	"github.com/djumanoff/amqp"
	"github.com/gin-gonic/gin"
	"github.com/itsjamie/gin-cors"
	"github.com/joho/godotenv"
	protos2 "github.com/kirigaikabuto/RecommendationSystemPythonApi/protos"
	auth_lib "github.com/kirigaikabuto/recommendation-system-auth-lib"
	auth_lib_tkn "github.com/kirigaikabuto/recommendation-system-auth-lib/auth"
	setdata_common "github.com/kirigaikabuto/setdata-common"
	"github.com/urfave/cli"
	"google.golang.org/grpc"
	"os"
	"time"
)

var (
	configPath = ""
	version    = "0.0.1"
	amqpUrl    = ""
	redisHost  = ""
	redisPort  = ""
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
	fmt.Println(configPath)
	if configPath != "" {
		godotenv.Overload(configPath)
	} else {
		godotenv.Overload("helm/dev.env")
	}
	amqpUrl = os.Getenv("AMQP_URL")
	redisHost = os.Getenv("REDIS_HOST")
	redisPort = os.Getenv("REDIS_PORT")
	fmt.Println(amqpUrl)
	fmt.Println(redisHost)
	fmt.Println(redisPort)
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
		port = "8000"
	}
	parseEnvFile()
	amqpConfig := amqp.Config{
		AMQPUrl: amqpUrl,
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
		Host: redisHost,
		Port: redisPort,
	})
	if err != nil {
		return err
	}
	mdw := auth_lib_tkn.NewMiddleware(redisStore)

	amqpRequests := auth_lib.NewAmqpRequests(clt)
	//grpc client
	connGrpc, err := grpc.Dial(":50051", grpc.WithInsecure())
	if err != nil {
		return err
	}
	defer connGrpc.Close()
	clientGrpc := protos2.NewGreeterClient(connGrpc)
	service := auth_lib.NewAuthLibService(amqpRequests, redisStore, clientGrpc)
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

	moviesGroup := r.Group("/movies")
	{
		moviesGroup.GET("/", httpEndpoints.MakeListMovies())
		moviesGroup.GET("/collrec", mdw.MakeMiddleware(), httpEndpoints.MakeListCollaborativeFiltering())
		moviesGroup.GET("/content", mdw.MakeMiddleware(), httpEndpoints.MakeContentBasedFiltering())
	}
	r.Use(cors.Middleware(cors.Config{
		Origins:         "*",
		Methods:         "GET, PUT, POST, DELETE, OPTIONS",
		RequestHeaders:  "Origin, Authorization, Content-Type",
		ExposedHeaders:  "",
		MaxAge:          50 * time.Second,
		Credentials:     false,
		ValidateHeaders: false,
	}))
	fmt.Println("server start in port:" + port)
	return r.Run("0.0.0.0:" + port)
}
