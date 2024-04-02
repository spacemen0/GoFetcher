package main

import (
	"GoFetcher/services"
	"fmt"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/textarea"
	"io"
	"net/http"
)

const (
	barPadding  = 2
	barMaxWidth = 80
)

type model struct {
	textarea textarea.Model
	bar      progress.Model
	releases []string
	artist   string
}

func main() {
	// URL for the GET request
	url := "https://api.discogs.com/artists/3840/releases"

	// Send the request
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

	// Check if response status code is OK
	if resp.StatusCode != http.StatusOK {
		fmt.Println("Error: Unexpected status code:", resp.StatusCode)
		return
	}

	// Decode the JSON response
	data, err := services.DecodeJSON(resp)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Filter master URLs from the data
	masterUrls := services.FilterMasterURLs(data)

	// Process master URLs
	services.BeautifyJson(services.FilterReleases(services.ProcessMasterURLs(masterUrls)))
}
