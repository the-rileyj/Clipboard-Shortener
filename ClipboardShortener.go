package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
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

func getDataStruct(resp *http.Response) (dataResponse, error) {
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
	var delay int
	var err error
	var cb, file, key, last, nURL string

	cache := make(map[string]string)
	client := http.Client{Timeout: time.Second * 10}

	flag.IntVar(&delay, "delay", 1500, "The delay between each time the clipboard is polled")
	flag.StringVar(&file, "file", "", "JSON file to read in API key data from")
	flag.StringVar(&key, "key", "", "Key for the Google URL Shortener API")
	flag.Parse()

	longMatch := regexp.MustCompile(`^(https?:\/\/)?[\d\w^\.]+\.[\w^\/]+(\/[^\/]*)*$`)
	shortMatch := regexp.MustCompile(`^(https?:\/\/)?goo\.gl\/([^\/]+\/?)+$`)

	if file == "" {
		matches, err := filepath.Glob("*_info.json")

		if err != nil {
			log.Fatal(err)
		}
		file = matches[0]
	}

	if key == "" {
		var dat info

		fi, err := os.Open(file)
		if err != nil {
			log.Fatal(err)
		}
		b, err := ioutil.ReadAll(fi)
		fi.Close()

		if err != nil {
			log.Fatal(err)
		}
		json.Unmarshal(b, &dat)

		key = dat.Key
	}

	apiSite := fmt.Sprintf("https://www.googleapis.com/urlshortener/v1/url?key=%s", key)

	getElongatedURL := func(URL string) (string, error) {
		s := apiSite + "&shortUrl=" + URL
		resp, err := client.Get(s)
		if err == nil {
			dataR, err := getDataStruct(resp)
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
			dataR, err := getDataStruct(resp)
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
		time.Sleep(time.Millisecond * time.Duration(delay))
	}
	fmt.Println("Ded")
}
