package controller

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pagoda-inference/one-api/common/client"
	relaymodel "github.com/pagoda-inference/one-api/relay/model"
	"github.com/pagoda-inference/one-api/relay/adaptor/openai"
	relaymeta "github.com/pagoda-inference/one-api/relay/meta"
)

func RelayAudioPredictHelper(c *gin.Context) *relaymodel.ErrorWithStatusCode {
	meta := relaymeta.GetByContext(c)
	baseURL := strings.TrimSuffix(meta.BaseURL, "/")
	targetURL := baseURL + "/predict"

	mmseHeader, err := c.FormFile("mmse_file")
	if err != nil {
		return openai.ErrorWrapper(err, "missing_mmse_file", http.StatusBadRequest)
	}
	dsptHeader, err := c.FormFile("dspt_file")
	if err != nil {
		return openai.ErrorWrapper(err, "missing_dspt_file", http.StatusBadRequest)
	}

	mmseFile, err := mmseHeader.Open()
	if err != nil {
		return openai.ErrorWrapper(err, "open_mmse_file_failed", http.StatusBadRequest)
	}
	defer mmseFile.Close()

	dsptFile, err := dsptHeader.Open()
	if err != nil {
		return openai.ErrorWrapper(err, "open_dspt_file_failed", http.StatusBadRequest)
	}
	defer dsptFile.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	mmsePart, err := writer.CreateFormFile("mmse_file", mmseHeader.Filename)
	if err != nil {
		return openai.ErrorWrapper(err, "create_mmse_part_failed", http.StatusInternalServerError)
	}
	if _, err = io.Copy(mmsePart, mmseFile); err != nil {
		return openai.ErrorWrapper(err, "copy_mmse_file_failed", http.StatusInternalServerError)
	}

	dsptPart, err := writer.CreateFormFile("dspt_file", dsptHeader.Filename)
	if err != nil {
		return openai.ErrorWrapper(err, "create_dspt_part_failed", http.StatusInternalServerError)
	}
	if _, err = io.Copy(dsptPart, dsptFile); err != nil {
		return openai.ErrorWrapper(err, "copy_dspt_file_failed", http.StatusInternalServerError)
	}
	if err = writer.Close(); err != nil {
		return openai.ErrorWrapper(err, "close_multipart_writer_failed", http.StatusInternalServerError)
	}

	req, err := http.NewRequest(http.MethodPost, targetURL, &body)
	if err != nil {
		return openai.ErrorWrapper(err, "new_request_failed", http.StatusInternalServerError)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", c.Request.Header.Get("Authorization"))

	resp, err := client.HTTPClient.Do(req)
	if err != nil {
		return openai.ErrorWrapper(err, "do_request_failed", http.StatusInternalServerError)
	}
	defer resp.Body.Close()

	for k, vals := range resp.Header {
		for _, v := range vals {
			c.Writer.Header().Add(k, v)
		}
	}
	c.Status(resp.StatusCode)
	if _, err = io.Copy(c.Writer, resp.Body); err != nil {
		return openai.ErrorWrapper(err, "copy_response_body_failed", http.StatusInternalServerError)
	}
	return nil
}

