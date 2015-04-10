package main

import (
    "gopkg.in/mgo.v2/bson"
    "time"
)

type Owner struct {
    Id    bson.ObjectId `json:"id" bson:"id"`
}

type Message struct {
    From    string   `json:"from" bson:"from"` 
    To      string   `json:"to" bson:"to"` 
    Msg     string   `json:"msg" bson:"msg"`
    Read	*time.Time	`json:"read,omitempty" bson:"read,omitempty"` 
}

type Thread struct {
    Id    bson.ObjectId `json:"id" bson:"_id"`
    Owner   Owner    `json:"owner" bson:"owner"`
    Meta	map[string]interface {}		`json:"meta,omitempty" bson:"meta,omitempty"` 
    Messages    []Message   `json:"messages" bson:"messages"`
}

type ThreadCreationReq struct {
	*Message
    Meta	map[string]interface {}		`json:"meta"`
}