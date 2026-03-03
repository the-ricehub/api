# RiceHub API

Repository containing RiceHub's API source code.

Written in Go using Gin web framework.

## Usage documentation

You can find the API usage documentation and list of all endpoints on our [Postman](https://www.postman.com/unaimeds/workspace/ricehub/collection/38099323-b1c34138-7d47-462b-8ddd-8a5413931b4b?action=share&source=copy-link&creator=38099323).

## Building

To build this API you need to have some programming and API development knowledge. I tried my best keeping it as beginner-friendly as possible.

If you have hard time building the API, you're welcome to ask for help on our Discord server (link below).

### Requirements

- Working [Postgres server](https://www.postgresql.org/docs/current/tutorial-install.html) (latest version ideally),
- Working [Redis server](https://redis.io/docs/latest/operate/oss_and_stack/install/archive/install-redis/),
- Installed [Go](https://go.dev/doc/install) language

### Steps

The building steps below assume you are on a Unix-like system with all basic development tools installed.

1. Clone the repository:

```sh
git clone https://github.com/ricehub-io/api.git
cd api
```

2. Generate keys for asymmetric token verification/signing using provided script

```sh
./keys/generate.sh
```

3. Create config file by copying `config.toml.example` to `config.toml`

```sh
cp config.toml.example config.toml
```

4. Edit the `config.toml` using your favorite text editor

5. Import database schema from `schema.sql` file. I recommend doing that using your favorite database explorer (I personally use [DataGrip](https://www.jetbrains.com/datagrip/)).

6. Run the API in development mode

```sh
go run src/main.go
```

If everything was done correctly, you should be able to access the API at http://127.0.0.1:3000.

To build the API for production, go to the root of the repository, and run:

```sh
go build -o build/api ./src
```

The executable can be found in `build/` directory.

## Contributing

If you're interested in contributing to the project, please first read [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md). Then check out [CONTRIBUTING.md](CONTRIBUTING.md) file which contains all the important information on how to contribute.

If your question is still unanswered, feel free to open an issue or ask on Discord server (link provided below).

## Contact

If you need to contact us, you can do so either by sending us an email to [contact@ricehub.io](mailto:contact@ricehub.io) or via Discord server: https://discord.gg/z7Zu8MeTdG

---

You can find the previous version of README for this project in [README.old](README.old). It's more complex but gives you the general idea of how the API works under the hood.
