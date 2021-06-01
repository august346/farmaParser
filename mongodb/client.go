package mongodb

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

const (
	URI     string = "mongodb://root:example@localhost:27017"
	DB_NAME string = "farma"
)

type MongoClient struct {
	client *mongo.Client
}

func client() *mongo.Client {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(URI))
	if err != nil {
		log.Fatal(err)
	}

	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		log.Fatal(err)
	}

	return client
}

func NewMongoClient() *MongoClient {
	return &MongoClient{client: client()}
}

func (mc *MongoClient) InsertMany(collectionName string, items []interface{}) int {
	collection := mc.client.Database(DB_NAME).Collection(collectionName)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := collection.InsertMany(ctx, items)
	if err != nil {
		log.Fatal(err)
	}

	cnt := len(result.InsertedIDs)

	return cnt
}

func (mc *MongoClient) InsertOne(collectionName string, item interface{}) {
	collection := mc.client.Database(DB_NAME).Collection(collectionName)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := collection.InsertOne(ctx, item)
	if err != nil {
		log.Fatal(err)
	}
}
