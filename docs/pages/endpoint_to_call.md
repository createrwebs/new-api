# Which endpoint to call

Base URL, host choice, and which API family to send each request to.

ToraApi uses one API key across several protocols. The JSON envelope is not the same for every model. Pick the family first, then copy the request from the matching guide or model page.

## Base URL

`https://ToraApi.com/v1`

China-facing EdgeOne traffic can use `https://origin.ToraApi.com/v1` for the same application. Choose the host before sending a request; do not redirect an in-flight POST between hosts.

Anthropic clients are the exception: `ANTHROPIC_BASE_URL` is `https://ToraApi.com` (the site origin, not `/v1`). The official Claude Code CLI appends the Messages path itself. See [Claude Code](/guides/clients/claude-code/).

## Authentication

`Authorization: Bearer YOUR_ToraApi_API_KEY`

Send this header on every request, including video polling and downloads. Create the key in the [console](https://ToraApi.com/keys). Details: [Authentication](/guides/authentication/).

## Families

**One ToraApi API key**

One key and one host, but a different request contract per family. The table below links to the guide or model page for each.

- **Chat Completions** `POST /v1/chat/completions`: OpenAI-compatible chat.
- **Responses** `POST /v1/responses`: Codex and Grok Build.
- **Anthropic Messages** `ANTHROPIC_BASE_URL=https://ToraApi.com`: The Claude Code CLI appends the Messages path itself.
- **Images** `POST /v1/images/generations`: GPT Images, Wan Image, and Seedream.
- **Gemini native** `POST /v1beta/models/{model}:generateContent`: Gemini CLI and Gemini image models.
- **Video** `POST /v1/videos`: Creates a task: poll it, then download the result.
- **Seedance assets** `POST /volc/asset/*`: Manage Seedance asset groups and provider-side media IDs.
- **Models** `GET /v1/models`: What the current key can call.

| Family | Typical entry | Use it for |
| --- | --- | --- |
| Chat Completions | `POST /v1/chat/completions` | OpenAI-compatible chat. First request: [Quickstart](/quickstart/). |
| Responses | `POST /v1/responses` | [Codex](/guides/clients/codex/) and [Grok Build](/guides/clients/grok-build/). |
| Anthropic Messages | Anthropic-compatible Messages | [Claude Code](/guides/clients/claude-code/). |
| Images | `POST /v1/images/generations` | GPT Images, Wan Image, and Seedream. Exact fields: [image models](/models/image/). |
| Gemini native | `POST /v1beta/models/{model}:generateContent` | Gemini CLI and Gemini image models. |
| Video | `POST /v1/videos` | Seedance, Wan, Kling, and other async video models. Poll `GET /v1/videos/{task_id}` and download `GET /v1/videos/{task_id}/content`. Exact envelope: [video models](/models/video/). |
| Seedance assets | `POST /volc/asset/*` | Create, query, update, and delete provider-side asset groups and media. See [Seedance asset management](/guides/video/seedance-asset-management/). |
| Models | `GET /v1/models` | List models the current key can call. |

Wrong-endpoint image and video requests are rejected with `unsupported_endpoint`. Do not send Gemini image fields to `/v1/images/*` or GPT Images fields to `generateContent`. Do not mix Chat Completions fields into a Responses or Anthropic client.

Use a `media` group API key for image and video. Result URLs can expire; transfer the file after a video task succeeds.

Seedance asset-management calls also require a `media` group API key. Asset creation is asynchronous; use `GetAsset` until `Active`, then pass the returned `asset://<ID>` unchanged in a supported Seedance request. The gateway keeps asset ownership per user and does not add a separate resource-management charge.

## List models for this key

`GET https://ToraApi.com/v1/models`

```bash
curl https://ToraApi.com/v1/models \
  -H "Authorization: Bearer YOUR_ToraApi_API_KEY"
```

The response lists models the current key can call. It is not a substitute for [Models & Pricing](https://ToraApi.com/pricing), which includes prices, groups, and protocol labels. Use the exact `id` in the JSON `model` field. See [Models and groups](/guides/models-and-groups/).