package main

import (
    "os"
    "fmt"
    "log"
    "net/http"
    "io"
    "flag"
    "bytes"
    "errors"
    "encoding/json"
    "encoding/base64"
    "strings"
    "github.com/codegangsta/negroni"
    "github.com/gorilla/mux"
    "github.com/gorilla/context"
    "gopkg.in/mgo.v2"
    "gopkg.in/mgo.v2/bson"
)



type key int

const db key = 0
const userId key = 1
var config map[string]string

func main() {
    mode := flag.String("m", "d", "d=debug p=production")
    flag.Parse()
    if *mode == "d" {
        PreFlight("conf_debug.json")
    }else{   //production
        PreFlight("")
    }
    n := StirNegroni()
    //and run
    n.Run(":" + config["PORT"])
}

func PreFlight(fname string){
    //load configuration
    LoadConfig(fname, &config)
}

//associates the routes to the router
func StirNegroni() *negroni.Negroni{
    n := negroni.Classic()
    //routes
    rt := mux.NewRouter().StrictSlash(true)
    rt.HandleFunc("/inbound", ProcessInbound).Methods("POST")  //OK
    rt.HandleFunc("/inbound", HeadInbound).Methods("HEAD")     
    authRoutes := mux.NewRouter().StrictSlash(true)
    authRoutes.HandleFunc("/threads", CreateThread).Methods("POST")     //OK
    authRoutes.HandleFunc("/threads", GetAllThreads).Methods("GET")     //OK
    authRoutes.HandleFunc("/threads/{threadId}", GetOneThread).Methods("GET")   //OK
    authRoutes.HandleFunc("/threads/{threadId}/reply", ReplyThread).Methods("POST") //OK
    rt.PathPrefix("/threads").Handler(negroni.New(
        negroni.HandlerFunc(BasicAuthMiddleware),
        negroni.Wrap(authRoutes),
    ))
    //some middleware
    n.Use(MongoMiddleware())
    // router goes last
    n.UseHandler(rt)
    return n
}

func HeadInbound(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte("OK"))
}

func ProcessInbound(w http.ResponseWriter, r *http.Request) {
    r.ParseMultipartForm(884808408)
    mPostValue := r.FormValue("mandrill_events")
    mEvents := []MandrillEvent{}
    err := UnmarshalObject(bytes.NewBuffer([]byte(mPostValue)), &mEvents)
    if err != nil{
        http.Error(w, "Your JSON is not GOOD", http.StatusBadRequest)
        log.Println(err.Error())
        log.Println(mPostValue)
        return
    }
    thedb := context.Get(r, db).(*mgo.Database)
    tColl := thedb.C("message_threads")
    //process inbound messages
    for _, val := range mEvents {   //TODO one goroutine for each event could be interesting
        if len(val.Msg.To) != 1{    //only one recipient is allowed
            log.Println("Invalid TO field in inbound message.")
            continue
        }
        //get thread id from To field
        var tId = strings.Split(val.Msg.To[0][0], "@")[0]
        if !bson.IsObjectIdHex(tId) {
            log.Println("Invalid thread identifier in address:", val.Msg.To)
            continue
        }
        // TODO set message header unique id to avoid reprocessing
        var thread Thread
        nMsg := Message{From: val.Msg.From_email, Msg: val.Msg.Text}
        err = AddThreadReply(tColl, tId, "", &nMsg, &thread)
        if err != nil{
            log.Println("Can't reply thread from inbound: %v\n", err)
            continue
        }
        //send mail 
        go NewMailProvider(config).SendMail(thread.Id.Hex(), nMsg.From, []string{nMsg.To}, nMsg.Msg)
    }
    w.Write([]byte("OK"))
}

func CreateThread(w http.ResponseWriter, r *http.Request) {
    var nMsg Message
    err := UnmarshalObject(r.Body, &nMsg)
    if err != nil{
        http.Error(w, "Your JSON is not GOOD", http.StatusBadRequest)
        return
    }
    thedb := context.Get(r, db).(*mgo.Database)
    tColl := thedb.C("message_threads")
    var owner = Owner{bson.ObjectIdHex(context.Get(r, userId).(string))}
    nThread := Thread{bson.NewObjectId(), owner, []Message{nMsg}}
    err = tColl.Insert(nThread)
    if err != nil {
        panic("Can't create thread:" + err.Error())
    }
    //actually send out the mail
    go NewMailProvider(config).SendMail(nThread.Id.Hex(), nMsg.From, []string{nMsg.To}, nMsg.Msg)
    //config thread creation
    JSONResponse(w, nThread)
}

//gets all the threads for which the authenticated user is the owner
func GetAllThreads(w http.ResponseWriter, r *http.Request) {
    thedb := context.Get(r, db).(*mgo.Database)
    threads := []Thread{}
    tColl := thedb.C("message_threads")
    iter := tColl.Find(bson.M{"owner.id": bson.ObjectIdHex(context.Get(r, userId).(string))}).Limit(50).Iter()
    err := iter.All(&threads)
    if err != nil {
        log.Fatal(err)
    }
    JSONResponse(w, threads)
}

//return the requested thread, verifying the owner is the authenticated user
func GetOneThread(w http.ResponseWriter, r *http.Request) {
    tId := mux.Vars(r)["threadId"]
    //verify threadid is a valid objectid
    if !bson.IsObjectIdHex(tId) {
        http.Error(w, "The thread id provided is not valid", http.StatusBadRequest)
        return
    }
    thedb := context.Get(r, db).(*mgo.Database)
    var thread Thread
    tColl := thedb.C("message_threads")
    err := tColl.Find(bson.M{"_id": bson.ObjectIdHex(tId),"owner.id": bson.ObjectIdHex(context.Get(r, userId).(string))}).One(&thread)
    if err != nil {
        http.Error(w, "NOT FOUND", http.StatusNotFound)
        return
    }

    JSONResponse(w, thread)
}

//handles request to add a reply to an existing thread
func ReplyThread(w http.ResponseWriter, r *http.Request) {
    tId := mux.Vars(r)["threadId"]
    //verify threadid is a valid objectid
    if !bson.IsObjectIdHex(tId) {
        http.Error(w, "Invalid thread id", http.StatusBadRequest)
        return
    }
    //unmarshal the new message to add
    var nMsg Message
    err := UnmarshalObject(r.Body, &nMsg)
    if err != nil{
        http.Error(w, "Your JSON is not GOOD", http.StatusBadRequest)
        return
    }
    bson.ObjectIdHex(context.Get(r, userId).(string))
    var thread Thread
    tColl := context.Get(r, db).(*mgo.Database).C("message_threads")
    err = AddThreadReply(tColl, tId, context.Get(r, userId).(string), &nMsg, &thread)
    if err != nil{
        log.Println("Can't reply thread: %v", err)
        http.Error(w, "Can't reply thread", http.StatusInternalServerError)
    }
    //everything ok, send the mail
    go NewMailProvider(config).SendMail(thread.Id.Hex(), nMsg.From, []string{nMsg.To}, nMsg.Msg)

    JSONResponse(w, thread)
}

//adds a new reply to an existing thread, updates the passed message and the passed thread
func AddThreadReply(tColl *mgo.Collection, tId string, ownerId string, nMsg *Message, thread *Thread) error {
    //ensure owner or not?
    qm := make(map[string]interface{})
    qm["_id"] = bson.ObjectIdHex(tId)
    if ownerId != "" {
        qm["owner.id"] = bson.ObjectIdHex(ownerId)
    }
    err := tColl.Find(qm).One(&thread)
    if err != nil {
        return errors.New("Thread not found")
    }
    for i := len(thread.Messages)-1; i >= 0; i-- {
        if thread.Messages[i].From != nMsg.From {
            nMsg.To = thread.Messages[i].From
            break
        }
    }
    //found?
    if nMsg.To == "" {
        return errors.New("Can't do this, loop will be")
    }
    //ready for update
    err = tColl.Update(qm, bson.M{"$push": bson.M{"messages": nMsg}})
    if err != nil {
        return errors.New("Can't update your thread")
    }
    //update the thread struct
    thread.Messages = append(thread.Messages, *nMsg)
    return nil
}

func JSONResponse(w http.ResponseWriter, m interface{}) {
    j, err := json.Marshal(m)
    if err != nil {
        panic(err)
    }
    w.Header().Set("Content-Type", "application/json")
    w.Write(j)
}

func MongoMiddleware() negroni.HandlerFunc {
    database := config["DBNAME"] //os.Getenv("DB_NAME")
    session, err := mgo.Dial(config["DBURI"])
    
    if err != nil {
        panic(err)
    }
    session.SetMode(mgo.Monotonic, true)
    
    return negroni.HandlerFunc(func(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
        reqSession := session.Clone()
        defer reqSession.Close()
        thedb := reqSession.DB(database)
        context.Set(r, db, thedb)
        next(rw, r)
    })
}

//exploits basic auth to auth the user making the request (MessageThread owner)
func BasicAuthMiddleware(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
    if len(r.Header["Authorization"]) == 0 {
        http.Error(rw, "NO auth header", http.StatusBadRequest)
        return
    }
    auth := strings.SplitN(r.Header["Authorization"][0], " ", 2)
    if len(auth) != 2 || auth[0] != "Basic" {
        http.Error(rw, "NO auth header", http.StatusBadRequest)
        return
    }
 
    payload, _ := base64.StdEncoding.DecodeString(auth[1])
    pair := strings.SplitN(string(payload), ":", 2)
    if len(pair) != 2 || !IsUserIdValid(pair[0]){
        http.Error(rw, "authorization failed", http.StatusUnauthorized)
        return
    }
    
    context.Set(r, userId, pair[0])
    next(rw, r)
    // do some stuff after
}
 

//this unmarshals json
func UnmarshalObject(body io.Reader, obj interface{}) error{
    return json.NewDecoder(body).Decode(obj)
}

//verifies the provided auth header values are actually valid 
func IsUserIdValid(uid string) bool {
    return bson.IsObjectIdHex(uid)
}

//loads the app configuration
func LoadConfig(fname string, config *map[string]string) {
    if fname != "" {
        file, _ := os.Open(fname)
        defer file.Close()
        decoder := json.NewDecoder(file)
        err := decoder.Decode(config)
        if err != nil {
            fmt.Println("error loading go config file:", err)
            panic(err)
        }
    }else{
        *config = map[string]string{
            "ENVIRONMENT" : os.Getenv("ENVIRONMENT"),
            "PORT": os.Getenv("PORT"),
            "DBURI": os.Getenv("DBURI"),
            "DBNAME": os.Getenv("DBNAME"),
            "INBOUND_EMAIL_DOMAIN": os.Getenv("INBOUND_EMAIL_DOMAIN"),
            "MAIL_PROVIDER": os.Getenv("MAIL_PROVIDER"),
            "MANDRILL_API_HOST": os.Getenv("MANDRILL_API_HOST"),
            "MANDRILL_API_URL": os.Getenv("MANDRILL_API_URL"),
            "MANDRILL_API_KEY": os.Getenv("MANDRILL_API_KEY"),
        }
    }
}

