package models

// MyMessage represents the structure of the JSON message
type Message struct {
	ID       int    `json:"id"`
	EMAIL    string `json:"email"`
	USERNAME string `json:"username"`
	PASSWORD string `json:"password"`
}

var PROJECT_ID string
var TOPIC_ID string
var SUB_ID string
