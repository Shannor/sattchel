package core

// Link represents a one-way relationship between two goals.
// Generally will be a child to a parent goal
type Link struct {
	TargetID     string           `json:"targetId"`
	Relationship LinkRelationship `json:"relationship"`
	Description  string           `json:"description"`
}

type LinkRelationship string

const (
	LinkRequired LinkRelationship = "required"
	LinkOptional LinkRelationship = "optional"
)

func linkValue(link LinkRelationship) int {
	switch link {
	case LinkRequired:
		return 1
	case LinkOptional:
		return 2
	}
	return 0
}

func (r LinkRelationship) Compare(lr LinkRelationship) int {
	return linkValue(r) - linkValue(lr)
}
