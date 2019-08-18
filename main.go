package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"io/ioutil"
	"log"
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
		return 0
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "text/plain")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0
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
		return 0
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "text/plain")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0
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
	checkErr(err)
	defer db.Close()

	var rateMinutes []float64
	var rateHours []float64

	for {
		now := time.Now()
		if now.Second() == 0 {
			if now.Minute() == 0 {
				if now.Hour() == 0 {
					rateDay := 0.0
					if len(rateHours) != 0 {
						sum := 0.0
						for _, v := range rateHours {
							sum += v
						}
						rateDay = sum / float64(len(rateHours))
					}
					stmt, err := db.Prepare(`insert into "rateDay"("rateDay", rate) values ($1, $2)`)
					checkErr(err)
					_, err = stmt.Exec(now.Format("2006-01-02"), fmt.Sprintf("%.2f", rateDay))
					checkErr(err)

					rateHours = rateHours[0:0]
				}
				rateHour := 0.0
				if len(rateMinutes) != 0 {
					sum := 0.0
					for _, v := range rateMinutes {
						sum += v
					}
					rateHour = sum / float64(len(rateMinutes))
				}
				stmt, err := db.Prepare(`insert into "rateHour"("rateHour", rate) values ($1, $2)`)
				checkErr(err)
				_, err = stmt.Exec(now.Format("2006-01-02 15:00:00"), fmt.Sprintf("%.2f", rateHour))
				checkErr(err)

				if rateHour != 0 {
					rateHours = append(rateHours, rateHour)
				}

				rateMinutes = rateMinutes[0:0]
			}
			bitcoinPrice := getBitcoinCNY()
			giftCardPrice := getGiftCardUSD()
			rate := 0.0
			if bitcoinPrice != 0 && giftCardPrice != 0 {
				rate = bitcoinPrice / giftCardPrice
			}
			stmt, err := db.Prepare(`insert into "rateMinute"("rateMinute", rate, "giftcardPrice", "bitcoinPrice") values ($1, $2, $3, $4)`)
			checkErr(err)
			_, err = stmt.Exec(now.Format("2006-01-02 15:04:00"), fmt.Sprintf("%.2f", rate), fmt.Sprintf("%.2f", giftCardPrice), fmt.Sprintf("%.2f", bitcoinPrice))
			checkErr(err)

			if rate != 0 {
				rateMinutes = append(rateMinutes, rate)
			}
		}
		time.Sleep(time.Second * 1)
	}
}

type RateMinute struct {
	RateMinute string  `json:"rateMinute"`
	Rate       float64 `json:"rate"`
}

type RateHour struct {
	RateHour string  `json:"rateHour"`
	Rate     float64 `json:"rate"`
}

type RateDay struct {
	RateDay string  `json:"rateDay"`
	Rate    float64 `json:"rate"`
}

func rateMinutes(c *gin.Context) {
	db, err := sql.Open("postgres", "host=localhost port=5432 user=postgres password=1314 dbname=paxfulrate sslmode=disable")
	checkErr(err)
	defer db.Close()

	rows, err := db.Query(`select * from (select "rateMinute", rate from "rateMinute" order by "rateMinute" desc limit 1440) as result order by "rateMinute"`)
	checkErr(err)
	defer rows.Close()

	rateMinutes := make([]RateMinute, 0)
	for rows.Next() {
		var rateMinute RateMinute
		var timeStr string
		rows.Scan(&timeStr, &rateMinute.Rate)
		tt, _ := time.Parse("2006-01-02T15:04:05Z", timeStr)
		rateMinute.RateMinute = tt.Format("2006/01/02 15:04")
		rateMinutes = append(rateMinutes, rateMinute)
	}
	if err = rows.Err(); err != nil {
		log.Fatalln(err)
	}

	c.JSON(http.StatusOK, gin.H{
		"rateMinutes": rateMinutes,
	})
}

func rateHours(c *gin.Context) {
	db, err := sql.Open("postgres", "host=localhost port=5432 user=postgres password=1314 dbname=paxfulrate sslmode=disable")
	checkErr(err)
	defer db.Close()

	rows, err := db.Query(`select * from (select * from "rateHour" order by "rateHour" desc limit 1440) as result order by "rateHour"`)
	checkErr(err)
	defer rows.Close()

	rateHours := make([]RateHour, 0)
	for rows.Next() {
		var rateHour RateHour
		var timeStr string
		rows.Scan(&timeStr, &rateHour.Rate)
		tt, _ := time.Parse("2006-01-02T15:04:05Z", timeStr)
		rateHour.RateHour = tt.Format("2006/01/02 15:04")
		rateHours = append(rateHours, rateHour)
	}
	if err = rows.Err(); err != nil {
		log.Fatalln(err)
	}

	c.JSON(http.StatusOK, gin.H{
		"rateHours": rateHours,
	})
}

func rateDays(c *gin.Context) {
	db, err := sql.Open("postgres", "host=localhost port=5432 user=postgres password=1314 dbname=paxfulrate sslmode=disable")
	checkErr(err)
	defer db.Close()

	rows, err := db.Query(`select * from (select * from "rateDay" order by "rateDay" desc limit 180) as result order by "rateDay"`)
	checkErr(err)
	defer rows.Close()

	rateDays := make([]RateDay, 0)
	for rows.Next() {
		var rateDay RateDay
		var timeStr string
		rows.Scan(&timeStr, &rateDay.Rate)
		tt, _ := time.Parse("2006-01-02T15:04:05Z", timeStr)
		rateDay.RateDay = tt.Format("2006/01/02")
		rateDays = append(rateDays, rateDay)
	}
	if err = rows.Err(); err != nil {
		log.Fatalln(err)
	}

	c.JSON(http.StatusOK, gin.H{
		"rateDays": rateDays,
	})
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	go calculatePaxfulrate()

	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	router.GET("/rateMinutes", rateMinutes)
	router.GET("/rateHours", rateHours)
	router.GET("/rateDays", rateDays)

	router.Run(":8000")
}
