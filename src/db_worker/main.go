package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gopkg.in/yaml.v2"
	"os"
)

type BlogPost struct {
	Title  string `json:"title"`
	Author string `json:"author"`
	Body   string `json:"body"`
}

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
	collectionName = "posts"
)

func insertDoc(mongoClient *mongo.Client, post BlogPost) (*mongo.InsertOneResult, error) {
	coll := mongoClient.Database(databaseName).Collection(collectionName)
	result, err := coll.InsertOne(context.TODO(), post)

	if err != nil {
		log.Error().Err(err).Msg("error occured while inserting doc to mongo")
		return nil, err
	}
	return result, err
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

	// connect to mongo
	mongoClient, err := mongo.NewClient(options.Client().ApplyURI(config.Mongo.Uri))
	if err != nil {
		log.Error().Err(err).Msg("error occured while connecting to mongo")
	}
	ctx := context.Background()
	err = mongoClient.Connect(ctx)
	if err != nil {
		log.Error().Err(err).Msg("error occured while connecting to mongo")
	}
	defer mongoClient.Disconnect(ctx)

	// connect to redis
	opt, err := redis.ParseURL(config.Redis.Uri)
	if err != nil {
		log.Error().Err(err).Msg("error occured while connecting to redis")
	}
	rdb := redis.NewClient(opt)

	// worker execution
	for {
		result, err := rdb.BLPop(ctx, 0, "queue:new-post").Result()
		if err != nil {
			log.Error().Err(err).Msg("error occured while reading from redis")
			continue
		}

		post := BlogPost{}
		err = json.Unmarshal([]byte(result[1]), &post)
		if err != nil {
			log.Error().Err(err).Msg("error occured while decoding response into Post object")
		}
		insertDoc(mongoClient, post)
		fmt.Println(result)

	}
}
