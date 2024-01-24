<img src="./assets/bot-avatar.png" width=250/>

# SettleExpensesBot
Simple Telegram Bot to manage group expenses and calculate the best options to pay. Written in Go.

#### Supported commands
* [/add](#supported-commands) - Adds an expense for you.
* [/addfor](#supported-commands) - Adds an expense for another user.
* [/expenses](#supported-commands) - Lists all the expenses with their IDs and allows to remove them.
* [/summary](#supported-commands) - Shows a summary of current debs and allows to settle them.
* [/import](#supported-commands) - Import expenses from a csv file.
* [/export](#supported-commands) - Export expenses to a csv file.
* [/help](#supported-commands) - Shows help message.

## How to host your bot?

### Requirements
* [Docker](https://www.docker.com/get-started/) 🐳 or [Go](https://go.dev/learn/) installed in your computer or server.
* A valid [Telegram Bot API token](https://core.telegram.org/bots/tutorial#obtain-your-bot-token).

### Run with Docker

1. **Create your own `.env`:** Copy the template and edit it with your own information. The template contains useful examples.

    ```sh
    cp example.env .env
    ```

2. **Build docker image:** Build the docker image in the repo root path.

    ```sh
    docker build -t settleexpensesbot-img .
    ```

3. **Run the docker container:** Run a container with the built image, mounting a volume to the current folder (to store data snapshots and logs) and defining the path of your `.env` file. 

    ```sh
    docker run -d \
        --name settleexpensesbot-container \
        -v ./:/app/data \
        --env-file .env \
        --restart always \
        settleexpensesbot-img
    ```

### Run with Go (for debug)

* **Run the bot**: Run the go command to start your bot defining the log level, the Telegram API token and the admin usernames and aliases.

    ```sh
    LOG_LEVEL=debug TELEGRAM_TOKEN=123456789:ABCDEF ADMIN_USER_IDS=11111 ADMIN_USER_ALIASES=super-dev go run ./cmd/bot
    ```