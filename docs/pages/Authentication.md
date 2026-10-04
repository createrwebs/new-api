# Authentication

Use a ToraApi API key as a Bearer token on every request.

ToraApi authenticates API requests with a Bearer token.

`Authorization: Bearer YOUR_ToraApi_API_KEY`

Create and rotate keys in the [API Keys](https://ToraApi.com/keys) console. A key is bound to a routing group; the group determines which models the key can call.

## Rules

- Send the header on every request, including task polling and media downloads.
- Do not put the key in a browser, mobile app, or public repository.
- Client setup guides write the key into a local credentials file. Those files should stay user-private (`chmod 600` on Unix).
- Image and video models still use the same key. They do not use a second auth scheme.

If a request returns `401`, the key is missing, malformed, or revoked. If it returns `403`, the key is valid but the selected model or group is not allowed. See [Errors](/guides/errors/).