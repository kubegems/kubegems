# quick start  develop environment 


##  depends

1. gitea

    docker-compose.yaml

        version: "3"
        networks:
        gitea:
            external: false
        services:
        server:
            image: gitea/gitea:1.15.0
            container_name: gitea
            environment:
            - USER_UID=1000
            - USER_GID=1000
            restart: always
            networks:
            - gitea
            volumes:
            - ./gitea:/data
            - /etc/timezone:/etc/timezone:ro
            - /etc/localtime:/etc/localtime:ro
            ports:
            - "13000:3000"
            - "10022:22"

    `docker-compose up -d`

2. redis

    `docker run --name redis -d -p6379:6379 redis`



## start dev server

local_debug.go

    package main

    import (
        "kubegems.io/pkg/services"
    )

    func main() {
        services.LocalDevRun()
    }

`go run local_debug.go`

TODO: //