# Errors

Read HTTP status codes and the JSON error object returned by Modelflare.

Failed requests return JSON with an `error` object. The HTTP status is the first signal; the `message` and `code` fields name the actual failure.

```json
{
  "error": {
    "message": "Invalid API key provided.",
    "type": "authentication_error",
    "code": "invalid_api_key"
  }
}
```

## Common status codes

| Status | Typical cause |
| --- | --- |
| `400` | Invalid JSON, unsupported field, or wrong endpoint for the model |
| `401` | Missing or invalid API key |
| `403` | The key cannot use this model or group |
| `404` | Unknown path or missing task |
| `429` | Rate limit |
| `499` | Client cancelled the request |
| `5xx` | Service provider or gateway failure |

Image and video requests sent to the wrong protocol return `400` with `unsupported_endpoint` and name the required API. Silent protocol conversion is only for text models.

Longer diagnosis, retries, and fallback belong in the technical articles on the main site, starting with [AI API error troubleshooting](https://toraapi.com/articles/ai-api-error-troubleshooting).