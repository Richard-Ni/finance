package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
)

const apiKey = "XXXXXXX"

type StockData struct {
	MetaData struct {
		Symbol string `json:"2. Symbol"`
	} `json:"Meta Data"`
	TimeSeries   map[string]map[string]string `json:"Time Series (Daily)"`
	VolumeSeries map[string]string            `json:"-"`
}

type EChartsData struct {
	Symbol       string     `json:"symbol"`
	Dates        []string   `json:"dates"`
	KLineValues  [][]string `json:"kLineValues"`
	VolumeValues []string   `json:"volumeValues"`
}

func main() {
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/getStockData", getStockDataHandler)
	http.HandleFunc("/static/", staticHandler)

	fmt.Println("Server running on http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	html, err := ioutil.ReadFile("index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Fprint(w, string(html))
}

func staticHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, r.URL.Path[1:])
}

func getStockDataHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	var requestData struct {
		Symbol string `json:"symbol"`
	}
	err := json.NewDecoder(r.Body).Decode(&requestData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	stockData, err := fetchStockData(requestData.Symbol)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	echartsData := generateEChartsData(stockData)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(echartsData)
}

func fetchStockData(symbol string) (*StockData, error) {
	url := fmt.Sprintf("https://www.alphavantage.co/query?function=TIME_SERIES_DAILY_ADJUSTED&symbol=%s&apikey=%s", symbol, apiKey)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Error getting stock data: %v", resp.Status)
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	stockData := &StockData{VolumeSeries: make(map[string]string)}
	err = json.Unmarshal(data, stockData)
	if err != nil {
		return nil, err
	}

	for date, values := range stockData.TimeSeries {
		stockData.VolumeSeries[date] = values["6. volume"]
	}

	return stockData, nil
}

func generateEChartsData(stockData *StockData) *EChartsData {
	var dates []string
	for date := range stockData.TimeSeries {
		dates = append(dates, date)
	}

	sort.Slice(dates, func(i, j int) bool {
		return dates[i] < dates[j]
	})

	var kLineValues [][]string
	var volumeValues []string

	for _, date := range dates {
		data := stockData.TimeSeries[date]
		kLineValue := []string{
			data["1. open"],
			data["4. close"],
			data["3. low"],
			data["2. high"],
		}
		kLineValues = append(kLineValues, kLineValue)
		volumeValues = append(volumeValues, stockData.VolumeSeries[date])
	}

	return &EChartsData{
		Symbol:       stockData.MetaData.Symbol,
		Dates:        dates,
		KLineValues:  kLineValues,
		VolumeValues: volumeValues,
	}
}
