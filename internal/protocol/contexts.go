package protocol

type ContextList struct {
	ProtocolVersion int      `json:"protocolVersion"`
	Contexts        []string `json:"contexts"`
}

func NewContextList(contexts []string) ContextList {
	copyOfContexts := make([]string, len(contexts))
	copy(copyOfContexts, contexts)
	return ContextList{ProtocolVersion: Version, Contexts: copyOfContexts}
}
