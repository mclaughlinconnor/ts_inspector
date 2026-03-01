package search

type tReadyState struct {
	embeddingsReady bool
	searchReady     bool
}

var readyState tReadyState = tReadyState{false, false}

func canSearch() bool {
	return readyState.searchReady
}

func canUseEmbeddings() bool {
	return readyState.embeddingsReady
}

func setSearchReady() {
	readyState.searchReady = true
}

func setEmbeddingsReady() {
	readyState.embeddingsReady = true
}
