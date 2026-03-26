package aiproxy

import "github.com/pagoda-inference/one-api/relay/adaptor/openai"

var ModelList = []string{""}

func init() {
	ModelList = openai.ModelList
}
