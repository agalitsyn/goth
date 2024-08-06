# Go + Templ + TailwindCSS (and even bootstrap) + HTMX starter code

## Setup

```sh
cp .env.example .env
docker compose up -d postgres
make db-migrate-run
make db-create-superuser
```

For tailwind version run `make run-app` and for bootstrap 5.3 version run `make run-admin`
