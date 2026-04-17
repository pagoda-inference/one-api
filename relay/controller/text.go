package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/pagoda-inference/one-api/common/config"
	"github.com/pagoda-inference/one-api/common/logger"
	"github.com/pagoda-inference/one-api/model"
	"github.com/pagoda-inference/one-api/relay"
	"github.com/pagoda-inference/one-api/relay/adaptor"
	"github.com/pagoda-inference/one-api/relay/adaptor/openai"
	"github.com/pagoda-inference/one-api/relay/apitype"
	"github.com/pagoda-inference/one-api/relay/billing"
	billingratio "github.com/pagoda-inference/one-api/relay/billing/ratio"
	"github.com/pagoda-inference/one-api/relay/channeltype"
	"github.com/pagoda-inference/one-api/relay/meta"
	relaymodel "github.com/pagoda-inference/one-api/relay/model"
	"github.com/pagoda-inference/one-api/relay/relaymode"
)

func RelayTextHelper(c *gin.Context) *relaymodel.ErrorWithStatusCode {
	ctx := c.Request.Context()
	meta := meta.GetByContext(c)
	// get & validate textRequest
	textRequest, err := getAndValidateTextRequest(c, meta.Mode)
	if err != nil {
		logger.Errorf(ctx, "getAndValidateTextRequest failed: %s", err.Error())
		return openai.ErrorWrapper(err, "invalid_text_request", http.StatusBadRequest)
	}
	meta.IsStream = textRequest.Stream

	// map model name
	meta.OriginModelName = textRequest.Model
	textRequest.Model, _ = getMappedModelName(textRequest.Model, meta.ModelMapping)
	meta.ActualModelName = textRequest.Model
	// set system prompt if not empty
	systemPromptReset := setSystemPrompt(ctx, textRequest, meta.ForcedSystemPrompt)
	// get model price & group ratio
	modelInfo, err := model.GetModelById(textRequest.Model)
	if err != nil {
		logger.Warnf(ctx, "model not found: %s, using default price 0", textRequest.Model)
		modelInfo = nil
	}
	inputPrice := 0.0
	outputPrice := 0.0
	if modelInfo != nil {
		inputPrice = modelInfo.InputPrice
		outputPrice = modelInfo.OutputPrice
	}
	groupRatio := billingratio.GetGroupRatio(meta.Group)
	// pre-consume quota
	promptTokens := getPromptTokens(textRequest, meta.Mode)
	meta.PromptTokens = promptTokens
	preConsumedQuota, bizErr := preConsumeQuota(ctx, textRequest, promptTokens, inputPrice, groupRatio, meta)
	if bizErr != nil {
		logger.Warnf(ctx, "preConsumeQuota failed: %+v", *bizErr)
		return bizErr
	}

	adaptor := relay.GetAdaptor(meta.APIType)
	if adaptor == nil {
		return openai.ErrorWrapper(fmt.Errorf("invalid api type: %d", meta.APIType), "invalid_api_type", http.StatusBadRequest)
	}
	adaptor.Init(meta)

	// get request body
	requestBody, err := getRequestBody(c, meta, textRequest, adaptor)
	if err != nil {
		return openai.ErrorWrapper(err, "convert_request_failed", http.StatusInternalServerError)
	}

	// do request
	logger.Debugf(ctx, "DoRequest: APIType=%d, Mode=%d, ChannelType=%d", meta.APIType, meta.Mode, meta.ChannelType)
	resp, err := adaptor.DoRequest(c, meta, requestBody)
	if err != nil {
		logger.Errorf(ctx, "DoRequest failed: %s", err.Error())
		return openai.ErrorWrapper(err, "do_request_failed", http.StatusInternalServerError)
	}
	if isErrorHappened(meta, resp) {
		billing.ReturnPreConsumedQuota(ctx, preConsumedQuota, meta.TokenId)
		return RelayErrorHandler(resp)
	}

	// do response
	usage, respErr := adaptor.DoResponse(c, resp, meta)
	if respErr != nil {
		logger.Errorf(ctx, "respErr is not nil: %+v", respErr)
		billing.ReturnPreConsumedQuota(ctx, preConsumedQuota, meta.TokenId)
		return respErr
	}
	// post-consume quota
	go postConsumeQuota(ctx, usage, meta, textRequest, inputPrice, outputPrice, groupRatio, preConsumedQuota, systemPromptReset)
	return nil
}

func getRequestBody(c *gin.Context, meta *meta.Meta, textRequest *relaymodel.GeneralOpenAIRequest, adaptor adaptor.Adaptor) (io.Reader, error) {
	// For rerank, we need to map the model name in the body
	if meta.Mode == relaymode.Rerank {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			return nil, err
		}
		var reqBody map[string]any
		if err := json.Unmarshal(body, &reqBody); err != nil {
			return nil, err
		}
		// Model name mapping
		if modelName, ok := reqBody["model"].(string); ok {
			if mappedModel, ok := meta.ModelMapping[modelName]; ok {
				reqBody["model"] = mappedModel
			}
		}
		// Custom type (TGI) needs documents → texts conversion
		if meta.ChannelType == channeltype.Custom {
			if docs, ok := reqBody["documents"].([]any); ok {
				reqBody["texts"] = docs
				delete(reqBody, "documents")
			}
		}
		body, _ = json.Marshal(reqBody)
		return bytes.NewBuffer(body), nil
	}
	// Custom type (TGI) embedding needs input → inputs conversion
	if meta.Mode == relaymode.Embeddings && meta.ChannelType == channeltype.Custom {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			return nil, err
		}
		var reqBody map[string]any
		if err := json.Unmarshal(body, &reqBody); err != nil {
			return nil, err
		}
		// input → inputs conversion
		if input, ok := reqBody["input"]; ok {
			reqBody["inputs"] = input
			delete(reqBody, "input")
		}
		// model name mapping
		if modelName, ok := reqBody["model"].(string); ok {
			if mappedModel, ok := meta.ModelMapping[modelName]; ok {
				reqBody["model"] = mappedModel
			}
		}
		body, _ = json.Marshal(reqBody)
		return bytes.NewBuffer(body), nil
	}
	if !config.EnforceIncludeUsage &&
		meta.APIType == apitype.OpenAI &&
		meta.OriginModelName == meta.ActualModelName &&
		meta.ChannelType != channeltype.Baichuan &&
		meta.ForcedSystemPrompt == "" {
		// no need to convert request for openai
		return c.Request.Body, nil
	}

	// get request body
	var requestBody io.Reader
	convertedRequest, err := adaptor.ConvertRequest(c, meta.Mode, textRequest)
	if err != nil {
		logger.Debugf(c.Request.Context(), "converted request failed: %s\n", err.Error())
		return nil, err
	}
	jsonData, err := json.Marshal(convertedRequest)
	if err != nil {
		logger.Debugf(c.Request.Context(), "converted request json_marshal_failed: %s\n", err.Error())
		return nil, err
	}
	logger.Debugf(c.Request.Context(), "converted request: \n%s", string(jsonData))
	requestBody = bytes.NewBuffer(jsonData)
	return requestBody, nil
}
