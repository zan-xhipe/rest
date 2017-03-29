package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"

	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	verbose = kingpin.Flag("verbose", "Verbose mode").Short('v').Bool()

	get     = kingpin.Command("get", "Perform a GET request")
	getPath = get.Arg("url", "url to perform request on").Required().String()

	post     = kingpin.Command("post", "Perform a POST request")
	postPath = post.Arg("url", "url to perform request on").Required().String()
	postData = post.Arg("data", "data to send in the POST request").Required().String()

	put     = kingpin.Command("put", "Perform a PUT request")
	putPath = put.Arg("url", "url to perform request on").Required().String()
	putData = put.Arg("data", "data to send in the PUT request").Required().String()

	delete     = kingpin.Command("delete", "Performa DELETE request")
	deletePath = delete.Arg("url", "url to perform request on").Required().String()
)

func main() {
	client := &http.Client{}

	reqType := strings.ToUpper(kingpin.Parse())
	req, err := http.NewRequest(reqType, path(), data())
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	fmt.Print(string(body))
}

func path() string {
	for _, p := range []*string{getPath, postPath, putPath, deletePath} {
		if p != nil {
			u, err := url.Parse(*p)
			if err != nil {
				panic(err)
			}

			if u.Scheme == "" {
				u.Scheme = "http"
			}

			if u.Host == "" {
				u.Host = "localhost:80"
			}

			return u.String()
		}
	}

	return ""
}

func data() io.Reader {
	for _, d := range []*string{postData, putData} {
		if d != nil {
			return strings.NewReader(*d)
		}
	}

	return nil
}
