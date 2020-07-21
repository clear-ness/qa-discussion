# [QA Discussion](https://qadiscussion.com/)

Fundamental modern questions & answers system written in Golang and React/Typescript.
site: https://qadiscussion.com/

**Fundamental**  
almost all basic features
* ask question, answer to question, post a comment
* reply to participants of a comment thread or author of the post
* post with images
* fulltext search posts
* post tagging
* vote posts, register favorite posts
* receive inbox messages (about new answers or comments to your posts, and comment replies)
* email notification of new inbox messages
* three types of users: normal users, moderators, and admin (moderators can lock or protect posts, and suspend normal users)

**Modern**  
* written in Golang and runs with MySQL.  
* currently the backend of [https://qadiscussion.com/](https://qadiscussion.com/) is managed within aws ECS container environment.  
* frontend code is written in React with TypeScript. Frontend code coming soon... please wait!!  

**Production Grade**  
* scalable: db read-replicas addable by setting a config file, batch jobs on multiple batch servers.
* customizable: a lot of settings can be changed by editing a config file(config.json).
* tested: many features are sufficiently tested.
* security considered: csrf protection, rate limiter, no html parsed contents of posts (for preventing xss).

**Open Source**  
This repository's code itself is open source (MIT).

### Installation

install this repository
```
git clone https://github.com/clear-ness/qa-discussion.git
```

### Setup

before running this app, please create a config.json file like below and mv it into ./config dir.
```
{
    "ServiceSettings": {
        "SiteURL": "http://localhost:8080",
        "ListenAddress": ":8080",
        "EnableAdminUser": false,
        "AllowCorsFrom": "http://localhost:3000",
        "CorsExposedHeaders": "X-CSRF-Token X-Requested-With",
        "CorsAllowCredentials": true
    },
    "SqlSettings": {
        "DataSource": "root:@tcp(localhost:3306)/qa_discussion?charset=utf8mb4,utf8\u0026readTimeout=30s\u0026writeTimeout=30s"
    },
    "PasswordSettings" : {
        "Symbol": false
    },
    "RateLimitSettings": {
        "Enable": true
    }
}
```

install dev tools for migration, debug:

```
make dev-install
```

migration of db(mysql):

```
make migrate
```

run the server:
```
make run-server
```

### Build

you can make a go binary for linux environment:

```
make build-linux
```

after that, you can also create a docker image:

```
make docker-build
```

### Test

migration of test db(mysql):

```
make migrate-test
```

run tests:
```
make run-test
```

### License
This repository's code itself is MIT licensed.  
Some contents of [site](https://qadiscussion.com/) are from stackoverflow, so they are cc-wiki (aka cc-by-sa) licensed.  
You can confirm if the content is from stackoverflow by checking the prefix '000~' id of contents link.
For example, https://qadiscussion.com/questions/00000000000000000059626892 is from https://stackoverflow.com/questions/59626892
Other contents of [site](https://qadiscussion.com/) are also cc-wiki (aka cc-by-sa) licensed. Anyone can post anything on the site.
