package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
	"github.com/microcosm-cc/bluemonday"

	"github.com/gorilla/mux"
	"github.com/jamespearly/loggly"
)

func AllHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var tag string
	client := loggly.New(tag)

	client.EchoSend("info", "/all endpoint called")

	w.Write(generateAllDBJson())
}

func StatusHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var tag string
	client := loggly.New(tag)

	client.EchoSend("info", "/status endpoint called")

	w.Write(DynamoDBInformation())
}

func SearchHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var tag string
	client := loggly.New(tag)

	client.EchoSend("info", "/search endpoint called")

	// Policy to sanitize incoming query requests
	policy := bluemonday.StrictPolicy()

	query := req.URL.Query()

	city := policy.Sanitize(query.Get("city"))
	weathercondition := policy.Sanitize(query.Get("weathercondition"))

	// If a query parameter is anything other than city or weathercondition, return 400.
	if city == "" && weathercondition == "" || len(query) == 0 {
		w.WriteHeader(400)
	}

	allJSON := []DynamoItem{}

	err := json.Unmarshal([]byte(generateAllDBJson()), &allJSON)
	if err != nil {
		fmt.Println(err)
	}

	results := []DynamoItem{}

	for _, item := range allJSON {
		var citySearch = city
		var weatherConditionSearch = weathercondition

		if item.City == citySearch {
			results = append(results, item)
		}

		if item.WeatherCondition == weatherConditionSearch {
			results = append(results, item)
		}

	}

	// If the search returns no results, return 404.
	if len(results) == 0 {
		w.WriteHeader(404)
	} else {
		w.Write(generateSearchJson(results))
	}

}

type DynamoItem struct {
	WeatherCondition string `json:"WeatherCondition"`
	Quote            string `json:"Quote"`
	City             string `json:"City"`
	Time             string `json:"Time"`
}

type DatabaseInfo struct {
	Name      string `json:"TableName"`
	ItemCount *int64 `json:"ItemCount"`
}

func RetrieveDB() []DynamoItem {
	// Start AWS Session (Env variables must be provided before hand)
	mySession := session.Must(session.NewSessionWithOptions(session.Options{SharedConfigState: session.SharedConfigEnable}))

	// Create DynamoDB client
	svc := dynamodb.New(mySession)
	_ = svc

	//Using Scan API and Query Projections to scan the DB
	proj := expression.NamesList(expression.Name("WeatherCondition"), expression.Name("Quote"), expression.Name("City"), expression.Name("Time"))
	expr, err := expression.NewBuilder().WithProjection(proj).Build()
	if err != nil {
		fmt.Println("Got error building expression:")
		fmt.Println(err.Error())
		os.Exit(1)
	}

	// Build the query input parameters
	params := &dynamodb.ScanInput{
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		FilterExpression:          expr.Filter(),
		ProjectionExpression:      expr.Projection(),
		TableName:                 aws.String("tkressma-WeatherQuotes"),
	}

	// Make the DynamoDB Query API call
	result, err := svc.Scan(params)
	if err != nil {
		fmt.Println("Query API call failed:")
		fmt.Println((err.Error()))
		os.Exit(1)
	}

	data := make([]DynamoItem, 0)

	for _, i := range result.Items {
		item := DynamoItem{}
		err = dynamodbattribute.UnmarshalMap(i, &item)
		if err != nil {
			fmt.Println("Got error unmarshalling:")
			fmt.Println(err.Error())
			os.Exit(1)
		}
		data = append(data, item)
	}

	return data
}

func generateSearchJson(results []DynamoItem) []byte {
	// Generate Json
	var buff bytes.Buffer
	buff.WriteByte('[')

	for i, item := range results {
		marsh, _ := json.Marshal(item)
		if i != 0 {
			buff.WriteByte(',')
		}
		buff.Write(marsh)
	}

	buff.WriteByte(']')
	return buff.Bytes()
}

func generateAllDBJson() []byte {
	// Generate Json
	var buff bytes.Buffer
	buff.WriteByte('[')

	for i, item := range RetrieveDB() {
		marsh, _ := json.Marshal(item)
		if i != 0 {
			buff.WriteByte(',')
		}
		buff.Write(marsh)
	}

	buff.WriteByte(']')
	return buff.Bytes()
}

func DynamoDBInformation() []byte {
	// Start AWS Session (Env variables must be provided before hand)
	mySession := session.Must(session.NewSessionWithOptions(session.Options{SharedConfigState: session.SharedConfigEnable}))

	// Create a DynamoDB client from just a session.
	svc := dynamodb.New(mySession)
	_ = svc

	input := &dynamodb.DescribeTableInput{
		TableName: aws.String("tkressma-WeatherQuotes"),
	}

	result, _ := svc.DescribeTable(input)

	var Info DatabaseInfo
	Info = DatabaseInfo{Name: *result.Table.TableName, ItemCount: result.Table.ItemCount}
	marshalledJSON, _ := json.Marshal(&Info)

	return marshalledJSON
}

func main() {
	GorillaMux := mux.NewRouter()

	GorillaMux.HandleFunc("/tkressma/all", AllHandler).Methods("GET")
	GorillaMux.HandleFunc("/tkressma/status", StatusHandler).Methods("GET")
	GorillaMux.HandleFunc("/tkressma/search", SearchHandler).Methods("GET")

	log.Println("Server is running...")
	log.Fatal(http.ListenAndServe(":8080", GorillaMux))

}
