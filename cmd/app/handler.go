package main

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
	"strings"
	"vip-integration/config"
)

type Meta struct {
	Application string `json:"application"`
	Version     string `json:"version"`
	Author      string `json:"author"`
}

type Item struct {
	ProductId string `json:"ProductId"`
	Balance   int    `json:"Balance"`
}

type Data struct {
	Items        []Item `json:"items"`
	ContractorId int    `json:"ContractorId"`
}

func (app *application) home(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	log.Println(r.Header)

	_ = app.writeJSON(w, http.StatusOK, JSONResponse{
		Meta: Meta{
			Application: config.ApplicationName,
			Version:     config.ApplicationVersion,
			Author:      config.ApplicationAuthor,
		},
	})
}

func (app *application) DraftSubmissionHandler(w http.ResponseWriter, r *http.Request) {
	url := fmt.Sprintf("%s/invoice/draft-submission", config.VipBaseURL)
	method := "POST"

	// Fetch the accessToken from Redis
	accessToken, err := rdb.Get(ctx, "accessToken").Result()
	if err != nil {
		http.Error(w, "Failed to fetch accessToken from Redis", http.StatusInternalServerError)
		return
	}

	//log.Println(accessToken)

	// Read and parse the incoming JSON payload
	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Convert the payload to JSON string
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		http.Error(w, "Failed to marshal payload", http.StatusInternalServerError)
		return
	}

	log.Println("request-body ====")
	log.Println(string(payloadBytes))
	log.Println("request-body ====")

	client := &http.Client{}
	req, err := http.NewRequest(method, url, strings.NewReader(string(payloadBytes)))
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	req.Header.Add("token", accessToken)
	req.Header.Add("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		http.Error(w, "Request failed", http.StatusInternalServerError)
		return
	}
	//log.Println("res ====")
	//log.Println(res)
	//log.Println("res ====")
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			http.Error(w, "Failed to close response body", http.StatusInternalServerError)
			return
		}
	}(res.Body)

	body, err := io.ReadAll(res.Body)
	if err != nil {
		http.Error(w, "Failed to read response", http.StatusInternalServerError)
		return
	}

	log.Println("response-body ====")
	log.Println(string(body))
	log.Println("response-body ====")

	// Write the response back to the client
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(res.StatusCode)
	_, err = w.Write(body)
	if err != nil {
		return
	}
}

func (app *application) UploadInvoiceHandler(w http.ResponseWriter, r *http.Request) {
	url := fmt.Sprintf("%s/invoice/upload", config.VipBaseURL)

	// Fetch the accessToken from Redis
	accessToken, err := rdb.Get(ctx, "accessToken").Result()
	if err != nil {
		http.Error(w, "Failed to fetch accessToken from Redis", http.StatusInternalServerError)
		return
	}

	// Create a new multipart writer
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Get the file from the form and write it to the multipart writer
	filePath := "/home/plnpusat/dummy.pdf"
	file, err := os.Open(filePath)
	if err != nil {
		http.Error(w, "Failed to open file", http.StatusInternalServerError)
		return
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Println("Failed to close file")
		}
	}(file)

	part, err := writer.CreateFormFile("files", filepath.Base(file.Name()))
	if err != nil {
		http.Error(w, "Failed to create form file", http.StatusInternalServerError)
		return
	}

	_, err = io.Copy(part, file)
	if err != nil {
		http.Error(w, "Failed to copy file data", http.StatusInternalServerError)
		return
	}

	// Close the multipart writer to complete the body
	err = writer.Close()
	if err != nil {
		http.Error(w, "Failed to close writer", http.StatusInternalServerError)
		return
	}

	// Build the request
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	req.Header.Add("token", accessToken)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Add query parameters
	query := req.URL.Query()
	query.Add("bap_id", "9")
	query.Add("document_type", "invoice")
	query.Add("vendor_clustered_id", "96176")
	req.URL.RawQuery = query.Encode()

	// Execute the request
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		http.Error(w, "Request failed", http.StatusInternalServerError)
		return
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			http.Error(w, "Failed to close response body", http.StatusInternalServerError)
			return
		}
	}(res.Body)

	// Read and write the response
	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		http.Error(w, "Failed to read response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(res.StatusCode)
	w.Write(responseBody)
}

func (app *application) GetInvoiceHandler(w http.ResponseWriter, r *http.Request) {
	// Fetch the accessToken from Redis
	accessToken, err := rdb.Get(ctx, "accessToken").Result()
	if err != nil {
		http.Error(w, "Failed to fetch accessToken from Redis", http.StatusInternalServerError)
		return
	}

	// Prepare the request
	url := fmt.Sprintf("%s/invoice", config.VipBaseURL)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Printf("Failed to create request: %v", err)
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	// Set query parameters
	q := req.URL.Query()
	q.Add("bap_id", "9")
	q.Add("vendor_clustered_id", "96176")
	req.URL.RawQuery = q.Encode()

	// Set headers
	req.Header.Set("token", accessToken)

	// Make the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to make request: %v", err)
		http.Error(w, "Failed to make request", http.StatusInternalServerError)
		return
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("Failed to close response body: %v", err)
			http.Error(w, "Failed to close response body", http.StatusInternalServerError)
			return
		}
	}(resp.Body)

	// Copy the response to the original writer
	w.WriteHeader(resp.StatusCode)
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		log.Printf("Failed to copy response body: %v", err)
		http.Error(w, "Failed to copy response body", http.StatusInternalServerError)
		return
	}
}
