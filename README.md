# `pigeonhole`

![Photograph of some physical pigeon holes, with mail in](pigeonhole.jpg)

`pigeonhole` is a small webservice that allows messages to be stored in
named cubbies, and later retrieved. The primary motivation is for cross-device
communication, e.g. a smart watch submitting a bit of data that can then
be picked up by a desktop computer when it's turned on or when a certain
apps is launched.

## Configuration

`pigeonhole` accepts configuration via command-line flags or environment
variables.

### `--tokens` / `TOKENS`

A list of bearer tokens authorised for use with `pigeonhole`. Tokens can
be any string of characters, but may not include `:` or `;`. Each token
can grant access to a single named cubby, or use the wildcard `*` to
allow access to everything. If a token can access a cubby, it can read
from it, write to it, and delete it.

Format: `token:cubby;token:cubby;token:cubby`, e.g.: `my-secret-token:reminders;another-secret-token:shower-thoughts;admin-token:*`

### `--db` / `DB`

The path to store the `pigeonhole` database on disk. If not specified,
defaults to `pigeonhole.db` in the current directory.

### `--listen` / `LISTEN`

The address to listen for HTTP connections on. If not specified,
defaults to `:8080`.

## Running

`pigeonhole` is designed to run in Docker:

```docker-compose
services:
  pigeon:
    image: ghcr.io/csmith/pigeonhole:dev
    restart: always
    environment:
      DB: "/data/pigeonhole.db"
      TOKENS: "some-token-here:cubby1;some-other-token:cubby2;etc:etc"
      LISTEN: ":8080"
    volumes:
      - data:/data

volumes:
  data:
```

You should run `pigeonhole` behind a TLS-terminating reverse proxy like
[Centauri](https://github.com/csmith/centauri), 
[Caddy](https://caddyserver.com/), etc. Alternatively you could expose
it over a secure VPN e.g. by using [thp](https://github.com/greboid/thp)
to connect it to Tailscale. Either way, don't expose plain HTTP to the
Internet!

## API

`pigeonhole` has a very simple API design:

### `POST /:cubby`

Adds a new message to the named cubby. Data can be sent as a standard or
multipart form, JSON, or just as plain text. For forms and JSON the
following fields/keys are checked: `message`, `text`, `content`.

e.g.:

```http
POST /reminders HTTP/1.1
Authorization: Bearer my-secret-token
Content-type: application/json

{"message": "Clean out the coop at 4pm"}
```

```http
POST /shower-thoughts HTTP/1.1
Authorization: Bearer another-secret-token
Content-type: text/plain

Apartment blocks are pigeon holes for people
```

```http
POST /pigeon-name-ideas HTTP/1.1
Authorization: Bearer admin-token
Content-type: application/x-www-form-urlencoded

content=Cher+Ami
```

### `GET /:cubby`

Returns all the messages in the cubby, along with the timestamps
they were added at.

e.g.:

```http
GET /reminders HTTP/1.1
Authorization: Bearer my-secret-token
```

returns:

```
HTTP/1.1 200 OK
Content-Type: application/json

[{"time":"2026-01-30T00:09:48.780173106Z","message":"Clean out the coop at 4pm"}]
```

### `DELETE /:cubby`

Deletes all messages in the cubby.

```http
DELETE /reminders HTTP/1.1
Authorization: Bearer my-secret-token
```

### `DELETE /:cubby?notafter=:time`

Deletes all messages in the cubby that were received before or at the given
RFC3339 timestamp. This allows you to delete the messages retrieved with a
`GET`, preserving any that happened to arrive since.

```http
DELETE /reminders?notafter=2026-01-30T00:09:48.780173106Z
Authorization: Bearer my-secret-token
```

---

Pigeon hole image: “<a href="https://www.flickr.com/photos/addedentry/3273096118" title="Dictionary pigeonholes">Dictionary pigeonholes</a>” by <a href="https://www.flickr.com/photos/addedentry/">Owen Massey McKnight</a>, <a href="https://creativecommons.org/licenses/by-sa/2.0/deed.en" rel="license noopener noreferrer">CC BY-SA 2.0</a>
