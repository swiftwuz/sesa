package protocol

const Version = 1

type CurrentRepository struct {
	ProtocolVersion int     `json:"protocolVersion"`
	Repository      string  `json:"repository"`
	Mapped          bool    `json:"mapped"`
	Context         *string `json:"context"`
}

func NewCurrentRepository(repository string, context *string) CurrentRepository {
	return CurrentRepository{
		ProtocolVersion: Version,
		Repository:      repository,
		Mapped:          context != nil,
		Context:         context,
	}
}
