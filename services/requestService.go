package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

type Record struct {
	Url   any
	Title any
}

func SendRequest(url string) (*http.Response, error) {
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

func DecodeJSON(resp *http.Response) (any, error) {
	// Decode the JSON response
	var data any
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("error decoding JSON: %w", err)
	}

	return data, nil
}

func FilterMasterURLs(data any) []Record {
	var masterUrls []Record

	if releases, ok := data.(map[string]any)["releases"]; ok {
		for _, release := range releases.([]any) {
			if releaseMap, ok := release.(map[string]any); ok {
				if releaseMap["type"] == "master" {
					masterUrls = append(masterUrls,
						Record{
							Url:   releaseMap["resource_url"],
							Title: releaseMap["title"],
						})
				}
			}
		}
	}

	return masterUrls
}

func ProcessMasterURLs(masterUrls []any) []any {
	var releases []any

	for _, masterUrl := range masterUrls {
		var url, _ = interfaceToString(masterUrl)
		resp, err := SendRequest(url)
		if err != nil {
			fmt.Println("Error:", err)
			return nil
		}
		data, err := DecodeJSON(resp)
		if err != nil {
			fmt.Println("Error:", err)
			return nil
		}
		releases = append(releases, data)

	}

	return releases
}

func FilterReleases(releases []any) []map[string]any {
	filteredReleases := make([]map[string]any, 0)

	for _, release := range releases {
		if releaseMap, ok := release.(map[string]any); ok {
			// Create a map to store only the desired fields
			filteredRelease := make(map[string]any)

			// Add the desired fields to the filtered release map
			filteredRelease["title"] = releaseMap["title"]
			filteredRelease["year"] = releaseMap["year"]
			filteredRelease["genre"] = releaseMap["genres"].([]any)[0]
			var trackInfo string
			tracks := releaseMap["tracklist"].([]any)
			for i, track := range tracks {
				trackString, _ := interfaceToString(track.(map[string]any)["title"])
				trackInfo += trackString
				if i < len(tracks)-1 {
					trackInfo += "\n"
				}
			}
			filteredRelease["additional"] = trackInfo
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

func BeautifyJson(value any) {
	prettyJSON, err := json.MarshalIndent(value, "", "    ")
	if err != nil {
		fmt.Println("Error formatting JSON:", err)
		return
	}
	err = os.WriteFile("data.json", prettyJSON, 1)
	if err != nil {
		fmt.Println(err)
	}
}
