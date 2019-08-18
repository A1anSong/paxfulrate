package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"github.com/tidwall/gjson"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

var paxfulAPIKey = "yLc8DU65v8EA6WAtA5Esv6ZFRiy0aJbV"
var paxfulSecretAPIKey = "ZvM9ohuJrI0Q8kSAzqZ3lj18tWISFclT"

func getBitcoinCNY() float64 {
	price := 0.0

	nonce := fmt.Sprintf("%d", time.Now().Unix())

	values := url.Values{}
	values.Add("apikey", paxfulAPIKey)
	values.Add("nonce", nonce)

	values.Add("offer_type", "buy")
	values.Add("payment_method", "alipay")
	values.Add("currency_code", "CNY")

	payload := values.Encode()

	mac := hmac.New(sha256.New, []byte(paxfulSecretAPIKey))
	mac.Write([]byte(payload))
	apiseal := hex.EncodeToString(mac.Sum(nil))

	values.Add("apiseal", apiseal)

	endpoint := "offer/all"
	url := "https://paxful.com/api/" + endpoint

	req, err := http.NewRequest(
		http.MethodPost, url,
		strings.NewReader(values.Encode()),
	)
	if err != nil {
		panic(err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "text/plain")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)

	if gjson.GetBytes(body, "status").String() == "success" {
		price = gjson.GetBytes(body, "data.offers.0.fiat_price_per_btc").Float()
	}

	return price
}

func getGiftCardUSD() float64 {
	price := 0.0

	nonce := fmt.Sprintf("%d", time.Now().Unix())

	values := url.Values{}
	values.Add("apikey", paxfulAPIKey)
	values.Add("nonce", nonce)

	values.Add("offer_type", "buy")
	values.Add("payment_method", "amazon-gift-card")
	values.Add("currency_code", "USD")

	payload := values.Encode()

	mac := hmac.New(sha256.New, []byte(paxfulSecretAPIKey))
	mac.Write([]byte(payload))
	apiseal := hex.EncodeToString(mac.Sum(nil))

	values.Add("apiseal", apiseal)

	endpoint := "offer/all"
	url := "https://paxful.com/api/" + endpoint

	req, err := http.NewRequest(
		http.MethodPost, url,
		strings.NewReader(values.Encode()),
	)
	if err != nil {
		panic(err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "text/plain")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)

	if gjson.GetBytes(body, "status").String() == "success" {
		sum := 0.0
		prices := gjson.GetBytes(body, "data.offers.#.fiat_price_per_btc").Array()
		for _, v := range prices[2:12] {
			sum += v.Float()
		}
		price = sum / 10
	}

	return price
}

func calculatePaxfulrate() {
	db, err := sql.Open("postgres", "host=localhost port=5432 user=postgres password=1314 dbname=paxfulrate sslmode=disable")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	var rateMinutes []float64
	var rateHours []float64

	for {
		now := time.Now()
		if now.Second() == 0 {
			if now.Minute() == 0 {
				if now.Hour() == 0 {
					if len(rateHours) != 0 {
						sum := 0.0
						for _, v := range rateHours {
							sum += v
						}
						rateDay := sum / float64(len(rateHours))

						stmt, err := db.Prepare(`insert into "rateDay"("rateDay", rate) values ($1, $2)`)
						if err != nil {
							panic(err)
						}
						_, err = stmt.Exec(now.Format("2006-01-02"), fmt.Sprintf("%.2f", rateDay))
						if err != nil {
							panic(err)
						}

						rateHours = rateHours[0:0]
					}
				}
				if len(rateMinutes) != 0 {
					sum := 0.0
					for _, v := range rateMinutes {
						sum += v
					}
					rateHour := sum / float64(len(rateMinutes))

					stmt, err := db.Prepare(`insert into "rateHour"("rateHour", rate) values ($1, $2)`)
					if err != nil {
						panic(err)
					}
					_, err = stmt.Exec(now.Format("2006-01-02 15:00:00"), fmt.Sprintf("%.2f", rateHour))
					if err != nil {
						panic(err)
					}

					rateHours = append(rateHours, rateHour)

					rateMinutes = rateMinutes[0:0]
				}
			}
			bitcoinPrice := getBitcoinCNY()
			giftCardPrice := getGiftCardUSD()
			rate := bitcoinPrice / giftCardPrice
			stmt, err := db.Prepare(`insert into "rateMinute"("rateMinute", rate, "giftcardPrice", "bitcoinPrice") values ($1, $2, $3, $4)`)
			if err != nil {
				panic(err)
			}
			_, err = stmt.Exec(now.Format("2006-01-02 15:04:00"), fmt.Sprintf("%.2f", rate), fmt.Sprintf("%.2f", giftCardPrice), fmt.Sprintf("%.2f", bitcoinPrice))
			if err != nil {
				panic(err)
			}

			rateMinutes = append(rateMinutes, rate)
		}
		time.Sleep(time.Second * 1)
	}
}

func main() {
	go calculatePaxfulrate()

	for {
		time.Sleep(time.Hour * 1)
	}
}
