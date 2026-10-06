# TORA AI — ORGANIC SOURCE QUALITY SCORECARD
## CONTINUOUS HEALTH & EDITORIAL PERFORMANCE METRICS ACROSS 24 DISCOVERY SOURCES

> **Domain Target**: `https://www.toraapi.com`  
> **Evaluation Period**: 30-Day Growth Baseline  
> **Source Registry Size**: 24 Verified Technology & AI Feeds  
> **Status**: All 24 Sources Active & 100% Healthy (0 Consecutive Failures)

---

### Source Quality Classification Framework

Per Section 6 & 48 of the Organic Growth Protocol, discovery sources are evaluated weekly and assigned dynamic editorial weights:
- **`HIGH_VALUE` (Score Multiplier: 1.25x)**: Official frontier model labs, research institutes, and key gateway technologies that consistently yield high-depth, primary-source engineering stories.
- **`USEFUL` (Score Multiplier: 1.0x)**: Cloud vendor blogs, ecosystem tools, and changelogs providing steady, verifiable updates.
- **`NOISY` (Score Multiplier: 0.75x)**: High-frequency feeds that produce marketing fluff, redundant recaps, or low-density PR announcements. Requires strict deduplication filtering.
- **`LOW_VALUE` (Score Multiplier: 0.5x)**: Low-yield sources requiring multiple manual review gates.
- **`DEGRADED` (Auto-Quarantine)**: Feeds with consecutive HTTP or XML parsing errors.

---

### The 24 Source Master Scorecard

| ID | Source Name | Feed Endpoint | Type | HTTP Status | Failures | Parse Status | Score Tier | Editorial Focus |
| :-: | :--- | :--- | :---: | :---: | :---: | :---: | :---: | :--- |
| 1 | **OpenAI Newsroom** | `https://openai.com/news/rss.xml` | RSS | 200 | 0 | OK | **HIGH_VALUE** | Frontier model releases, safety updates, API enhancements |
| 2 | **Anthropic Research & News**| `https://www.anthropic.com/feed` | RSS | 200 | 0 | OK | **HIGH_VALUE** | Claude models, constitutional AI, prompt caching, reasoning |
| 3 | **Google AI Blog** | `https://blog.google/technology/ai/rss/` | RSS | 200 | 0 | OK | **HIGH_VALUE** | Gemini models, Transformer architecture, TPUs |
| 4 | **DeepSeek Releases** | `https://github.com/deepseek-ai/DeepSeek-V3/releases.atom` | Atom | 200 | 0 | OK | **HIGH_VALUE** | Open-source reasoning models, MLA attention, cost efficiency |
| 5 | **OpenRouter Announcements** | `https://openrouter.ai/changelog.atom` | Atom | 200 | 0 | OK | **HIGH_VALUE** | Multi-provider routing, model pricing, token discounts |
| 6 | **New-API Core Releases** | `https://github.com/QuantumNous/new-api/releases.atom` | Atom | 200 | 0 | OK | **USEFUL** | Gateway core updates, upstream bugfixes, channel protocols |
| 7 | **Google DeepMind** | `https://deepmind.google/blog/rss.xml` | RSS | 200 | 0 | OK | **HIGH_VALUE** | Frontier algorithms, AlphaFold, agentic decision making |
| 8 | **Google Research** | `https://research.google/blog/rss/` | RSS | 200 | 0 | OK | **HIGH_VALUE** | Foundational computer science, quantization, embeddings |
| 9 | **Google Developers Blog** | `https://developers.googleblog.com/feeds/posts/default` | RSS | 200 | 0 | OK | **USEFUL** | Android, Web, Firebase, Cloud SDK integrations |
| 10 | **AWS Machine Learning Blog** | `https://aws.amazon.com/blogs/machine-learning/feed/` | RSS | 200 | 0 | OK | **USEFUL** | Bedrock, SageMaker, GPU clustering, enterprise LLMs |
| 11 | **NVIDIA Technical Blog** | `https://developer.nvidia.com/blog/feed` | RSS | 200 | 0 | OK | **HIGH_VALUE** | CUDA, TensorRT-LLM, Blackwell architecture, inference |
| 12 | **Apple ML Research** | `https://machinelearning.apple.com/feed.xml` | RSS | 200 | 0 | OK | **USEFUL** | On-device inference, CoreML, small language models |
| 13 | **Microsoft Official Blog - AI** | `https://blogs.microsoft.com/ai/feed/` | RSS | 200 | 0 | OK | **USEFUL** | Azure OpenAI, Copilot ecosystem, enterprise AI |
| 14 | **Hugging Face Blog** | `https://huggingface.co/blog/feed.xml` | RSS | 200 | 0 | OK | **HIGH_VALUE** | Open weights, Transformers library, TGI, leaderboard |
| 15 | **MIT News - AI** | `https://news.mit.edu/rss/topic/artificial-intelligence` | RSS | 200 | 0 | OK | **USEFUL** | Academic breakthroughs, robotics, neural architecture |
| 16 | **Berkeley AI Research (BAIR)** | `https://bair.berkeley.edu/blog/feed.xml` | Atom | 200 | 0 | OK | **HIGH_VALUE** | Reinforcement learning, vLLM, speculative decoding |
| 17 | **GitHub Blog** | `https://github.blog/feed/` | RSS | 200 | 0 | OK | **USEFUL** | Developer workflows, open source trends |
| 18 | **GitHub Changelog** | `https://github.blog/changelog/feed/` | RSS | 200 | 0 | OK | **USEFUL** | Platform features, security advisories, API changes |
| 19 | **GitHub Copilot Changelog** | `https://github.blog/changelog/label/copilot/feed/` | RSS | 200 | 0 | OK | **HIGH_VALUE** | AI-assisted programming, agent mode, IDE tooling |
| 20 | **Cloudflare Blog** | `https://blog.cloudflare.com/rss/` | RSS | 200 | 0 | OK | **USEFUL** | Edge AI, Workers AI, Vectorize, web performance |
| 21 | **Cohere Blog & Changelog** | `https://docs.cohere.com/changelog.atom` | Atom | 200 | 0 | OK | **HIGH_VALUE** | Embeddings, RAG, Command models, enterprise search |
| 22 | **Replicate Blog** | `https://replicate.com/blog/rss` | RSS | 200 | 0 | OK | **USEFUL** | Serverless open-source model hosting, diffusion APIs |
| 23 | **Amazon Science** | `https://www.amazon.science/index.rss` | RSS | 200 | 0 | OK | **USEFUL** | Scientific ML, distributed training, speech models |
| 24 | **Carnegie Mellon ML Blog** | `https://blog.ml.cmu.edu/feed/` | RSS | 200 | 0 | OK | **USEFUL** | Frontier systems research, model pruning, alignment |

---

### Weekly Dynamic Re-weighting Policy

1. **High Quality Yield Promotion**: When a source produces $\ge 3$ accepted articles per week with zero quality rejections and $\ge 1$ evergreen proposal, its editorial weight is automatically boosted to `HIGH_VALUE`.
2. **Repetition Penalty**: If a source triggers $\ge 5$ duplicate rejections in a 7-day rolling window, its scout priority is temporarily decreased to prevent pipeline noise.
3. **Failure Isolation**: Any source experiencing $\ge 3$ consecutive HTTP 5xx or timeout errors will trigger an administrative health review without blocking other feeds.
