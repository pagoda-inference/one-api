package model

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`

	CompletionTokensDetails *CompletionTokensDetails `json:"completion_tokens_details,omitempty"`
}

type CompletionTokensDetails struct {
	ReasoningTokens          int `json:"reasoning_tokens"`
	AcceptedPredictionTokens int `json:"accepted_prediction_tokens"`
	RejectedPredictionTokens int `json:"rejected_prediction_tokens"`
}

type Error struct {
	Message string `json:"message,omitempty"`
	Type    string `json:"type,omitempty"`
	Param   string `json:"param,omitempty"`
	Code    any    `json:"code,omitempty"`
}

type ErrorWithStatusCode struct {
	Error
	StatusCode int `json:"status_code"`
}
