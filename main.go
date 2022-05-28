package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	firebase "firebase.google.com/go"
	"github.com/gocolly/colly"
	"google.golang.org/api/option"
)

// NBS link for bank page
var NBS_URL string = "https://nbs.rs/kursnaListaModul/zaDevize.faces?lang=lat"

// List of all monetes in which we are interested
var ALL_MONETES []string = []string{"EUR", "AUD", "CAD", "CNY", "HRK", "CZK", "DKK", "HUF", "INR", "JPY", "KWD", "NOK", "RUB", "SEK", "CHF", "GBP", "USD", "BYN", "RON", "TRY", "BGN", "BAM", "PLN"}

const HOURS = 24
const MINUTES = 60
const SECOND = 60

type NBSMonete struct {
	moneteName          string
	moneteCode          int64
	countryName         string
	moneteCountRelation int64
	toByCurse           float64
	toSellCurse         float64
	timestamp           time.Time
}

func main() {

	task(time.Now())
	tick := time.NewTicker(time.Second * HOURS * MINUTES * SECOND)
	go scheduler(tick)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs
	tick.Stop()
}

func scheduler(tick *time.Ticker) {
	for t := range tick.C {
		task(t)
	}
}

func task(t time.Time) {
	scrapeNbsMoneteData()
	fmt.Println("================ collecting done for day", time.Now())
}

func scrapeNbsMoneteData() {
	// Init map schema
	moneteMap := prepareMapSchema(ALL_MONETES)
	var activeObject *NBSMonete
	var possitionMappingCounter int = 0

	// Set domain boundary
	c := colly.NewCollector(
		colly.AllowedDomains("nbs.rs"),
	)

	// Parse all td tags from specific HTML page
	c.OnHTML("td", func(e *colly.HTMLElement) {
		// element value from specific table column
		elementValue := e.DOM.Text()

		activeMonete := activeSchemaElement(e.DOM.Text(), moneteMap)

		// workaround to fullfil data on exact spot
		if activeMonete == nil {
			possitionMappingCounter += 1
		} else {
			possitionMappingCounter = 0
			activeObject = activeMonete
		}

		// parse the element to right structure
		htmlElementDataParser(elementValue, activeObject, possitionMappingCounter)
	})

	c.OnError(func(r *colly.Response, err error) {
		fmt.Println("Request URL:", r.Request.URL, "failed with response:", r, "\nError:", err)
	})

	// Visit the specific website
	c.Visit(NBS_URL)

	// display collected data
	for _, v := range ALL_MONETES {
		fmt.Println("%+v\n", moneteMap[v])
	}

	dumpToFirebaseDB(moneteMap)
}

func prepareMapSchema(moneteList []string) map[string]*NBSMonete {
	var monetMap = make(map[string]*NBSMonete)
	for _, s := range moneteList {
		monetMap[s] = &NBSMonete{moneteName: s}
	}
	return monetMap
}

func activeSchemaElement(activeMonete string, moneteMapSchema map[string]*NBSMonete) *NBSMonete {
	if contains(ALL_MONETES, activeMonete) {
		return moneteMapSchema[activeMonete]
	}
	return nil
}

func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}

func htmlElementDataParser(elementValue string, activeObject *NBSMonete, possitionMappingCounter int) {
	if activeObject != nil {
		activeObject.timestamp = time.Now()

		switch possitionMappingCounter {
		case 1:
			intVar, _ := strconv.ParseInt(elementValue, 10, 64)
			activeObject.moneteCode = intVar
		case 2:
			activeObject.countryName = elementValue
		case 3:
			intVar, _ := strconv.ParseInt(elementValue, 10, 64)
			activeObject.moneteCountRelation = intVar
		case 4:
			floatVal, _ := strconv.ParseFloat(strings.Replace(elementValue, ",", ".", -1), 64)
			activeObject.toByCurse = floatVal
		case 5:
			floatVal, _ := strconv.ParseFloat(strings.Replace(elementValue, ",", ".", -1), 64)
			activeObject.toSellCurse = floatVal
		default:
			break
		}
	}
}

func dumpToFirebaseDB(moneteMap map[string]*NBSMonete) {
	// Save to firebase DB
	opt := option.WithCredentialsFile("./nbs-data-e2e97-firebase-adminsdk-szrjl-eaec7dfe8b.json")
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		fmt.Errorf("error initializing app: %v", err)
	}

	client, err := app.Firestore(context.Background())

	if err != nil {
		log.Fatalln(err)
	}

	for _, v := range ALL_MONETES {
		injectionObj := make(map[string]interface{})
		injectionObj["moneteName"] = moneteMap[v].moneteName
		injectionObj["moneteCode"] = &moneteMap[v].moneteCode
		injectionObj["countryName"] = &moneteMap[v].countryName
		injectionObj["moneteCountRelation"] = &moneteMap[v].moneteCountRelation
		injectionObj["toByCurse"] = &moneteMap[v].toByCurse
		injectionObj["toSellCurse"] = &moneteMap[v].toSellCurse
		injectionObj["timestamp"] = &moneteMap[v].timestamp

		result, _, err := client.Collection("monete").Add(context.Background(), injectionObj)

		if err != nil {
			log.Fatalln(err)
		}
		fmt.Println("Saved %+v\n", result)
	}

	defer client.Close()
}
