# Quickstart

Create a Toraapi API key and send a first OpenAI-compatible request.

## 1. Create an API key

Sign in to the [ToraApi console](https://toraapi.com), open [API Keys](https://toraapi.com/keys), and create a key. Choose a non-default group that lists the model you want to call.

Keep the key ready. Do not put it in client-side code or a public repository.

## 2. Send a first request

The OpenAI-compatible Chat Completions endpoint is:

`POST https://toraapi.com/v1/chat/completions`

  **cURL**

```bash
curl https://toraapi.com/v1/chat/completions \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-5.6-sol",
    "messages": [
      {"role": "user", "content": "Hello"}
    ]
  }'
```

  
  **Python**

```python
from openai import OpenAI

client = OpenAI(
    api_key="YOUR_API_KEY",
    base_url="https://toraapi.com/v1",
)

response = client.chat.completions.create(
    model="gpt-5.6-sol",
    messages=[{"role": "user", "content": "Hello"}],
)

print(response.choices[0].message.content)
```

  
  **Node.js**

```javascript

const client = new OpenAI({
  apiKey: "YOUR_API_KEY",
  baseURL: "https://toraapi.com/v1",
});

const response = await client.chat.completions.create({
  model: "gpt-5.6-sol",
  messages: [{ role: "user", content: "Hello" }],
});

console.log(response.choices[0].message.content);
```

  

Replace `gpt-5.6-sol` with a model your key can access. The live catalog is on [Models & Pricing](https://toraapi.com/pricing).

## 3. Pick the right surface next

- Coding agents: [Codex](/guides/clients/codex/), [Claude Code](/guides/clients/claude-code/), [Grok Build](/guides/clients/grok-build/)
- Image models: [Image models](/models/image/)
- Video models: [Video models](/models/video/)
- Agent workflow: [Media Skill](/guides/clients/media-skill/)
- Which family to call: [Which endpoint to call](/guides/endpoints/)