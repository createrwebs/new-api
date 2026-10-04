# Models and groups

Choose an exact model ID and a routing group that can serve it.

Two identifiers matter on every request:

1. The **model ID** in the JSON body, which must match a public catalog ID exactly.
2. The **routing group** bound to your API key, which decides which service routes can serve that model.

Both identifiers have to agree: a valid JSON body still fails when the key's group has no channel for that model.

- **Your request**: The model ID in the JSON body must match a public catalog ID exactly.
- **API key**: Names one primary group and, optionally, fallback groups in priority order.
- **Routing group**: A set of service provider channels. If none of them serves that model, the request fails.
- **Service provider channel**: The group tries its own channel priorities in order before a fallback group is used.

The live catalog is [Models & Pricing](https://ToraApi.com/pricing). This documentation site does not override prices or availability.

## Exact model IDs

Use the ID shown on the pricing page or on the model catalog page. Do not substitute a provider marketing name, a date suffix, or a nearby variant.

Chat models that share the OpenAI-compatible Chat Completions or Responses contract are documented on [Which endpoint to call](/guides/endpoints/) and the client guides, not by duplicating one page per model. Image and video models that have their own request envelope have [model pages](/models/) and catalog pages.

## Groups

Create the API key in a non-default group that lists the model. There is no single required group. If the key’s group cannot serve the model, the request fails even when the JSON is valid.

Fallback order is a property of the group, not of the request body. See [Usage and cost](/guides/usage-and-cost/) for how charges are recorded.