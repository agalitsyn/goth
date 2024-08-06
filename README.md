# Go + Templ + TailwindCSS (and even bootstrap) + HTMX starter code

## Setup

```sh
cp .env.example .env
docker compose up -d postgres
make db-migrate-run
make db-create-superuser
```

For tailwind version run `make run-app` and for bootstrap 5.3 version run `make run-admin`

## Notes

- created both for references and for starter-kit
- have more library code than business-logic, because it was extracted from production app
- all code needed for app is vendored, cli tools are not vendored
- apps builds as a single binaries, assets are baked into binary
- frontend libs and templates are separate for each app
- includes DB tools like migrations, ready template for testing with database in docker and etc
- includes CLI tool example and several commands

Now Go and create your own full-stack framework.
