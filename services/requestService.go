package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

type Record struct {
	url   string
	title string
	image string
}

type Request struct {
	Title       string
	Genre       string
	Additional  string
	Description string
	ReleaseDate string
	ImageUrl    string
	AuthorId    uint
	Image       *os.File
}

func (r Record) FilterValue() string {
	return r.title
}

func (r Record) Title() string {
	return r.title
}

func (r Record) Description() string {
	return r.url
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

	if releases, ok := data.(map[string]any)["results"]; ok {
		for _, release := range releases.([]any) {
			if releaseMap, ok := release.(map[string]any); ok {
				if releaseMap["type"] == "master" {
					masterUrls = append(masterUrls,
						Record{
							url:   releaseMap["resource_url"].(string),
							title: releaseMap["Title"].(string),
							image: releaseMap["cover_image"].(string),
						})
				}
			}
		}
	}

	return masterUrls
}

func DownloadImage(url, filename string) *os.File {
	// Send an HTTP GET request to the Image URL
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal()
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Fatal()
		}
	}(resp.Body)

	// Create a new file to save the downloaded Image
	file, err := os.Create(filename)
	if err != nil {
		log.Fatal()
	}

	// Copy the Image data from the HTTP response body to the file
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		err := file.Close()
		if err != nil {
			log.Fatal()
		}
		log.Fatal()
	}

	return file
}

func ProcessMasterURLs(masterUrls []Record, id uint) ([]any, []*os.File, uint) {
	var releases []any
	var images []*os.File

	for _, record := range masterUrls {
		resp, err := SendRequest(record.url)
		if err != nil {
			fmt.Println("Error:", err)
			return nil, nil, id
		}
		data, err := DecodeJSON(resp)
		if err != nil {
			fmt.Println("Error:", err)
			return nil, nil, id
		}
		releases = append(releases, data)
		images = append(images, DownloadImage(record.image, record.title))
	}

	return releases, images, id
}

func FilterReleases(releases []any, images []*os.File, authorId uint) []Request {
	var filteredReleases []Request

	for i, release := range releases {
		if releaseMap, ok := release.(map[string]any); ok {
			// Create a map to store only the desired fields
			filteredRelease := Request{
				Title:       "",
				Genre:       "",
				Additional:  "",
				Description: "",
				ReleaseDate: "",
				ImageUrl:    "placeHolder",
				AuthorId:    authorId,
				Image:       images[i],
			}

			// Add the desired fields to the filtered release map
			filteredRelease.Title = releaseMap["Title"].(string)
			filteredRelease.ReleaseDate = releaseMap["year"].(string) + "-01-01"
			filteredRelease.Genre = releaseMap["genres"].([]any)[0].(string)
			var trackInfo string
			tracks := releaseMap["tracklist"].([]any)
			for i, track := range tracks {
				trackString, _ := interfaceToString(track.(map[string]any)["Title"])
				trackInfo += trackString
				if i < len(tracks)-1 {
					trackInfo += "\n"
				}
			}
			filteredRelease.Additional = trackInfo
			filteredRelease.Description = releaseMap["notes"].(string)

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

func AddMusic(reqData Request) error {
	url := "http://localhost:8080/medias"
	method := "POST"

	payload := &bytes.Buffer{}
	writer := multipart.NewWriter(payload)

	// Add Image file to the request if available
	if reqData.Image != nil {
		defer reqData.Image.Close()
		part, err := writer.CreateFormFile("Image", filepath.Base(reqData.Image.Name()))
		if err != nil {
			return err
		}
		_, err = io.Copy(part, reqData.Image)
		if err != nil {
			return err
		}
	}

	// Add other form fields
	_ = writer.WriteField("Title", reqData.Title)
	_ = writer.WriteField("Genre", reqData.Genre)
	_ = writer.WriteField("Additional", reqData.Additional)
	_ = writer.WriteField("Description", reqData.Description)
	_ = writer.WriteField("ReleaseDate", reqData.ReleaseDate)
	_ = writer.WriteField("ImageUrl", reqData.ImageUrl)
	_ = writer.WriteField("AuthorId", fmt.Sprintf("%d", reqData.AuthorId))

	err := writer.Close()
	if err != nil {
		return err
	}

	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", "Bearer <your_token_here>")
	req.Header.Set("Content-Type", writer.FormDataContentType())

	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Fatal()
		}
	}(res.Body)

	response, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	fmt.Println(json.MarshalIndent(response, "", ""))
	return nil
}

func WriteToFile(value any) {
	prettyJSON, err := json.MarshalIndent(value, "", "    ")
	if err != nil {
		fmt.Println("Error formatting JSON:", err)
		return
	}
	err = os.WriteFile("data.json", prettyJSON, 0644)
	if err != nil {
		fmt.Println(err)
	}
}
