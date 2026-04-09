// Package controller is a package for handling the relay controller
package controller

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pagoda-inference/one-api/common/logger"
	"github.com/pagoda-inference/one-api/relay"
	"github.com/pagoda-inference/one-api/relay/adaptor/openai"
	proxyadaptor "github.com/pagoda-inference/one-api/relay/adaptor/proxy"
	"github.com/pagoda-inference/one-api/relay/channeltype"
	"github.com/pagoda-inference/one-api/relay/meta"
	relaymodel "github.com/pagoda-inference/one-api/relay/model"
)

// RelayProxyHelper is a helper function to proxy the request to the upstream service
func RelayProxyHelper(c *gin.Context, relayMode int) *relaymodel.ErrorWithStatusCode {
	ctx := c.Request.Context()
	meta := meta.GetByContext(c)

	adaptor := relay.GetAdaptor(meta.APIType)
	if adaptor == nil {
		return openai.ErrorWrapper(fmt.Errorf("invalid api type: %d", meta.APIType), "invalid_api_type", http.StatusBadRequest)
	}
	adaptor.Init(meta)

	resp, err := adaptor.DoRequest(c, meta, c.Request.Body)
	if err != nil {
		logger.Errorf(ctx, "DoRequest failed: %s", err.Error())
		return openai.ErrorWrapper(err, "do_request_failed", http.StatusInternalServerError)
	}

	// do response
	_, respErr := adaptor.DoResponse(c, resp, meta)
	if respErr != nil {
		logger.Errorf(ctx, "respErr is not nil: %+v", respErr)
		return respErr
	}

	return nil
}

// RelayRerankHelper handles rerank requests by proxying to the upstream service
func RelayRerankHelper(c *gin.Context) *relaymodel.ErrorWithStatusCode {
	ctx := c.Request.Context()
	meta := meta.GetByContext(c)

	// Get the appropriate adaptor based on API type for request
	adaptor := relay.GetAdaptor(meta.APIType)
	if adaptor == nil {
		return openai.ErrorWrapper(fmt.Errorf("invalid api type: %d", meta.APIType), "invalid_api_type", http.StatusBadRequest)
	}
	adaptor.Init(meta)

	// Transform request body: model mapping + documents→texts conversion
	requestBody, err := getRequestBody(c, meta, nil, adaptor)
	if err != nil {
		logger.Errorf(ctx, "getRequestBody failed: %s", err.Error())
		return openai.ErrorWrapper(err, "convert_request_failed", http.StatusInternalServerError)
	}

	resp, err := adaptor.DoRequest(c, meta, requestBody)
	if err != nil {
		logger.Errorf(ctx, "DoRequest failed: %s", err.Error())
		return openai.ErrorWrapper(err, "do_request_failed", http.StatusInternalServerError)
	}

	// For Custom (TGI) channels, use proxy adaptor for response passthrough
	// For other channels (OpenAICompatible/vLLM), use the native adaptor
	var respErr *relaymodel.ErrorWithStatusCode
	if meta.ChannelType == channeltype.Custom {
		proxyAdaptor := &proxyadaptor.Adaptor{}
		proxyAdaptor.Init(meta)
		_, respErr = proxyAdaptor.DoResponse(c, resp, meta)
	} else {
		_, respErr = adaptor.DoResponse(c, resp, meta)
	}
	if respErr != nil {
		logger.Errorf(ctx, "respErr is not nil: %+v", respErr)
		return respErr
	}

	return nil
}
