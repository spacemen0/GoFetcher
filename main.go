package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func sendRequest(url string) (*http.Response, error) {
	// Create a new GET request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// Perform the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error performing request: %w", err)
	}

	return resp, nil
}

func decodeJSON(resp *http.Response) (any, error) {
	// Decode the JSON response
	var data any
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("error decoding JSON: %w", err)
	}

	return data, nil
}

func filterMasterURLs(data any) []any {
	var masterUrls []any

	if releases, ok := data.(map[string]any)["releases"]; ok {
		for _, release := range releases.([]any) {
			if releaseMap, ok := release.(map[string]any); ok {
				if releaseMap["type"] == "master" {
					masterUrls = append(masterUrls, releaseMap["resource_url"])
				}
			}
		}
	}

	return masterUrls
}

func processMasterURLs(masterUrls []any) []any {
	var releases []any

	for _, masterUrl := range masterUrls {
		var url, _ = interfaceToString(masterUrl)
		resp, err := sendRequest(url)
		if err != nil {
			fmt.Println("Error:", err)
			return nil
		}
		data, err := decodeJSON(resp)
		if err != nil {
			fmt.Println("Error:", err)
			return nil
		}
		releases = append(releases, data)

	}

	return releases
}

func filterReleases(releases []any) []map[string]any {
	filteredReleases := make([]map[string]any, 0)

	for _, release := range releases {
		if releaseMap, ok := release.(map[string]any); ok {
			// Create a map to store only the desired fields
			filteredRelease := make(map[string]any)

			// Add the desired fields to the filtered release map
			filteredRelease["title"] = releaseMap["title"]
			filteredRelease["year"] = releaseMap["year"]
			filteredRelease["genres"] = releaseMap["genres"]
			filteredRelease["trackList"] = releaseMap["tracklist"]
			// Add other fields you want to include

			// Append the filtered release to the result
			filteredReleases = append(filteredReleases, filteredRelease)
		}
	}

	return filteredReleases
}

func interfaceToString(value any) (string, bool) {
	// Check if the value is actually a string
	str, ok := value.(string)
	if !ok {
		return "", false // Return false indicating the conversion failed
	}
	return str, true // Return the string and true indicating successful conversion
}

func beautifyJson(value any) {
	prettyJSON, err := json.MarshalIndent(value, "", "    ")
	if err != nil {
		fmt.Println("Error formatting JSON:", err)
		return
	}

	fmt.Println(string(prettyJSON))
}

func main() {
	// URL for the GET request
	url := "https://api.discogs.com/artists/3840/releases"

	// Send the request
	resp, err := sendRequest(url)
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
	data, err := decodeJSON(resp)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Filter master URLs from the data
	masterUrls := filterMasterURLs(data)

	// Process master URLs
	beautifyJson(filterReleases(processMasterURLs(masterUrls)))
}
