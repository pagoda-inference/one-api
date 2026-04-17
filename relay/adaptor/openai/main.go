package openai

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/pagoda-inference/one-api/common/render"

	"github.com/gin-gonic/gin"
	"github.com/pagoda-inference/one-api/common"
	"github.com/pagoda-inference/one-api/common/conv"
	"github.com/pagoda-inference/one-api/common/logger"
	"github.com/pagoda-inference/one-api/relay/model"
	"github.com/pagoda-inference/one-api/relay/relaymode"
)

const (
	dataPrefix       = "data: "
	done             = "[DONE]"
	dataPrefixLength = len(dataPrefix)
)

func StreamHandler(c *gin.Context, resp *http.Response, relayMode int, originModelName string, hideUpstreamModel bool) (*model.ErrorWithStatusCode, string, *model.Usage) {
	responseText := ""
	scanner := bufio.NewScanner(resp.Body)
	scanner.Split(bufio.ScanLines)
	var usage *model.Usage

	common.SetEventStreamHeaders(c)

	doneRendered := false
	for scanner.Scan() {
		data := scanner.Text()
		if len(data) < dataPrefixLength { // ignore blank line or wrong format
			continue
		}
		if data[:dataPrefixLength] != dataPrefix && data[:dataPrefixLength] != done {
			continue
		}
		if strings.HasPrefix(data[dataPrefixLength:], done) {
			render.StringData(c, data)
			doneRendered = true
			continue
		}
		switch relayMode {
		case relaymode.ChatCompletions:
			var streamResponse ChatCompletionsStreamResponse
			err := json.Unmarshal([]byte(data[dataPrefixLength:]), &streamResponse)
			if err != nil {
				logger.SysError("error unmarshalling stream response: " + err.Error())
				render.StringData(c, data) // if error happened, pass the data to client
				continue                   // just ignore the error
			}
			if len(streamResponse.Choices) == 0 && streamResponse.Usage == nil {
				// but for empty choice and no usage, we should not pass it to client, this is for azure
				continue // just ignore empty choice
			}
			// If hideUpstreamModel is enabled, replace the model name
			if hideUpstreamModel && originModelName != "" {
				streamResponse.Model = originModelName
				modifiedData := dataPrefix + toJson(streamResponse)
				render.StringData(c, modifiedData)
			} else {
				render.StringData(c, data)
			}
			for _, choice := range streamResponse.Choices {
				responseText += conv.AsString(choice.Delta.Content)
			}
			if streamResponse.Usage != nil {
				usage = streamResponse.Usage
			}
		case relaymode.Completions:
			var streamResponse CompletionsStreamResponse
			err := json.Unmarshal([]byte(data[dataPrefixLength:]), &streamResponse)
			if err != nil {
				logger.SysError("error unmarshalling stream response: " + err.Error())
				continue
			}
			// If hideUpstreamModel is enabled, replace the model name
			if hideUpstreamModel && originModelName != "" {
				streamResponse.Model = originModelName
				modifiedData := dataPrefix + toJson(streamResponse)
				render.StringData(c, modifiedData)
			} else {
				render.StringData(c, data)
			}
			for _, choice := range streamResponse.Choices {
				responseText += choice.Text
			}
		}
	}

	if err := scanner.Err(); err != nil {
		logger.SysError("error reading stream: " + err.Error())
	}

	if !doneRendered {
		render.Done(c)
	}

	err := resp.Body.Close()
	if err != nil {
		return ErrorWrapper(err, "close_response_body_failed", http.StatusInternalServerError), "", nil
	}

	return nil, responseText, usage
}

func Handler(c *gin.Context, resp *http.Response, promptTokens int, modelName string, originModelName string, hideUpstreamModel bool) (*model.ErrorWithStatusCode, *model.Usage) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return ErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError), nil
	}
	err = resp.Body.Close()
	if err != nil {
		return ErrorWrapper(err, "close_response_body_failed", http.StatusInternalServerError), nil
	}

	// Try SlimTextResponse first (chat completions)
	var textResponse SlimTextResponse
	if err := json.Unmarshal(responseBody, &textResponse); err == nil {
		if textResponse.Error != nil && textResponse.Error.Type != "" {
			return &model.ErrorWithStatusCode{
				Error:      *textResponse.Error,
				StatusCode: resp.StatusCode,
			}, nil
		}

		// If hideUpstreamModel is enabled, replace the model name
		outputBody := responseBody
		if hideUpstreamModel && originModelName != "" {
			textResponse.Model = originModelName
			outputBody, err = json.Marshal(textResponse)
			if err != nil {
				return ErrorWrapper(err, "marshal_response_body_failed", http.StatusInternalServerError), nil
			}
		}

		// Reset response body
		resp.Body = io.NopCloser(bytes.NewBuffer(outputBody))

		for k, v := range resp.Header {
			c.Writer.Header().Set(k, v[0])
		}
		c.Writer.WriteHeader(resp.StatusCode)
		_, err = io.Copy(c.Writer, resp.Body)
		if err != nil {
			return ErrorWrapper(err, "copy_response_body_failed", http.StatusInternalServerError), nil
		}
		err = resp.Body.Close()
		if err != nil {
			return ErrorWrapper(err, "close_response_body_failed", http.StatusInternalServerError), nil
		}

		if textResponse.Usage.TotalTokens == 0 || (textResponse.Usage.PromptTokens == 0 && textResponse.Usage.CompletionTokens == 0) {
			completionTokens := 0
			for _, choice := range textResponse.Choices {
				completionTokens += CountTokenText(choice.Message.StringContent(), modelName)
			}
			textResponse.Usage = model.Usage{
				PromptTokens:     promptTokens,
				CompletionTokens: completionTokens,
				TotalTokens:      promptTokens + completionTokens,
			}
		}
		return nil, &textResponse.Usage
	}

	// Try RerankResponse
	var rerankResponse RerankResponse
	if err := json.Unmarshal(responseBody, &rerankResponse); err == nil {
		// Rerank response - if hideUpstreamModel is enabled, replace the model name
		if hideUpstreamModel && originModelName != "" {
			rerankResponse.Model = originModelName
			responseBody, err = json.Marshal(rerankResponse)
			if err != nil {
				return ErrorWrapper(err, "marshal_rerank_response_failed", http.StatusInternalServerError), nil
			}
		}

		resp.Body = io.NopCloser(bytes.NewBuffer(responseBody))

		for k, v := range resp.Header {
			c.Writer.Header().Set(k, v[0])
		}
		c.Writer.WriteHeader(resp.StatusCode)
		_, err = io.Copy(c.Writer, resp.Body)
		if err != nil {
			return ErrorWrapper(err, "copy_response_body_failed", http.StatusInternalServerError), nil
		}
		err = resp.Body.Close()
		if err != nil {
			return ErrorWrapper(err, "close_response_body_failed", http.StatusInternalServerError), nil
		}
		return nil, rerankResponse.Usage
	}

	// Fallback: unmarshal failed for both types, just pass through
	for k, v := range resp.Header {
		c.Writer.Header().Set(k, v[0])
	}
	c.Writer.WriteHeader(resp.StatusCode)
	_, err = c.Writer.Write(responseBody)
	if err != nil {
		return ErrorWrapper(err, "write_response_body_failed", http.StatusInternalServerError), nil
	}
	return nil, nil
}

// EmbeddingHandler handles embedding responses, especially for TGI backends
// that return array format [[0.1, 0.2, ...]] instead of OpenAI format
func EmbeddingHandler(c *gin.Context, resp *http.Response, modelName string, originModelName string, hideUpstreamModel bool) *model.ErrorWithStatusCode {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return ErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
	}
	err = resp.Body.Close()
	if err != nil {
		return ErrorWrapper(err, "close_response_body_failed", http.StatusInternalServerError)
	}

	// Try to unmarshal as OpenAI format first
	var openaiResponse EmbeddingResponse
	if err := json.Unmarshal(responseBody, &openaiResponse); err == nil {
		// Already in OpenAI format
		// If hideUpstreamModel is enabled, replace the model name
		if hideUpstreamModel && originModelName != "" {
			openaiResponse.Model = originModelName
			responseBody, _ = json.Marshal(openaiResponse)
		}
		for k, v := range resp.Header {
			c.Writer.Header().Set(k, v[0])
		}
		c.Writer.WriteHeader(resp.StatusCode)
		_, err = c.Writer.Write(responseBody)
		if err != nil {
			return ErrorWrapper(err, "write_response_body_failed", http.StatusInternalServerError)
		}
		return nil
	}

	// TGI returns array format [[0.1, 0.2, ...]], need to convert to OpenAI format
	var tgiEmbedding [][]float64
	if err := json.Unmarshal(responseBody, &tgiEmbedding); err != nil {
		return ErrorWrapper(err, "unmarshal_embedding_response_failed", http.StatusInternalServerError)
	}

	// Convert to OpenAI embedding response format
	// Use originModelName if hideUpstreamModel is enabled, otherwise use modelName
	respModel := modelName
	if hideUpstreamModel && originModelName != "" {
		respModel = originModelName
	}
	openaiResp := EmbeddingResponse{
		Object: "list",
		Model:  respModel,
	}
	for i, embedding := range tgiEmbedding {
		openaiResp.Data = append(openaiResp.Data, EmbeddingResponseItem{
			Object:    "embedding",
			Index:     i,
			Embedding: embedding,
		})
	}

	// Write converted response
	respBody, _ := json.Marshal(openaiResp)
	for k, v := range resp.Header {
		if k == "Content-Type" {
			c.Writer.Header().Set(k, "application/json")
		} else {
			c.Writer.Header().Set(k, v[0])
		}
	}
	c.Writer.WriteHeader(http.StatusOK)
	_, err = c.Writer.Write(respBody)
	if err != nil {
		return ErrorWrapper(err, "write_response_body_failed", http.StatusInternalServerError)
	}
	return nil
}

// toJson marshals a value to JSON string
func toJson(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
