package main

import (
    "gopkg.in/mgo.v2/bson"
)

type Owner struct {
    Id    bson.ObjectId `json:"id" bson:"id"`
}

type Message struct {
    From    string   `json:"from" bson:"from"` 
    To      string   `json:"to" bson:"to"` 
    Msg     string   `json:"msg" bson:"msg"` 
}

type Thread struct {
    Id    bson.ObjectId `json:"id" bson:"_id"`
    Owner   Owner    `json:"owner" bson:"owner"` 
    Messages    []Message   `json:"messages" bson:"messages"`
}