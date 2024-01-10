
## Build docker image

```sh
docker build -t settleexpensesbot-img .
```

## Run the docker container

```sh
docker run -d --name settleexpensesbot-container -v ./:/app/snapshots --env-file .env --restart always settleexpensesbot-img
```