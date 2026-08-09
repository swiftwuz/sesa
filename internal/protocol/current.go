package protocol

const Version = 2

type CurrentRepository struct {
	ProtocolVersion int      `json:"protocolVersion"`
	Repository      string   `json:"repository"`
	Mapped          bool     `json:"mapped"`
	Contexts        []string `json:"contexts"`
}

func NewCurrentRepository(repository string, contexts []string) CurrentRepository {
	allowed := make([]string, len(contexts))
	copy(allowed, contexts)
	return CurrentRepository{
		ProtocolVersion: Version,
		Repository:      repository,
		Mapped:          len(allowed) > 0,
		Contexts:        allowed,
	}
}
