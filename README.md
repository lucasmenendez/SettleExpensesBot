
## Build docker image

```sh
docker build -t expensesbot-img .
```

## Run the docker container

```sh
docker run -d --name expensesbot-container -v ./:/app/snapshots --env-file .env --restart always expensesbot-img
```