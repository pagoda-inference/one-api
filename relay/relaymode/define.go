package relaymode

const (
	Unknown = iota
	ChatCompletions
	Completions
	Embeddings
	Moderations
	ImagesGenerations
	Edits
	AudioSpeech
	AudioTranscription
	AudioTranslation
	AudioPredict
	FileParse
	FileParseTask
	FileParseTaskResult
	Rerank
	// Proxy is a special relay mode for proxying requests to custom upstream
	Proxy
	Responses
	// Video is the async OpenAI-compatible video generation create mode.
	// POST /v1/videos creates a task and returns immediately with the task id.
	Video
	// VideoSync is the synchronous OpenAI-compatible video generation create mode.
	// POST /v1/videos/sync creates a task and blocks, polling the upstream until
	// the task reaches a terminal state or the sync timeout elapses.
	VideoSync
)
