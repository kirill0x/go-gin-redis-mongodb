package main

import (
	"context"
	"fmt"
	"gopkg.in/yaml.v3"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Config struct {
	Mongo struct {
		Host     string `yaml:"host"`
		Port     string `yaml:"port"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
		Database string `yaml:"database"`
		Uri      string `yaml:"uri"`
	} `yaml:"mongo"`
	Redis struct {
		Host     string `yaml:"host"`
		Port     string `yaml:"port"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
		Database string `yaml:"database"`
		Uri      string `yaml:"uri"`
	} `yaml:"redis"`
}

const (
	databaseName   = "blog"
	collectionName = "views"
)

func getAnalyticsDataByTitle(ctx *gin.Context, mongoClient *mongo.Client, title string) (bson.D, error) {
	coll := mongoClient.Database(databaseName).Collection(collectionName)
	var result bson.D
	err := coll.FindOne(ctx, bson.D{{"title", title}}).Decode(&result)
	if err != nil {
		log.Error().Err(err).Msg("error occured while connecting to mongo")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get analytics data"})
		return nil, err
	}
	return result, err
}

func getAllBlogViews(ctx *gin.Context, mongoClient *mongo.Client) ([]bson.M, error) {
	coll := mongoClient.Database(databaseName).Collection(collectionName)
	cursor, err := coll.Find(context.TODO(), bson.D{})
	if err != nil {
		panic(err)
	}
	var results []bson.M
	if err = cursor.All(ctx, &results); err != nil {
		log.Error().Err(err).Msg("error occured while connecting to mongo")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get analytics data"})
		return nil, err
	}
	return results, err
}

func main() {

	// open config file
	file, err := os.Open("../../config.yml")
	if err != nil {
		fmt.Println("error opening file:", err)
	}
	defer file.Close()

	// read config file
	var config Config
	decoder := yaml.NewDecoder(file)
	err = decoder.Decode(&config)
	if err != nil {
		fmt.Println("error:", err)
	}

	mongoClient, err := mongo.NewClient(options.Client().ApplyURI(config.Mongo.Uri))
	if err != nil {
		log.Error().Err(err).Msg("error occured while connecting to mongo")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = mongoClient.Connect(ctx)
	if err != nil {
		log.Error().Err(err).Msg("error occured while connecting to mongo")
	}
	defer mongoClient.Disconnect(ctx)
	router := gin.Default()

	router.GET("/views/:title", func(ctx *gin.Context) {
		title := ctx.Param("title")
		result, err := getAnalyticsDataByTitle(ctx, mongoClient, title)
		if err != nil {
			log.Error().Err(err).Msg("error occured")
		}
		ctx.JSON(http.StatusOK, gin.H{
			"Data": result,
		})
	})

	router.GET("/views", func(ctx *gin.Context) {
		result, err := getAllBlogViews(ctx, mongoClient)
		if err != nil {
			log.Error().Err(err).Msg("error occured")
		}
		ctx.JSON(http.StatusOK, gin.H{
			"Data": result,
		})
	})
	router.Run(":8081")
}
