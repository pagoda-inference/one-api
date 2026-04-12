package channeltype

const (
	Unknown = iota
	OpenAI
	API2D
	Azure
	CloseAI
	OpenAISB
	OpenAIMax
	OhMyGPT
	Custom
	Ails
	AIProxy
	PaLM
	API2GPT
	AIGC2D
	Anthropic
	Baidu
	Zhipu
	Ali
	Xunfei
	AI360
	OpenRouter
	AIProxyLibrary
	FastGPT
	Tencent
	Gemini
	Moonshot
	Baichuan
	Minimax
	Mistral
	Groq
	Ollama
	LingYiWanWu
	StepFun
	AwsClaude
	Coze
	Cohere
	DeepSeek
	Cloudflare
	DeepL
	TogetherAI
	Doubao
	Novita
	VertextAI
	Proxy
	SiliconFlow
	XAI
	Replicate
	BaiduV2
	XunfeiV2
	AliBailian
	OpenAICompatible
	GeminiOpenAICompatible
	Dummy
)

// ChannelTypeNames maps channel type int to string name
var ChannelTypeNames = map[int]string{
	Unknown:                 "Unknown",
	OpenAI:                  "OpenAI",
	API2D:                   "API2D",
	Azure:                   "Azure",
	CloseAI:                 "CloseAI",
	OpenAISB:                "OpenAISB",
	OpenAIMax:               "OpenAIMax",
	OhMyGPT:                 "OhMyGPT",
	Custom:                  "自定义",
	Ails:                    "Ails",
	AIProxy:                 "AIProxy",
	PaLM:                    "PaLM",
	API2GPT:                 "API2GPT",
	AIGC2D:                  "AIGC2D",
	Anthropic:               "Anthropic",
	Baidu:                   "百度",
	Zhipu:                   "智谱",
	Ali:                     "阿里",
	Xunfei:                  "讯飞",
	AI360:                   "360智脑",
	OpenRouter:              "OpenRouter",
	AIProxyLibrary:          "AIProxyLibrary",
	FastGPT:                 "FastGPT",
	Tencent:                 "腾讯",
	Gemini:                  "Gemini",
	Moonshot:                "月之暗面",
	Baichuan:                "百川",
	Minimax:                 "MiniMax",
	Mistral:                 "Mistral",
	Groq:                    "Groq",
	Ollama:                  "Ollama",
	LingYiWanWu:             "零一万物",
	StepFun:                 "阶跃星辰",
	AwsClaude:               "AWSClaude",
	Coze:                    "Coze",
	Cohere:                  "Cohere",
	DeepSeek:                "DeepSeek",
	Cloudflare:              "Cloudflare",
	DeepL:                   "DeepL",
	TogetherAI:              "TogetherAI",
	Doubao:                  "豆包",
	Novita:                  "Novita",
	VertextAI:               "VertexAI",
	Proxy:                   "代理",
	SiliconFlow:             "SiliconFlow",
	XAI:                     "XAI",
	Replicate:               "Replicate",
	BaiduV2:                 "百度V2",
	XunfeiV2:                "讯飞V2",
	AliBailian:              "阿里百炼",
	OpenAICompatible:        "OpenAI兼容",
	GeminiOpenAICompatible:  "Gemini兼容",
	Dummy:                   "Dummy",
}

// GetTypeName returns the name of a channel type
func GetTypeName(t int) string {
	if name, ok := ChannelTypeNames[t]; ok {
		return name
	}
	return "Unknown"
}
