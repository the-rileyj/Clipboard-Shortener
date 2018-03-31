package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"net/http"

	"github.com/atotto/clipboard"
)

type info struct {
	Key string `json:"key"`
}

type errorResponse struct {
	ERROR string `json:"message"`
}

type dataResponse struct {
	ID   string `json:"id"`
	LONG string `json:"longUrl"`
}

const (
	//KILL String for killing this program
	KILL = "kill cs"
)

func getURL(resp *http.Response) (dataResponse, error) {
	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)
	if resp.StatusCode == 200 {
		var dataR dataResponse
		err := decoder.Decode(&dataR)

		if err != nil {
			return dataResponse{}, err
		}
		return dataR, nil
	}
	var dataE errorResponse
	err := decoder.Decode(&dataE)

	if err != nil {
		return dataResponse{}, err
	}
	return dataResponse{}, fmt.Errorf(dataE.ERROR)
}

func main() {
	var ok bool
	var dat info
	var err error
	var cb, last, nURL string

	cache := make(map[string]string)
	longMatch := regexp.MustCompile(`^(https?:\/\/)?[\d\w^\.]+\.[\w^\/]+(\/[^\/]*)*$`)
	shortMatch := regexp.MustCompile(`^(https?:\/\/)?goo\.gl\/([^\/]+\/?)+$`)

	client := http.Client{Timeout: time.Second * 10}
	fi, err := os.Open("info.json")
	if err != nil {
		log.Fatal(err)
	}
	b, err := ioutil.ReadAll(fi)
	fi.Close()

	if err != nil {
		log.Fatal(err)
	}
	json.Unmarshal(b, &dat)

	apiSite := fmt.Sprintf("https://www.googleapis.com/urlshortener/v1/url?key=%s", dat.Key)

	getElongatedURL := func(URL string) (string, error) {
		s := apiSite + "&shortUrl=" + URL
		resp, err := client.Get(s)
		if err == nil {
			dataR, err := getURL(resp)
			if err != nil {
				return "", err
			}
			return dataR.LONG, nil
		}
		return "", err
	}

	getShortenedURL := func(URL string) (string, error) {
		JSON := fmt.Sprintf(`{"longUrl": "%s"}`, URL)
		s := strings.NewReader(JSON)
		resp, err := client.Post(apiSite, "application/json", s)
		if err == nil {
			dataR, err := getURL(resp)
			if err != nil {
				return "", err
			}
			return dataR.ID, nil
		}
		return "", err
	}

	for cb != KILL {
		cb, err = clipboard.ReadAll()

		if err != nil {
			fmt.Println(err)
		} else {
			if nURL, ok = cache[cb]; last != cb && ok {
				fmt.Printf("Found Transformed URL %s from %s in Cache\n", nURL, cb)
				clipboard.WriteAll(nURL)
				last = nURL
			} else if last != cb && shortMatch.MatchString(cb) {
				if nURL, err = getElongatedURL(cb); err != nil {
					fmt.Println(err)
				} else {
					fmt.Printf("New Elongated URL %s from %s\n", nURL, cb)
					cache[nURL] = cb
					cache[cb] = nURL
					clipboard.WriteAll(nURL)
					last = nURL
				}
			} else if last != cb && longMatch.MatchString(cb) {
				if nURL, err = getShortenedURL(cb); err != nil {
					fmt.Println(err)
				} else {
					fmt.Printf("New Shortened URL %s from %s\n", nURL, cb)
					cache[nURL] = cb
					cache[cb] = nURL
					clipboard.WriteAll(nURL)
					last = nURL
				}
			} else {
				last = cb
			}
		}
		time.Sleep(time.Second * 5)
	}
	fmt.Println("Ded")
}
