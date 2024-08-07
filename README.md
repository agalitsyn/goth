# GOTH

Go + Templ + TailwindCSS (and even bootstrap) + HTMX starter code

![](./docs/images/image.png)

## Setup

```sh
cp .env.example .env
docker compose up -d postgres
make db-migrate-run
make db-create-superuser
```

For tailwind version run `make run-app` and for bootstrap 5.3 version run `make run-admin`

## Notes

- Repo was created for reference and for usage as a starter-kit.
- It has more library code than business-logic code because it was extracted from production app. Some code may be broken, but it compiles. Maybe I will create full example of CRUD with htmx later.
- I'm not sure I quite like templ and tailwindcss, but it's really popular tools, so I tried them. In production app I used stdlib [Go template](https://pkg.go.dev/html/template) + Bootstrap 5.3 version. I kept templ + tailwindcss version as a separate app. Also I want to try PicoCSS, like lightweight Bootstrap.
- It's recommended to use htmx for SPA-like block updates and Alpinejs fro interactivity. In production app this combination behaves very well.
- Frontend libs and templates are separate for each app. This repo also shows concept how to build multiple apps with shared code base but with different purpose and UIs.
- All code which are needed for build is vendored into repo, js and css libs too. CLI tools are not vendored, see `./bin-deps.mk`.
- Each app builds as a single binary, assets (templates, css, js, migrations) are baked inside.
- Includes DB tools like migrations, template for writing storage tests and etc.
- CLI part uses Cobra, heavy framework but I like it. You will need this management commans in real project anyway.
- Web part uses modern Go 1.22 stdlib. Codebase is very tiny, without any frameworks.

Now Go and create your own full-stack framework. Or use Django.
