package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/tidwall/gjson"
	"io/ioutil"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"
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
	bitcoinPrice := getBitcoinCNY()
	giftCardPrice := getGiftCardUSD()
	rate := bitcoinPrice / giftCardPrice
	rate = math.Trunc(rate*1e2+0.5) * 1e-2
	fmt.Println(bitcoinPrice)
	fmt.Println(giftCardPrice)
	fmt.Println(rate)
}

func main() {
	calculatePaxfulrate()
}
