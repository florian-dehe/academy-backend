# Remojii Academy API

Go modular monolith: one API (`cmd/api`) plus a worker (`cmd/worker`) on Postgres.

## Quick start

```sh
make up           # Postgres via docker compose
make run          # API on :8080; .env sets APP_ENV=dev (enables learner-token / editor-token / admin-token, seeded as users) and AUTO_MIGRATE=true (GORM AutoMigrate at startup)
make test
```

```sh
curl localhost:8080/healthz
curl -H 'Authorization: Bearer admin-token' -X POST localhost:8080/v1/chapters   # 501 until implemented
```

`APP_ENV` (dev|staging|prod) is required, with no default, so a missing value can never enable dev auth. `AUTO_MIGRATE` defaults to false.
Request logs carry `user_id` and `role`: admin writes have no audit table, so these are the record.

## Layout

```
cmd/api, cmd/worker
api/              openapi.yaml, embedded; served at /openapi.yaml, Swagger UI at /docs
internal/
  app/            router wiring + permission-matrix test
  platform/       config, db (GORM, WithTx), events (bus), httpx (errors, middleware)
  identity/       roles, auth middleware, TokenResolver   <- real
  entitlement/    CanAccessChapter + RequireChapter 402 hook (AllowAll stub until P6)
  content/ learning/ gamification/ billing/ introtest/ platformadmin/
                  http.go = every route from endpoints.md with role guards, returning 501
  schema/         GORM AutoMigrate over every module's models + foreign keys + seed rows (levels, plans)
```

Module rules: a module touches only its own tables; cross-module calls go
through Go interfaces; side effects use `platform/events`; handlers stay thin.

## Status

Roadmap P0 scaffold. Done: config, errors (401/402/403/409), auth + role guards, schema (AutoMigrate),
route table with guards, permission tests, GORM example (`identity/store`).
Next: P1 accounts (needs the open decision on the deviceToken / Job Board resolver, see
`identity.UnconfiguredResolver`), then P2 content.

## Not yet added (deliberately)

River (queue), stripe-go, oapi-codegen, validator, jsonschema, otter, sentry, prometheus. Add them in the phase that needs them.
