package relay

import (
	"github.com/pagoda-inference/one-api/relay/adaptor"
	"github.com/pagoda-inference/one-api/relay/adaptor/aiproxy"
	"github.com/pagoda-inference/one-api/relay/adaptor/ali"
	"github.com/pagoda-inference/one-api/relay/adaptor/anthropic"
	"github.com/pagoda-inference/one-api/relay/adaptor/aws"
	"github.com/pagoda-inference/one-api/relay/adaptor/baidu"
	"github.com/pagoda-inference/one-api/relay/adaptor/cloudflare"
	"github.com/pagoda-inference/one-api/relay/adaptor/cohere"
	"github.com/pagoda-inference/one-api/relay/adaptor/coze"
	"github.com/pagoda-inference/one-api/relay/adaptor/deepl"
	"github.com/pagoda-inference/one-api/relay/adaptor/gemini"
	"github.com/pagoda-inference/one-api/relay/adaptor/ollama"
	"github.com/pagoda-inference/one-api/relay/adaptor/openai"
	"github.com/pagoda-inference/one-api/relay/adaptor/palm"
	"github.com/pagoda-inference/one-api/relay/adaptor/proxy"
	"github.com/pagoda-inference/one-api/relay/adaptor/replicate"
	"github.com/pagoda-inference/one-api/relay/adaptor/tencent"
	"github.com/pagoda-inference/one-api/relay/adaptor/vertexai"
	"github.com/pagoda-inference/one-api/relay/adaptor/xunfei"
	"github.com/pagoda-inference/one-api/relay/adaptor/zhipu"
	"github.com/pagoda-inference/one-api/relay/apitype"
)

func GetAdaptor(apiType int) adaptor.Adaptor {
	switch apiType {
	case apitype.AIProxyLibrary:
		return &aiproxy.Adaptor{}
	case apitype.Ali:
		return &ali.Adaptor{}
	case apitype.Anthropic:
		return &anthropic.Adaptor{}
	case apitype.AwsClaude:
		return &aws.Adaptor{}
	case apitype.Baidu:
		return &baidu.Adaptor{}
	case apitype.Gemini:
		return &gemini.Adaptor{}
	case apitype.OpenAI:
		return &openai.Adaptor{}
	case apitype.PaLM:
		return &palm.Adaptor{}
	case apitype.Tencent:
		return &tencent.Adaptor{}
	case apitype.Xunfei:
		return &xunfei.Adaptor{}
	case apitype.Zhipu:
		return &zhipu.Adaptor{}
	case apitype.Ollama:
		return &ollama.Adaptor{}
	case apitype.Coze:
		return &coze.Adaptor{}
	case apitype.Cohere:
		return &cohere.Adaptor{}
	case apitype.Cloudflare:
		return &cloudflare.Adaptor{}
	case apitype.DeepL:
		return &deepl.Adaptor{}
	case apitype.VertexAI:
		return &vertexai.Adaptor{}
	case apitype.Proxy:
		return &proxy.Adaptor{}
	case apitype.Replicate:
		return &replicate.Adaptor{}
	}
	return nil
}
