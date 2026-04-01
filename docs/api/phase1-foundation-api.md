# Phase1 Foundation API

## Base URL

`/api/v1`

## Authentication

All endpoints (except `/wechat/login` and `/wechat/bind`) require a JWT token in the `Authorization` header:

```
Authorization: Bearer <token>
```

Token payload contains: `sub` (user ID), `role`, `class_id`, `grade`.

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
Authorization: Bearer eyJhbGciOiJIUzI1NiIs...
```
