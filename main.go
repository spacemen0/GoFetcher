package main

import (
	"GoFetcher/services"
	"fmt"
	"io"
	"net/http"
)

func main() {

	url := "https://api.discogs.com/artists/3840/releases"

	resp, err := services.SendRequest(url)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Println("Error closing response body:", err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		fmt.Println("Error: Unexpected status code:", resp.StatusCode)
		return
	}

	data, err := services.DecodeJSON(resp)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	records := services.FilterMasterURLs(data)
	var masterUrls []any
	for _, record := range records {
		masterUrls = append(masterUrls, record.Url)
	}
	fmt.Println(records)

	services.BeautifyJson(services.FilterReleases(services.ProcessMasterURLs(masterUrls)))
}
