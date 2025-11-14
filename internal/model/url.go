package model

type Code string

type URL string

func (U URL) String() string {
	return string(U)
}
