package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jamespearly/loggly"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

type DynamoItem struct {
	WeatherCondition string `json:"WeatherCondition"`
	Quote            string `json:"Quote"`
	City             string `json:"City"`
	Time             string `json:"Time"`
}

type weatherRequest struct {
	Current struct {
		WeatherDescription []string `json:"weather_descriptions"`
	} `json:"current"`
	Location struct {
		Name string `json:"name"`
		Time string `json:"localtime"`
	} `json:"location"`
}

type quotesRequest struct {
	Contents struct {
		Quotes []struct {
			Quote string `json:"quote"`
		} `json:"quotes"`
	} `json:"contents"`
}

func addItemtoDB(item DynamoItem) {
	// Start AWS Session (Env variables must be provided before hand)
	sess := session.Must(session.NewSessionWithOptions(session.Options{SharedConfigState: session.SharedConfigEnable}))

	// Create DynamoDB client
	svc := dynamodb.New(sess)
	_ = svc

	av, err := dynamodbattribute.MarshalMap(item)
	if err != nil {
		fmt.Println("Got error marshalling item:")
		fmt.Println(err.Error())
		os.Exit(1)
	}

	input := &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String("tkressma-WeatherQuotes"),
	}

	_, err = svc.PutItem(input)

	if err != nil {
		fmt.Println("Got error calling addItemtoDB: ")
		fmt.Println(err.Error())
		os.Exit(1)
	}
	fmt.Println("Item: " + item.WeatherCondition + "\nQuote: " + item.Quote + "\nCity: " + item.City + "\nTime: " + item.Time + "\n")

	// Loggly
	var tag string
	client := loggly.New(tag)
	client.EchoSend("info", "Item Received: "+item.WeatherCondition+"| Quote: "+item.Quote+"| City: "+item.City+"| Time: "+item.Time)
	fmt.Println()

}

func fetchWeatherCondition(city string) []string {
	response, err := http.Get("https://api.weatherstack.com/current?access_key=3af39d571bfc45b0285388d03bc69971&query=" + city + "%New%York&units=f&prettyprint=true&alt=json")
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	responseData, err := ioutil.ReadAll(response.Body)

	if err != nil {
		log.Fatal(err)
	}

	var weatherData weatherRequest
	err2 := json.Unmarshal(responseData, &weatherData)
	if err2 != nil {
		log.Fatal(err2)
	}

	// Go slice containing Weather Condition in index 0, Location in Index 1, Time in index 2
	requestData := make([]string, 0, 3)
	requestData = append(requestData, weatherData.Current.WeatherDescription[0], weatherData.Location.Name, weatherData.Location.Time)

	return requestData

}

func fetchQuotes(term string) string {
	if strings.Contains(term, " ") {
		term = strings.ReplaceAll(term, " ", "%20")
	}

	response, err := http.Get("https://quotes.rest/quote/search?minlength=100&maxlength=300&query=" + term + "&private=false&language=en&limit=1&sfw=false&api_key=kuS8QPhtGOWGQIW6dq_KVQeF")
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	responseData, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}

	var quoteData quotesRequest
	err2 := json.Unmarshal(responseData, &quoteData)
	if err2 != nil {
		log.Fatal(err2)
	}

	return quoteData.Contents.Quotes[0].Quote

}

func storeAndPoll(pollingRateSecs time.Duration) {
	time.Sleep(pollingRateSecs * time.Second)
	fmt.Println("DEBUG: POLLING NOW")
	addItemtoDB(DynamoItem{WeatherCondition: fetchWeatherCondition("Oswego")[0], Quote: fetchQuotes(fetchWeatherCondition("Oswego")[0]), City: fetchWeatherCondition("Oswego")[1], Time: fetchWeatherCondition("Oswego")[2]})
	addItemtoDB(DynamoItem{WeatherCondition: fetchWeatherCondition("NewYork")[0], Quote: fetchQuotes(fetchWeatherCondition("NewYork")[0]), City: fetchWeatherCondition("NewYork")[1], Time: fetchWeatherCondition("NewYork")[2]})
	addItemtoDB(DynamoItem{WeatherCondition: fetchWeatherCondition("Yonkers")[0], Quote: fetchQuotes(fetchWeatherCondition("Yonkers")[0]), City: fetchWeatherCondition("Yonkers")[1], Time: fetchWeatherCondition("Yonkers")[2]})
	fmt.Println("DEBUG: FINISHED POLLING")
}

func main() {
	for true {
		storeAndPoll(43200)
	}
}
