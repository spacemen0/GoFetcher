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
	"regexp"
	"strconv"
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
	Image       string
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
							title: releaseMap["title"].(string),
							image: releaseMap["cover_image"].(string),
						})
				}
			}
		}
	}

	return masterUrls
}

func DownloadImage(url, filename string) (string, error) {
	// Send an HTTP GET request to the Image URL
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	// Create a new file to save the downloaded Image
	file, err := os.Create("images/" + filename) // Save image in images directory
	if err != nil {
		return "", err
	}
	// Copy the Image data from the HTTP response body to the file
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		file.Close()
		return "", err
	}
	// Close the file
	err = file.Close()
	if err != nil {
		return "", err
	}
	return file.Name(), nil
}
func sanitizeFilename(filename string) string {
	re := regexp.MustCompile(`[\\/:*?"<>|]`)
	return re.ReplaceAllString(filename, "_")
}
func ProcessMasterURLs(masterUrls []Record, id uint) ([]any, []string, uint) {
	var releases []any
	var imagePaths []string

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
		imagePath, err := DownloadImage(record.image, sanitizeFilename(record.title+".jpeg"))
		if err != nil {
			fmt.Println("Error:", err)
			return nil, nil, id
		}
		imagePaths = append(imagePaths, imagePath)
	}

	return releases, imagePaths, id
}

func FilterReleases(releases []any, images []string, authorId uint) []Request {
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
			filteredRelease.Title = releaseMap["title"].(string)
			if year, ok := releaseMap["year"].(float64); ok {
				filteredRelease.ReleaseDate = strconv.Itoa(int(year)) + "-01-01"
			} else {
				filteredRelease.ReleaseDate = "1900-01-01"
			}
			filteredRelease.Genre = releaseMap["genres"].([]any)[0].(string)
			var trackInfo string
			tracks := releaseMap["tracklist"].([]any)
			for i, track := range tracks {
				trackString, _ := interfaceToString(track.(map[string]any)["title"])
				trackInfo += trackString
				if i < len(tracks)-1 {
					trackInfo += "\n"
				}
			}
			filteredRelease.Additional = trackInfo
			if notes, ok := releaseMap["notes"].(string); ok {
				filteredRelease.Description = notes
			} else {
				filteredRelease.Description = "No description available."
			}

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

func AddMusic(reqData Request, token string) error {

	url := "http://localhost:8080/medias"
	method := "POST"

	payload := &bytes.Buffer{}
	writer := multipart.NewWriter(payload)

	// Add Image file to the request if available
	if reqData.Image != "" {
		// Open the image file
		file, err := os.Open(reqData.Image)
		if err != nil {
			println(err)
			return err
		}
		defer file.Close()

		println(reqData.Image)
		// Create a new form file part
		part, err := writer.CreateFormFile("image", filepath.Base(reqData.Image))
		if err != nil {
			println(err)
			return err
		}

		// Copy the image data to the form file part
		_, err = io.Copy(part, file)
		if err != nil {
			println(err)
			return err
		}
	}

	// Add other form fields
	_ = writer.WriteField("title", reqData.Title)
	_ = writer.WriteField("genre", reqData.Genre)
	_ = writer.WriteField("additional", reqData.Additional)
	_ = writer.WriteField("description", reqData.Description)
	_ = writer.WriteField("releaseDate", reqData.ReleaseDate)
	_ = writer.WriteField("imageUrl", reqData.ImageUrl)
	_ = writer.WriteField("average", "0")
	_ = writer.WriteField("wants", "0")
	_ = writer.WriteField("ratings", "0")
	_ = writer.WriteField("doings", "0")
	_ = writer.WriteField("type", "Music")
	_ = writer.WriteField("authorId", fmt.Sprintf("%d", reqData.AuthorId))

	err := writer.Close()
	if err != nil {
		return err
	}

	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", "Bearer "+token)
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

	//response, err := DecodeJSON(res)
	//if err != nil {
	//	return nil
	//}
	//
	//// Assuming the JSON response contains a field named "title" that holds the string value
	//titleField, ok := response.(map[string]any)["title"].(string)
	//if !ok {
	//	return nil
	//}
	//
	//if strings.Contains(titleField, reqData.Title) {
	//	fmt.Println("Success!")
	//}

	if err != nil {
		return err
	}
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
