package model

type Layer struct {
	Digest    string `json:"digest"`
	Size      int64  `json:"size"`
	CreatedBy string `json:"createdBy"`
}
