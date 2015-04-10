# postman-go [ ![Codeship Status for wannaup/postman-go](https://codeship.com/projects/7b0b4400-b115-0132-8a1c-3a7a9fb44a4e/status?branch=master)](https://codeship.com/projects/69724) [![Coverage Status](https://coveralls.io/repos/wannaup/postman-go/badge.svg?branch=HEAD)](https://coveralls.io/r/wannaup/postman-go)
The Golang version of our preferred postman mail to threaded messaging relay microservice in Go. 

## Requirements
Every request to the ```threads``` endpoint must be authenticated with basic HTTP auth header, the password **must not be empty** but actually it is not used/checked.
### Inbound
`POST  /inbound`

callback that manage inbound messages come from outside service

in:
``` 
```
out: 
``` 
```

### Threads
`GET   /threads`

return all threads of 'authenticated' user based on the auth header user

out: 
```
[
  {
    id: "...",
    meta: {
      "some": "meta",
      "other": { 
        "beautiful": [ "meta" ]
      }
    },
    owner: {
      id: "..."
    },
    msgs: [
      {
        from: "pinco@random.com",
        to: "pinco@random.com",
        msg: "hello!"
      }
    ]
  }
] 
```

`GET   /threads/:id`

return detail of a thread identified with id verifying the owner of the thread is actually the authenticated user

out: 
```  
{
  id: "...",
  meta: {
    "some": "meta",
    "other": { 
      "beautiful": [ "meta" ]
    }
  },
  owner: {
    id: "..."
  },
  msgs: [
    {
      from: "pinco@random.com",
      to: "pinco@random.com",
      msg: "hello!"
    }
  ]
}
```

`POST   /threads`

create a new thread, upon creation postman sends a mail containing the message *msg* to the *to* email address setting the sender as the *from* mail address and the *reply-to* field to the email address of the mail node (inbound.yourdomain.com). You can include additional info in the *meta* field, they will be always returned when you ask for the thread.

in:
``` 
{
  
  from: "pinco@random.com",
  to: "pinco@random.com",
  msg: "hello!",
  meta: {
    "some": "meta",
    "other": { 
      "beautiful": [ "meta" ]
    }
  }
  
}
```
out: 
``` 
{
  id: "...",
  owner: {
    id: "..."
  },
  msgs: [
    {
      from: "pinco@random.com",
      to: "pinco@random.com",
      msg: "hello!"
    }
  ]
}
```

`POST   /threads/<id>/reply`

reply with a new message

in:
``` 
{
  from: "pallo@random.com",
  msg: "Hi!"
}
```
out: 
``` 
{
  id: "...",
  owner: {
    id: "..."
  },
  msgs: [
    {
      from: "pinco@random.com",
      to: "pallo@random.com",
      msg: "hello!"
    },
    {
      from: "pallo@random.com",
      to: "pinco@random.com",
      msg: "Hi!"
    }
  ]
}
```
