# nsi_survey_server

env GOOS=linux GOARCH=amd64 go build

### Dev environment

The development env uses two bind mounts to connect the current local workspace and local microauth package to the container. To deploy development environment,
setup auth public key in pk.pem and link via IPPK env var. To connect to container:

    docker-compose -f deploy/dev/docker-compose.yaml up -d
    docker exec -it NSISERVER_DEV bash

Env variables:

    PORT=3031
    DBUSER=
    DBPASS=
    DBNAME=
    DBHOST=
    DBSTORE=pgx
    DBDRIVER=postgres
    DBSSLMODE=
    DBPORT=
    IPPK=

To override using an .env file:

    export $(grep -v '^#' .env | xargs)

To forward local env to DB via an ssh tunnel:

    ssh -i ./<private key> -NL <localport>:<db server host>:<db server port>> <tunnel username>@<tunnel host>
