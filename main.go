package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gocolly/colly"
)

// NBS link for bank page
var NBS_URL string = "https://nbs.rs/kursnaListaModul/zaDevize.faces?lang=lat"

// List of all monetes in which we are interested
var ALL_MONETES []string = []string{"EUR", "AUD", "CAD", "CNY", "HRK", "CZK", "DKK", "HUF", "INR", "JPY", "KWD", "NOK", "RUB", "SEK", "CHF", "GBP", "USD", "BYN", "RON", "TRY", "BGN", "BAM", "PLN"}

type NBSMonete struct {
	moneteName          string
	moneteCode          int64
	countryName         string
	moneteCountRelation int64
	toByCurse           float64
	toSellCurse         float64
}

func main() {
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