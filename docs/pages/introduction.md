# Introduction

OpenAI-compatible Chat and Responses, Anthropic Messages, and 14 image plus 27 video models on one ToraA API key, with setup guides for Codex, Claude Code and other coding agents.

ToraApi is a unified API gateway for text, image, and video models. One key covers OpenAI-compatible chat plus the native contracts that image and video providers still require. Console, keys, and live prices stay on the [main site](https://toraapi.com).

## Get started

<StartGrid
  actions={[
    { href: '/quickstart/', label: 'Quickstart', primary: true },
    { href: '/guides/endpoints/', label: 'Which endpoint' },
  ]}
>
  <StartStep number="01" icon="key" title="Create an API key" description="Sign in to the console and create a key on a group that lists your model." />
  <StartStep number="02" icon="terminal" title="Send a request" description="Point an OpenAI SDK or curl at https://toraapi.com/v1 with the key." />
  <StartStep number="03" icon="layers" title="Pick a surface" description="Chat, Responses, Anthropic Messages, Images or Video, all on the same key." />
</StartGrid>

## What ToraApi provides

  
    One key for text, image, and video
    Chat Completions, Responses, Anthropic Messages, Images, and Video share the same authentication and routing groups.
  
  
    OpenAI-compatible chat
    If you already use the OpenAI SDK, switch the base URL and keep your client code.
  
  
    Native contracts where they matter
    GPT Images, Gemini native image, and async video families keep their own request fields. The docs do not flatten them into one fake schema.
  
  
    Live prices on the main site
    Availability, groups, and billing always come from <a href="https://toraapi.com/pricing">Models &amp; Pricing</a>. This site does not keep a second price list.
  

## Explore the docs

  - [Coding agents](/guides/clients/codex/): Codex, Claude Code, OpenCode, Grok Build, Gemini CLI, and CC-Switch setup.
  - [Media Skill](/guides/clients/media-skill/): Let an agent choose, price, call, and save image or video generations.
  - [Platform features](/guides/features/): Smart API Keys, group failover, the cache hit guarantee, and data retention.
  - [Models](/models/): Text, image, and video models with the exact request contract for each.
  - [Which endpoint to call](/guides/endpoints/): Hostnames, auth, and which family each request uses.