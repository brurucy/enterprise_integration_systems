package test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"rentit/pkg/domain"
	"testing"
)

const (
	port = 8080
)

func TestGetAllHttp(t *testing.T) {

	expectedCount := 8

	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/plants", port))

	if err != nil {
		t.Error("Failed to get all plants: " + err.Error())
		return 
	}

	defer resp.Body.Close()

	var data []*domain.Plant

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		t.Error("Couldn't decode the response: " + err.Error())
		return 
	}

	if len(data) != expectedCount{
		t.Error(fmt.Sprintf("Expected %d results, got %d", expectedCount, len(data)))
		return 
	}

	for _, plant := range data {
		if plant == nil{
			t.Error("One plant was nil")
		}
	}
}

func TestEstimatePriceHttp(t *testing.T) {
	verifyPrice(t, "http://localhost:%d/estimate?name=bulldozer&from=2020-01-01&to=2020-01-10", 45000)
	verifyPrice(t, "http://localhost:%d/estimate?name=forklift&from=2020-01-01&to=2020-01-03", 10000)
}

func verifyPrice(t *testing.T, url string, expected float32){
	resp, err := http.Get(fmt.Sprintf(url, port))

	if err != nil {
		t.Error("Failed to get estimate: " + err.Error())
		return 
	}

	defer resp.Body.Close()

	var data map[string]float32

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		t.Error("Couldn't decode the response: " + err.Error())
		return 
	}

	for key, _ := range data {
		if key != "price"{
			t.Error("Invalid field in the response")
			return 
		}
	}

	if _, ok := data["price"]; ok {
		if !ok{
			t.Error("\"price\" filed not present in response")
			return 
		}
	}

	if data["price"] != expected{
		t.Error("Wrong price returned")
		return 
	}
}

func TestAvailabilityHttp(t *testing.T) {
	verifyAvailability(t, "http://localhost:%d/availability?name=bulldozer&from=2021-10-19&to=2021-10-21", true)
	verifyAvailability(t, "http://localhost:%d/availability?name=forklift&from=2021-10-19&to=2021-10-21", true)
}

func verifyAvailability(t *testing.T, url string, expected bool){
	resp, err := http.Get(fmt.Sprintf(url, port))

	if err != nil {
		t.Error("Failed to get availability: " + err.Error())
		return 
	}

	defer resp.Body.Close()

	var data map[string]bool


	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		t.Error("Couldn't decode the response: " + err.Error())
		return 
	}

	for key, _ := range data {
		if key != "isAvailable"{
			t.Error("Invalid field in the response: " + err.Error())
		}
	}

	if _, ok := data["isAvailable"]; ok {
		if !ok{
			t.Error("\"isAvailable\" field not present in response")
			return
		}
	}

	if data["isAvailable"] != expected{
		t.Error("Wrong availability value returned")
	}
}