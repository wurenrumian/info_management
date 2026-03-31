# Phase1 Foundation API

## Base URL

`/api/v1`

## Required Headers (Phase 1)

- `X-User-Id`
- `X-User-Role`
- `X-User-Class-Id` (required for cadre/teacher scope behavior)
- `X-User-Grade` (required for teacher grade scope behavior)

## Student Endpoint

- `GET /api/v1/me`

## Admin User Endpoints

- `GET /api/v1/admin/users`
- `GET /api/v1/admin/users/:id`
- `PATCH /api/v1/admin/users/:id`

## Admin Class Endpoints

- `GET /api/v1/admin/classes`
- `GET /api/v1/admin/classes/:id`
- `POST /api/v1/admin/classes`
- `PATCH /api/v1/admin/classes/:id`

## Admin Logs Endpoint

- `GET /api/v1/admin/logs`

## Example Request

```http
GET /api/v1/admin/users HTTP/1.1
X-User-Id: 300
X-User-Role: 3
X-User-Class-Id: 1
X-User-Grade: 2023
```

