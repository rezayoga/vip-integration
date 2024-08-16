package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"vip-integration/config"
)

type JSONResponse struct {
	OrderID string      `json:"order_id,omitempty"`
	Error   bool        `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Meta    interface{} `json:"meta,omitempty"`
	Page    interface{} `json:"page,omitempty"`
}

func (app *application) writeJSON(w http.ResponseWriter, status int, data interface{}, headers ...http.Header) error {
	out, err := json.MarshalIndent(data, "", "")
	if err != nil {
		return err
	}

	if len(headers) > 0 {
		for key, value := range headers[0] {
			w.Header()[key] = value
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_, err = w.Write(out)
	if err != nil {
		return err
	}

	return nil
}

func (app *application) readJSON(w http.ResponseWriter, r *http.Request, data interface{}) error {
	maxBytes := 1048576 // 1MB

	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))

	dec := json.NewDecoder(r.Body)

	dec.DisallowUnknownFields()

	err := dec.Decode(data)

	if err != nil {
		return err
	}

	err = dec.Decode(&struct{}{})

	if err != io.EOF {
		return errors.New("body must only have a single JSON value")
	}

	return nil

}

func (app *application) errorJSON(w http.ResponseWriter, err error, status ...int) error {
	statusCode := http.StatusBadRequest

	if len(status) > 0 {
		statusCode = status[0]
	}

	var payload JSONResponse
	payload.Error = true
	payload.Message = err.Error()

	return app.writeJSON(w, statusCode, payload)
}

func (app *application) initPaginationResponse(r *http.Request) (int, int, string, string) {
	// get the page and per_page query params
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page < 1 {
		page = 1
	}

	perPage, err := strconv.Atoi(r.URL.Query().Get("per_page"))
	if err != nil || perPage < 1 {
		perPage = 10
	}

	sort := r.URL.Query().Get("sort")
	if sort == "" {
		sort = "created_at"
	}

	order := r.URL.Query().Get("order")
	if order == "" {
		order = "DESC"
	}

	if order == "desc" {
		order = "DESC"
	} else {
		order = "ASC"
	}

	return page, perPage, sort, order
}

func (app *application) GetIpAddrAndUserAgent(r *http.Request) (string, string) {
	var ipAddr string
	var userAgent string

	ipAddr = r.RemoteAddr
	userAgent = r.Header.Get("User-Agent")

	return ipAddr, userAgent
}

func (app *application) createJSONResponse(data interface{}, e bool, msg ...string) JSONResponse {
	var respMsg string

	if msg != nil {
		respMsg = msg[0]
	}

	response := JSONResponse{
		Data: data,
		Meta: map[string]interface{}{
			"application": config.ApplicationName,
			"version":     config.ApplicationVersion,
			"author":      config.ApplicationAuthor,
		},
		Error:   e,
		Message: respMsg,
	}

	return response
}

func generateAPIKey(length int) (string, error) {
	// Determine the number of bytes needed based on the desired length
	numBytes := length * 3 / 4 // Base64 encoding uses 4 characters for every 3 bytes

	// Generate random bytes
	randomBytes := make([]byte, numBytes)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", err
	}

	// Encode random bytes to Base64
	apiKey := base64.URLEncoding.EncodeToString(randomBytes)

	// Trim padding characters (=) from the end of the string
	apiKey = apiKey[:length]

	return apiKey, nil
}

func (app *application) failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}

// nullableString is a function to convert a string to a nullable string
func (app *application) nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func (app *application) pointerString(s string) *string {
	return &s
}

func (app *application) limitStringByWords(s string, limit int) string {
	// Split the string into words
	words := strings.Fields(s)

	// Truncate the slice to the desired length
	if len(words) > limit {
		words = words[:limit]
	}

	// Join the words back into a string
	return strings.Join(words, " ")
}

// GetAESDecrypted decrypts given text in AES 256 CBC
func (app *application) GetAESDecrypted(encrypted string) ([]byte, error) {
	key := "Ft0xQc1y2F7bjmSlfV2vpW9w2IspWwJ6"
	iv := "fHid4wwHo9vkOds8"

	ciphertext, err := base64.StdEncoding.DecodeString(encrypted)

	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher([]byte(key))

	if err != nil {
		return nil, err
	}

	if len(ciphertext)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("block size cant be zero")
	}

	mode := cipher.NewCBCDecrypter(block, []byte(iv))
	mode.CryptBlocks(ciphertext, ciphertext)
	ciphertext = app.PKCS5UnPadding(ciphertext)

	return ciphertext, nil
}

// PKCS5UnPadding  pads a certain blob of data with necessary data to be used in AES block cipher
func (app *application) PKCS5UnPadding(src []byte) []byte {
	length := len(src)
	unpadding := int(src[length-1])

	return src[:(length - unpadding)]
}

// GetAESEncrypted encrypts given text in AES 256 CBC
func (app *application) GetAESEncrypted(plaintext string) (string, error) {
	key := "Ft0xQc1y2F7bjmSlfV2vpW9w2IspWwJ6"
	iv := "fHid4wwHo9vkOds8"

	var plainTextBlock []byte
	length := len(plaintext)

	if length%16 != 0 {
		extendBlock := 16 - (length % 16)
		plainTextBlock = make([]byte, length+extendBlock)
		copy(plainTextBlock[length:], bytes.Repeat([]byte{uint8(extendBlock)}, extendBlock))
	} else {
		plainTextBlock = make([]byte, length)
	}

	copy(plainTextBlock, plaintext)
	block, err := aes.NewCipher([]byte(key))

	if err != nil {
		return "", err
	}

	ciphertext := make([]byte, len(plainTextBlock))
	mode := cipher.NewCBCEncrypter(block, []byte(iv))
	mode.CryptBlocks(ciphertext, plainTextBlock)

	str := base64.StdEncoding.EncodeToString(ciphertext)

	return str, nil
}

func (app *application) createPaginationResponse(data []interface{}, totalItems, totalPages, page, perPage int, sort, order string) JSONResponse {

	response := JSONResponse{
		Data: data,
		Page: map[string]interface{}{
			"total":      totalItems,
			"total_page": totalPages,
			"current":    page,
			"per_page":   perPage,
			"from":       (page-1)*perPage + 1,
			"to":         page * perPage,
			"sort":       sort,
			"order":      order,
		},
		Meta: map[string]interface{}{
			"application": config.ApplicationName,
			"version":     config.ApplicationVersion,
			"author":      config.ApplicationAuthor,
		},
	}

	return response
}
