# Phase 7A — LumenFlow Mobile Architecture & Backend Integration Audit

**Execution Date**: 2026-10-05  
**Primary Target Codebase**: LumenFlow (`/Users/noppanan/.gemini/antigravity/scratch/LumenFlow`, commit `1c0cbf90e6ff05118747a3297a77e8be202e2fa9`)  
**Reference Codebase**: Oriveo (`/Users/noppanan/.gemini/antigravity/scratch/oriveo`, reference-only, AGPL-3.0-or-later)  
**Backend Integration Baseline**: SaaSCover / New-API (`/Users/noppanan/new-api`, branch `feat/formobile`, commit `8d68eaa8b5dda6c647d1e88278d1f667a8e0d0e5`)  
**Scope**: AUDIT & DISCOVERY ONLY (Zero application code modifications)  

---

## 1. Executive Summary

This report establishes the architectural baseline for **Phase 7A — Mobile Product Integration**, connecting the **LumenFlow** mobile application to the hardened **SaaSCover / Tora New-API** backend.

Following the completion and passing of the Phase 6H Production Release Gate on the backend, this audit inspects the primary mobile application repository (`LumenFlow`, a cross-platform Flutter application licensed under MIT) and references `Oriveo` (AGPL-3.0-or-later, architecture reference only, no code reuse). 

### Key Audit Findings

1. **Architecture & Framework**: LumenFlow is an elegant, modular **Flutter 3.x / Dart 3.11** application using Cupertino iOS-native styling, Material design support, and desktop windowing. It implements a clean Provider abstraction pattern for LLM services, local SQLite caching, and granular settings management.
2. **Current Egress Model (Pure Client-Side BYOK)**: LumenFlow currently operates as a 100% local, client-side application. The mobile app communicates **directly with third-party LLM providers** (OpenAI, Anthropic, Google Gemini, DeepSeek, Moonshot/Kimi, SiliconFlow, ZhiPu, MiniMax, Grok, OpenRouter, LM-Studio). It has **zero backend accounts, zero remote authentication, and zero reliance on a remote application server**.
3. **OpenAI Protocol Compatibility**: LumenFlow’s `OtherProvider` and `SiliconFlowProvider` implementations already utilize standard OpenAI-compatible endpoints (`/chat/completions` and `/v1/models`) with full support for Server-Sent Events (SSE) streaming, reasoning deltas (`reasoning_content`), multimodal image Data URLs, and exponential-backoff retries. This aligns directly with New-API’s `/v1/chat/completions` relay engine.
4. **Discrepancy in `OpenAIProvider`**: The built-in `OpenAIProvider` targets OpenAI’s experimental `/responses` endpoint rather than `/chat/completions`. However, `OtherProvider` cleanly targets `/chat/completions` and serves as the ideal template for the Tora/New-API integration client.
5. **Security Gaps (Mobile)**: API keys and platform configurations are currently persisted in **unencrypted `SharedPreferences`** (plain XML/plist on mobile devices). Neither iOS Keychain nor Android Keystore is utilized.
6. **Integration Strategy (No BFF Required)**: Because the New-API backend (`feat/formobile`) already exposes OpenAI-compatible `/v1/chat/completions`, authenticated `/api/user_provider` (encrypted server-side BYOK), `/v1/models`, and user authentication/subscription endpoints, **no separate Backend-for-Frontend (BFF) is necessary**. Mobile can directly target New-API via a dedicated `ToraProvider` client.

---

## 2. Repository Instructions

- **`AGENTS.md` Status**: Checked root and subdirectories of `/Users/noppanan/.gemini/antigravity/scratch/LumenFlow`. **No `AGENTS.md` exists** in LumenFlow.
- **Repository Documentation**:
  - `README.md`: Documents features, multi-platform support (Android, iOS, macOS, Windows, Linux), model management, and prompt preset importing.
  - `LumenFlowFormatSpecification.md`: Defines the versioned `.lumenflow` JSON settings export/import schema (format version `1.0`).
  - `CONTRIBUTING.md` / `LICENSE`: Governed under the permissive **MIT License**.
- **Reference Repository (`Oriveo`) Note**: Oriveo is licensed under **AGPL-3.0-or-later**. In compliance with project legal rules, **zero code or assets from Oriveo will be merged or copied into LumenFlow**. Oriveo was evaluated strictly as a reference design for Android/provider streaming and tool-calling models.

---

## 3. Technology Stack

| Layer | Component | Version / Specification | Rationale / Detail |
| :--- | :--- | :--- | :--- |
| **Language** | Dart | `^3.11.1` (`pubspec.yaml`) | Strict typing, null safety |
| **UI Framework** | Flutter SDK | Flutter 3.x (Material + Cupertino) | Native cross-platform rendering |
| **Design Language** | Cupertino | `CupertinoApp`, `CupertinoThemeData` | iOS-first design aesthetic |
| **State Management** | `StatefulWidget` / `ValueNotifier` | Local state + listeners | Clean, minimal dependencies; no heavy Redux/Bloc |
| **Networking** | `package:http` | `^1.6.0` | Standard Dart HTTP client with custom streaming |
| **Local Database** | `sqlite3` | `^3.2.0` | High-performance C-level SQLite via Dart FFI |
| **Key-Value Store** | `shared_preferences` | `^2.5.4` | App preferences, platform configs, API keys |
| **Markdown** | `flutter_markdown_plus` | `^1.0.7` | Renders assistant responses with LaTeX / syntax |
| **Media / Files** | `image_picker` / `file_picker` | `^1.2.1` / `^10.3.10` | Multimodal attachments (images, audio, PDF) |
| **Localization** | `flutter_localizations`, `intl` | `intl: any` | Multi-language (en, zh, ja, ko, lzh, es) |
| **Platform Channels**| `me.huanmeng.lumenflow/live_update` | MethodChannel | Android 16 Live Update notifications |
| **Embedded Server** | `dart:io` `HttpServer` | Port 5050 | Local offline prompt generator web UI |

---

## 4. Repository Architecture

The codebase adheres to a modular, service-oriented structure:

```text
lib/
├── l10n/                               # Localization files and ARB translations
│   ├── app_en.arb, app_zh.arb, ...
│   └── app_localizations.dart
├── models/                             # Core entity schemas
│   ├── ai_platform.dart                # Provider/platform configuration schema
│   ├── attachment.dart                 # Multimodal file metadata & mime mapping
│   ├── conversation.dart               # Conversation header & message aggregation
│   ├── message.dart                    # Chat message entity & reasoning content
│   ├── prompt_preset.dart              # Role-play & system prompt presets
│   └── user_profile.dart               # Local user profile model
├── providers/                          # LLM provider network adapters
│   ├── ai_provider.dart                # Base abstract interface (sendMessage, streaming)
│   ├── http_provider_base.dart         # Base HTTP client with retry & error parsing
│   ├── openai_provider.dart            # OpenAI /responses adapter
│   ├── other_provider.dart             # Standard OpenAI /chat/completions adapter
│   ├── deepseek_provider.dart          # DeepSeek chat completions adapter
│   ├── siliconflow_provider.dart       # SiliconFlow chat completions adapter
│   ├── claude_provider.dart            # Anthropic /messages adapter
│   ├── gemini_provider.dart            # Google Generative AI adapter
│   └── ...                             # Kimi, ZhiPu, MiniMax, Grok, OpenRouter, LM-Studio
├── screens/                            # User Interface screens
│   ├── chat_screen.dart                # Primary interactive chat workspace
│   ├── conversation_list_screen.dart   # Conversation drawer & management
│   ├── platform_settings_screen.dart   # Multi-platform management & model fetcher
│   ├── api_settings_screen.dart        # Legacy API endpoint & key configuration
│   ├── model_settings_screen.dart      # Temperature, tokens, model selector
│   ├── settings_screen.dart            # Hub for all settings
│   └── user_profile_screen.dart        # Local profile customization
├── services/                           # Application business logic
│   ├── ai_service.dart                 # Provider factory, system prompt compiler
│   ├── conversation_database.dart      # SQLite schema, indexes & transactions
│   ├── conversation_service.dart       # Conversation memory caching & CRUD
│   ├── file_service.dart               # Local file I/O & data URL conversion
│   ├── live_update_service.dart        # Android Live Update notification channel
│   ├── notification_service.dart       # Local completion alerts
│   ├── prompt_service.dart             # Preset loading from assets & user files
│   ├── settings_service.dart           # SharedPreferences access layer
│   └── user_service.dart               # Local user profile persistence
├── utils/                              # Theming, paths, time utilities
│   ├── app_theme.dart                  # Brightness, Cupertino themes
│   └── path_utils.dart                 # Application sandbox directory paths
└── widgets/                            # Reusable UI widgets
    ├── chat_input.dart                 # Multi-line input bar with attachment tray
    ├── message_bubble.dart             # Chat bubble with reasoning accordion
    └── settings/...                    # Reusable settings tiles
```

---

## 5. Application Startup Flow

```text
lib/main.dart: main()
    ↓
WidgetsFlutterBinding.ensureInitialized()
    ↓
NotificationService.initialize()
    ↓
Platform-specific Windowing / Orientation (Windows/Linux windowManager, Android portrait lock)
    ↓
LiveUpdateService.initialize()
    ↓
SettingsService: Theme resolution (followSystemTheme vs stored appTheme)
    ↓
runApp(MyApp)
    ↓
MyApp: initState() -> observer registered, locale loaded from SettingsService
    ↓
CupertinoApp: home -> ChatScreen()
    ↓
ChatScreen: initState()
    ├─ _checkConfiguration() -> checks if any platform has apiKey
    └─ _loadCurrentConversation() -> loads active conversation from SQLite
```

- **Authentication Screening**: There is **no splash gate, login check, or route guard**.
- **First Launch Experience**: If unconfigured, the app opens directly to `ChatScreen`. When the user attempts to send a prompt, a dialog prompts them to configure an API key in settings.

---

## 6. Authentication Architecture

- **Existing Auth Mechanisms**:
  - Anonymous usage: Yes (pure local mode).
  - Username/password: **None**.
  - OAuth / SSO: **None**.
  - JWT / Session Cookies: **None**.
  - Mobile App API Tokens: **None**.
  - Provider API Keys: Stored locally per platform.
- **Credential Storage**: Stored as plaintext string values in `SharedPreferences` under key `api_key` or serialized within `ai_platforms` JSON.
- **Can the existing auth client be adapted?**  
  There is **no existing authentication client** to adapt. An authentication service (`AuthService`) and user session state must be created from scratch in LumenFlow to authenticate against New-API’s `/api/user/login` and manage personal access tokens (`sk-...`).

---

## 7. API Client Architecture

- **Network Client**: Standard `http.Client` from `package:http`.
- **Abstraction Hierarchy**:
  ```text
  UI (ChatScreen)
      ↓
  Service (AIService)
      ↓
  Provider Interface (AIProvider)
      ↓
  Provider Base (HttpProviderBase)
      ↓
  Platform Adapter (OtherProvider / ClaudeProvider / etc.)
      ↓
  HTTP Client (package:http Client)
      ↓
  Outbound Network (Direct to Provider API)
  ```
- **Error Handling & Retries**: `HttpProviderBase` encapsulates retry logic:
  - Max retries: 3.
  - Exponential backoff: `1000ms * (1 << retryCount)` capped at 10,000ms.
  - Retry triggers: `SocketException`, `TimeoutException`, `TlsException`, HTTP 429, HTTP 5xx.
- **Timeouts**: Connection timeout = 30s; Read timeout = 60s; Streaming timeout = 5m.

---

## 8. Chat Request & Execution Flow

```text
User taps Send in ChatInput
    ↓
ChatScreen: _handleSubmitted()
    ├─ Builds Message(isUser: true)
    ├─ Adds to _messages state list
    ├─ Generates initial conversation title (if message #1)
    ├─ Saves message to SQLite (ConversationDatabase)
    ├─ Appends placeholder Message(isUser: false, status: sending)
    └─ Starts Android Live Update notification (if enabled)
    ↓
ChatScreen calls AIService.sendMessageStreaming()
    ├─ Compiles system prompt (merging presets + user variables ${userProfile.username})
    ├─ Appends system timestamp (if addTimeToPrompt is enabled)
    ├─ Retrieves API key, temperature, maxTokens from SettingsService
    └─ Dispatches to selected AIProvider (e.g., OtherProvider)
    ↓
AIProvider: sendMessageStreaming()
    ├─ Compiles message history (up to historyContextLength * 2)
    ├─ Encodes attachments (Base64 data URLs for vision, extracted text for docs)
    ├─ Formats HTTP request body
    └─ Executes http.Client.send(request)
    ↓
HTTP Response Stream received (SSE)
    ├─ Chunk buffered in StringBuffer
    ├─ Delimited by "\n\n", parsed line-by-line ("data: ")
    ├─ Extracts delta.reasoning_content and delta.content
    └─ Yields Stream<Map<String, dynamic>>
    ↓
ChatScreen listens to Stream
    ├─ Batches setState() updates to UI (reasoning accordion + answer text)
    ├─ Updates Android Live Update notification
    └─ Debounces database persistence (_debouncedSaveConversation)
    ↓
Stream onDone:
    ├─ Updates Message.status = sent
    ├─ Finalizes database record
    └─ Triggers background title generation if needed
```

---

## 9. Streaming Architecture

- **Protocol**: HTTP/1.1 chunked transfer with Server-Sent Events (SSE).
- **Boundary Handling**: In `HttpProviderBase` / `OtherProvider`:
  ```dart
  // Handles split chunks across TCP packet boundaries
  sseBuffer.write(chunk);
  final events = bufferContent.split('\n\n');
  if (!bufferContent.endsWith('\n\n') && events.isNotEmpty) {
    final lastEvent = events.removeLast();
    sseBuffer.write(lastEvent);
  }
  ```
- **Parsing**:
  - Filters lines with `data: `.
  - Recognizes `[DONE]` termination token.
  - JSON-decodes payload and checks:
    - `delta['reasoning']` or `delta['reasoning_content']` -> emits `{type: 'reasoning', content: ...}`.
    - `delta['content']` -> emits `{type: 'answer', content: ...}`.
- **Cancellation**: Implemented via `StreamSubscription.cancel()` on user tap of the stop button, safely updating message status to `MessageStatus.stopped`.

---

## 10. Conversation Persistence

- **Database Engine**: Native SQLite via `package:sqlite3` (`sqlite3.open(dbPath)`).
- **Storage Location**: Managed by `PathUtils.getDatabasePath()` inside the application documents sandbox.
- **Schema**:
  - `conversations`: `id (TEXT PK), title (TEXT), created_at (INT), updated_at (INT)`
  - `messages`: `id (TEXT PK), conversation_id (TEXT FK), content (TEXT), reasoning_content (TEXT), is_user (INT), timestamp (INT), status (INT)`
  - `attachments`: `id (TEXT PK), message_id (TEXT FK), file_name (TEXT), file_path (TEXT), url (TEXT), type (INT), file_size (INT), mime_type (TEXT), created_at (INT)`
  - `settings`: `key (TEXT PK), value (TEXT)`
- **Foreign Keys**: Enforced via `PRAGMA foreign_keys = ON` with `ON DELETE CASCADE`.
- **Concurrency**: Guarded by an in-memory lock (`_synchronized<T>` via `Completer<void>`) to serialize writes on the main isolate.

---

## 11. Provider & Platform Abstraction

- **Entity Model**: `AIPlatform` (`lib/models/ai_platform.dart`):
  ```dart
  class AIPlatform {
    final String id;
    final String name;
    final String type;
    final String endpoint;
    final String apiKey;
    final List<String> availableModels;
    final String defaultModel;
    final bool enabled;
    final DateTime? lastModelUpdate;
    final String icon;
  }
  ```
- **Supported Platforms**: 13 built-in configurations: `openai`, `gemini`, `claude`, `grok`, `deepseek`, `kimi`, `zhipu`, `minimax`, `xiaomimimo`, `siliconflow`, `openrouter`, `lmstudio`, `other`.
- **Provider Registry**: `AIService._getProvider()` instantiates concrete subclasses of `AIProvider` based on the active platform type.

---

## 12. Model Architecture & Catalog

- **Catalog Origin**: Stored **locally** in `AIPlatform.availableModels`.
- **Dynamic Model Fetching**: Triggered manually in `PlatformSettingsScreen._refreshPlatformModels()`:
  - For OpenAI-compatible endpoints (`type == 'other'`, `siliconflow`, etc.):
    `GET $endpoint/v1/models` (or `$endpoint/models`) with `Authorization: Bearer <apiKey>`
  - Parses response: `{"data": [{"id": "model-id"}]}`.
- **Model Parameters**: Temperature (0.0 – 2.0, default 0.7), Max Tokens (default 8192), Thinking Mode toggle (adds `thinking_budget: 4096`).

---

## 13. Existing BYOK Architecture

- **Current Implementation**: Pure client-side BYOK.
- **Key Ingestion**: User enters API keys manually into text fields in `PlatformSettingsScreen` or `ApiSettingsScreen`.
- **Key Storage**: Plaintext in `SharedPreferences`.
- **Egress Path**: Directly from mobile device to provider clouds.
- **Backend Role**: None.

---

## 14. Secure Storage Audit

- **Audit Finding**: **CRITICAL DEFICIENCY**.
- **Findings**:
  - LumenFlow contains **no encrypted storage**.
  - `flutter_secure_storage` is not installed.
  - iOS Keychain is not touched.
  - Android EncryptedSharedPreferences / Keystore is not touched.
  - All keys, endpoints, and profiles exist in cleartext XML on Android and cleartext `.plist` on iOS/macOS.
- **Requirement for Tora Integration**: In Phase 7B/7C, integrate `flutter_secure_storage` to safeguard the Tora user session token and any transient credentials.

---

## 15. Settings Architecture

Settings are managed centrally through `SettingsService` and mapped to dedicated screens:
1. `PlatformSettingsScreen`: Manage multiple AI providers, endpoints, and fetch model lists.
2. `ModelSettingsScreen`: Select active model, adjust temperature and max tokens.
3. `ConversationSettingsScreen`: Adjust context length (default: 30 messages) and auto-title generation.
4. `AppearanceSettingsScreen`: Dark mode, follow system theme, custom chat background image.
5. `PresetManagementScreen`: Browse, import, and edit role-play system prompts.
6. `ToolManagementScreen`: Toggle automatic timestamp injection into prompts.
7. `AdvancedSettingsScreen`: Export/import all settings in `.lumenflow` JSON format.

---

## 16. Attachments & Multimodal Architecture

- **Supported Media**: Images (`image/*`), Audio (`audio/*`), Video (`video/*`), Documents (`pdf`, `doc`, `txt`, `md`).
- **Processing**:
  - Images: Converted to Base64 data URLs (`data:image/jpeg;base64,...`) and wrapped in standard OpenAI vision payloads:
    ```json
    {"type": "image_url", "image_url": {"url": "data:image/png;base64,..."}}
    ```
  - Text Documents: Content extracted locally from disk and injected directly into prompt text.
- **Enforced Limits**:
  - Max single file for base64: 25 MB
  - Max single file for text extraction: 10 MB
  - Max total attachments per request: 50 MB

---

## 17. Current Direct-to-Provider Paths

The mobile app currently calls provider APIs directly from device network interfaces:
- `https://api.openai.com/v1/responses`
- `https://api.anthropic.com/v1/messages`
- `https://generativelanguage.googleapis.com/v1beta`
- `https://api.deepseek.com/chat/completions`
- `https://api.moonshot.cn/v1/chat/completions`
- `https://api.siliconflow.cn/v1/chat/completions`
- `https://open.bigmodel.cn/api/paas/v4/chat/completions`
- `https://api.minimaxi.com/v1/text/chatcompletion_v2`
- `https://api.x.ai/v1/responses`
- `https://openrouter.ai/api/v1/responses`

---

## 18. Existing Backend Dependencies

- **Current Backend Dependencies**: **Zero**.
- LumenFlow operates completely self-contained with on-device SQLite and direct provider connections.
- It contains no analytics backends, no crash reporting services, and no update check servers.

---

## 19. New-API Compatibility Assessment

| Domain | LumenFlow Implementation | New-API Baseline (`feat/formobile`) | Compatibility Status |
| :--- | :--- | :--- | :--- |
| **Chat Endpoint** | `OtherProvider`: `/chat/completions` | `/v1/chat/completions` | **100% Compatible** |
| **Streaming** | SSE `data: {...}` + `[DONE]` | Standard SSE + `delta.content` | **100% Compatible** |
| **Reasoning** | `delta.reasoning_content` | DeepSeek / Claude thinking deltas | **100% Compatible** |
| **Models API** | `GET /v1/models` -> `data[].id` | `GET /v1/models` -> `data[].id` | **100% Compatible** |
| **Auth Header** | `Authorization: Bearer <key>` | `Authorization: Bearer sk-...` | **100% Compatible** |
| **Vision/Multimodal**| `image_url: {url: data:...}` | Standard OpenAI image format | **100% Compatible** |
| **User Accounts** | None (local only) | `/api/user/login`, `/api/user/register` | **Adapter Required** |
| **Server BYOK** | None (client-side only) | `/api/user_provider` (encrypted) | **UI & API Client Required** |

---

## 20. Target Architecture Evaluation: Direct Backend vs. BFF

### Evaluated Options

#### Option A: Direct Backend (`LumenFlow ↔ Tora/New-API`)
```text
┌─────────────────┐       HTTPS / SSE        ┌───────────────────────┐
│                 │ ───────────────────────> │                       │
│    LumenFlow    │                          │     Tora / New-API    │
│   Mobile App    │ <─────────────────────── │        Backend        │
│                 │      Token / Cookie      │                       │
└─────────────────┘                          └───────────────────────┘
                                                         │
                                        ┌────────────────┴────────────────┐
                                        ▼                                 ▼
                             Managed Upstreams (Stripe/Quota)     Encrypted BYOK Routing
```

#### Option B: Dedicated Backend-for-Frontend (BFF)
```text
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│  LumenFlow   │ ──> │   BFF / API  │ ──> │   New-API    │
│  Mobile App  │ <── │   Gateway    │ <── │   Backend    │
└──────────────┘     └──────────────┘     └──────────────┘
```

### Architectural Verdict: **Option A (Direct Backend) is Strongly Recommended**

**Rationale**:
1. **Redundancy Avoidance**: New-API already serves as an enterprise-grade API gateway and reverse proxy. It implements rate limiting, quota management, SSRF protection, channel failover, and token authentication.
2. **Zero Protocol Friction**: LumenFlow's HTTP client already speaks standard OpenAI HTTP/REST and SSE protocols.
3. **Reduced Latency & Complexity**: Eliminating a BFF removes an unnecessary network hop, reducing streaming latency on mobile connections and avoiding duplicate data models.
4. **Maintenance Efficiency**: Having mobile target New-API directly allows the existing web dashboard and mobile app to share identical backend services, models, and endpoints.

---

## 21. Target BYOK Integration Analysis

In the integrated product, BYOK transitions from client-side cleartext keys to **Server-Managed Encrypted BYOK**:

```text
1. User configures BYOK in Mobile App
       ↓
2. LumenFlow calls Tora Backend:
   POST /api/user_provider
   Headers: Authorization: Bearer <tora_token>
   Body: {"name": "My Anthropic", "type": 3, "key": "sk-ant-...", "base_url": ...}
       ↓
3. New-API validates credentials via TestUserProvider (isolated transport, SSRF safe)
       ↓
4. New-API encrypts key using AES-256-GCM + user-scoped AAD and stores in PostgreSQL
       ↓
5. When user chats in LumenFlow:
   POST /v1/chat/completions (model: "claude-3-7-sonnet")
       ↓
6. New-API distributor selects user's encrypted provider, decrypts in request scope, 
   verifies quota/subscription, and relays request securely upstream
```

**Security Advantage**:
- Mobile device never stores raw provider keys after initial server-side enrollment.
- Prevents device theft or memory dumping from exposing high-limit corporate keys.
- Centralizes egress filtering and audit logging on the Tora backend.

---

## 22. Target Subscription Integration Analysis

```text
LumenFlow Login
    ↓
GET /api/user/self  (or /api/user/subscription)
    ↓
Response contains:
  - role (user/admin)
  - quota balance
  - subscription: { plan_id, status: "active", expires_at, model_access }
    ↓
LumenFlow state stores active Entitlement
    ↓
Chat UI adjusts Model Picker:
  - Shows Premium models if subscription is active
  - Prompts Upgrade / Purchase if subscription expired or free-tier quota exhausted
```

---

## 23. Store Billing Boundary

- **Current State**: LumenFlow contains **zero** in-app purchase code.
- **Integration Boundary**:
  - In Phase 7D, integrate `purchases_flutter` (RevenueCat) or `in_app_purchase` into LumenFlow.
  - When a transaction completes on iOS (StoreKit) or Android (Google Play Billing), the receipt is submitted to New-API’s subscription webhook or receipt validation endpoint.
  - The server acts as the **single source of truth** for entitlement activation (`UserSubscription` table).

---

## 24. Model Availability Design

To prevent users from selecting models they cannot execute:
1. **Server Catalog Retrieval**: On app startup or switch to Managed mode, LumenFlow invokes `GET /v1/models`.
2. **Backend Filtering**: New-API automatically populates the model list based on:
   - Models available under the user's active subscription tier.
   - Models provided by the user's configured BYOK accounts.
3. **UI Grouping**: The model picker groups models into:
   - *Included in Plan* (ready to use)
   - *BYOK Configured* (routes to user's key)
   - *Upgrade Required* (disabled or marked with premium badge)

---

## 25. BYOK vs. Managed Mode Representation

Instead of an ambiguous global boolean, mobile will represent connection modes as distinct **Profiles / Provider Accounts**:

```dart
enum RoutingMode {
  managed, // Routes via Tora subscription / quota pool
  byok,    // Routes via server-stored encrypted BYOK channel
  local,   // Legacy: direct device-to-provider egress
}
```

The user selects either **Tora AI (Managed / Subscription)** or a specific **BYOK Account** from the platform selector.

---

## 26. API Contract Gap Matrix

| Mobile Capability | LumenFlow Existing | Backend Existing (`feat/formobile`) | Gap Description | Required Action |
| :--- | :--- | :--- | :--- | :--- |
| **Login / Register** | None | `POST /api/user/login`, `/register` | Missing UI & Auth Client | Build `LoginScreen`, `AuthService` |
| **Session Token Store**| Plaintext `SharedPreferences` | Cryptographic personal tokens (`sk-...`) | Missing secure keychain storage | Integrate `flutter_secure_storage` |
| **Account Info** | Local `UserProfile` | `GET /api/user/self` | No remote profile sync | Sync profile & quota from backend |
| **Chat Completions** | `OtherProvider` (`/chat/completions`) | `/v1/chat/completions` | Minor path/header adjustments | Add `ToraProvider` to `lib/providers/` |
| **Streaming (SSE)** | `HttpProviderBase` SSE parser | Standard SSE stream | None (fully compatible) | Reuse existing SSE accumulator |
| **Reasoning Deltas** | `reasoning_content` delta parser | DeepSeek / Claude thinking output | None (fully compatible) | Reuse existing accordion UI |
| **Model Listing** | `GET /v1/models` in platform settings| `GET /v1/models` | None (fully compatible) | Point model fetcher to Tora endpoint |
| **BYOK Management** | Local text inputs | `/api/user_provider` (CRUD) | Missing server BYOK API client | Build `ByokService` & Management UI |
| **Subscriptions** | None | `/api/subscription` & Stripe top-ups| Missing StoreKit / Play Billing SDK| Integrate `in_app_purchase` in Phase 7D |
| **Usage / Quota** | None | `user.quota`, `GetRemainingBudget` | No balance display in mobile UI | Add balance / quota card to drawer |

---

## 27. Mobile Security Boundary

The mobile application is an **untrusted client** running in an untrusted environment. Under zero-trust architecture:

```text
Untrusted Client (Mobile App)                Authoritative Boundary (New-API Backend)
─────────────────────────────                ─────────────────────────────────────────
User ID / Identity claim        ──[ MUST NOT TRUST ]──>  Derived solely from validated Token
Subscription status in UI       ──[ MUST NOT TRUST ]──>  Verified via DB before relaying
Quota / Budget checks in UI     ──[ MUST NOT TRUST ]──>  Enforced atomically via PreConsumeBilling
Channel / Upstream credentials  ──[ NEVER EXPOSE ]   ──> Decrypted in memory only on backend
Model pricing calculations      ──[ MUST NOT TRUST ]──>  Calculated authoritatively on server
```

---

## 28. Migration Concerns

- **Legacy Local Settings**: Existing LumenFlow users may have local API keys stored in `SharedPreferences`.
- **Migration Strategy**:
  - Keep legacy local provider mode intact under a distinct "Local / Direct" platform option.
  - Provide an "Export to Tora BYOK" migration button in settings: reads the local key and POSTs it to New-API’s `/api/user_provider`, then optionally purges the local plaintext key from `SharedPreferences`.

---

## 29. Offline Behavior

- **Available Offline**:
  - Viewing past conversations and message histories (cached in SQLite).
  - Searching conversation titles.
  - Viewing, editing, and managing local prompt presets.
  - Adjusting local appearance settings, themes, and background images.
- **Unavailable Offline**:
  - Sending new chat prompts to cloud LLMs (displays clear network connectivity error).
  - Refreshing remote model lists.
  - Purchasing subscriptions or syncing quota.

---

## 30. Error Contract & Mapping

| Backend Response / Status | New-API Error Payload | Proposed LumenFlow Mobile Mapping |
| :--- | :--- | :--- |
| **HTTP 401 Unauthorized** | `{"error": {"message": "Invalid token", "type": "auth_error"}}` | Invalidate token, display session expired dialog, navigate to Login |
| **HTTP 402 Payment Required**| `{"error": {"message": "Insufficient quota", "type": "quota_error"}}`| Open Quota Top-Up / Upgrade Subscription Sheet |
| **HTTP 403 Forbidden** | `{"error": {"message": "User is disabled or banned"}}` | Display account status alert with support contact |
| **HTTP 404 Model Not Found** | `{"error": {"message": "Model not available for user"}}` | Prompt user to refresh model catalog or configure BYOK |
| **HTTP 429 Rate Limit** | `{"error": {"message": "Rate limit exceeded, retry later"}}` | Back off automatically; display retry countdown banner |
| **HTTP 502 / 504 Upstream** | `{"error": {"message": "Channel error: ...", "type": "upstream"}}`| Show upstream provider outage banner with option to retry |

---

## 31. Environment Configuration

To support development, staging, and production environments without code modification:
- Use Dart environment defines (`--dart-define`):
  - `TORA_ENV`: `development` | `staging` | `production`
  - `TORA_API_BASE_URL`: defaults to `https://api.toraapi.com` (Prod), `https://staging-api.toraapi.com` (Staging), or `http://10.0.2.2:3001` (Local Android Emulator).
- Persist active backend URL in `SettingsService` for developer-mode switching.

---

## 32. Privacy, Telemetry & Secret Leakage Audit

- **Audit Result**: **CLEAN**.
- LumenFlow contains zero third-party telemetry, zero advertising SDKs, and zero background analytics.
- Prompts, attachments, and tokens remain strictly between the client and the configured endpoint.
- Logging via `debugPrint` is stripped or deactivated in release builds.

---

## 33. Recommended Target Architecture

**Architecture A — Direct Backend Integration** is selected:

```text
┌────────────────────────────────────────────────────────┐
│                   LumenFlow Mobile                     │
│                                                        │
│  ┌──────────────┐   ┌──────────────┐   ┌────────────┐  │
│  │  Chat UI &   │   │ Secure Store │   │   SQLite   │  │
│  │  Model Tray  │   │  (Keychain)  │   │ DB Storage │  │
│  └──────┬───────┘   └──────┬───────┘   └────────────┘  │
│         │                  │                           │
│         ▼                  ▼                           │
│  ┌──────────────────────────────────────────────────┐  │
│  │     ToraProvider (Implements AIProvider)         │  │
│  │   - Base URL: https://api.toraapi.com            │  │
│  │   - Header: Authorization: Bearer <tora_token>   │  │
│  │   - SSE Parser: Chunk & Reasoning Accumulator    │  │
│  └──────────────────────────┬───────────────────────┘  │
└─────────────────────────────┼──────────────────────────┘
                              │ HTTPS / SSE
                              ▼
┌────────────────────────────────────────────────────────┐
│               SaaSCover / New-API Backend              │
│                                                        │
│  ┌─────────────────────┐      ┌─────────────────────┐  │
│  │ Auth & User Service │      │ Relay Engine (/v1)  │  │
│  │  - /api/user/login  │      │  - Chat completions │  │
│  │  - /api/user/self   │      │  - Streaming proxy  │  │
│  └─────────────────────┘      └──────────┬──────────┘  │
│                                          │             │
│                                          ▼             │
│  ┌──────────────────────────────────────────────────┐  │
│  │          Router & Account Ownership Gate         │  │
│  │  - Checks UserSubscription / FreeTierQuota       │  │
│  │  - Selects Channel (Managed Channel vs BYOK)     │  │
│  │  - Decrypts AES-256-GCM BYOK Credentials (if set)│  │
│  └──────────────────────────────────────────────────┘  │
└────────────────────────────────────────────────────────┘
```

---

## 34. Proposed Phase 7 Implementation Roadmap

- **Phase 7B — Mobile / Backend API Contract & Authentication Client**:
  - Integrate `flutter_secure_storage`.
  - Implement `AuthService` and `LoginScreen` / `RegisterScreen`.
  - Implement token refresh and session state management.
- **Phase 7C — Tora Provider & Chat Stream Integration**:
  - Implement `ToraProvider` inside `lib/providers/` reusing `HttpProviderBase`.
  - Connect mobile model catalog directly to `GET /v1/models`.
  - Verify end-to-end SSE streaming, reasoning display, and cancellation against local New-API server.
- **Phase 7D — Server-Side BYOK & Subscription UI**:
  - Implement `ByokManagementScreen` connected to `/api/user_provider`.
  - Add subscription status card and plan entitlement checks.
  - Implement in-app purchase listener for Apple StoreKit & Google Play.
- **Phase 7E — Final End-to-End Verification & Release**:
  - Test offline/online transitions, token expiration, quota exhaustion, and cross-platform compilation (iOS & Android).

---

## 35. Mandatory Question Answers

### 1. What framework/language is Lumen built with?
LumenFlow is built with the **Flutter framework** using the **Dart language** (`sdk: ^3.11.1`). It uses Cupertino design components (`CupertinoApp`, `CupertinoThemeData`, `CupertinoPageScaffold`) for its core UI.

### 2. Where is the application entry point?
The application entry point is `void main() async` located in [`lib/main.dart`](file:///Users/noppanan/.gemini/antigravity/scratch/LumenFlow/lib/main.dart#L14-L59), which initializes window managers, orientation, notifications, theme settings, and executes `runApp(const MyApp())` loading `home: ChatScreen()`.

### 3. How does Lumen authenticate today?
LumenFlow currently uses **local-only / client-side BYOK authentication**. It has no user login, no backend account, no OAuth, and no JWT sessions. It simply injects the user's raw third-party API key directly into outbound provider HTTP headers.

### 4. Where are authentication credentials stored?
Credentials (API keys, endpoints) are stored in **plaintext inside `SharedPreferences`** (`lib/services/settings_service.dart`, lines 84, 577).

### 5. What API client does it use?
It uses Dart’s official `package:http/http.dart` client (`http.Client`, `http.Request`, `http.Response`), wrapped by the custom abstract base class `HttpProviderBase` (`lib/providers/http_provider_base.dart`).

### 6. Does it already support OpenAI-compatible APIs?
**Yes.** `OtherProvider` (`lib/providers/other_provider.dart`), `SiliconFlowProvider` (`lib/providers/siliconflow_provider.dart`), and `DeepSeekProvider` (`lib/providers/deepseek_provider.dart`) implement standard OpenAI-compatible `/chat/completions` and `/v1/models` protocols.

### 7. Which chat endpoint does it use?
- `OtherProvider`, `SiliconFlowProvider`, `DeepSeekProvider`: `$apiEndpoint/chat/completions`.
- `OpenAIProvider`, `GrokProvider`, `LMStudioProvider`: `$apiEndpoint/responses`.
- `ClaudeProvider`: `$apiEndpoint/messages`.
- `GeminiProvider`: Google REST generateContent endpoint.

### 8. How does streaming work?
Streaming is executed via HTTP POST chunked transfer with Server-Sent Events (SSE). Chunks are accumulated in a `StringBuffer` in `HttpProviderBase` / `OtherProvider`, split by `\n\n`, parsed for `data: ` JSON lines, and parsed for `delta.content` and `delta.reasoning_content`.

### 9. How are conversations persisted?
Conversations are persisted in a local **SQLite database** (`sqlite3: ^3.2.0`) managed by `ConversationDatabase` (`lib/services/conversation_database.dart`), with tables for `conversations`, `messages`, `attachments`, and `settings`, backed by an in-memory cache in `ConversationService`.

### 10. How are providers represented?
Providers are represented by the `AIPlatform` data model (`lib/models/ai_platform.dart`), containing `id`, `name`, `type`, `endpoint`, `apiKey`, `availableModels`, `defaultModel`, and `enabled`.

### 11. How are models represented?
Models are represented as string identifiers (`String`) stored in the `availableModels` list within each `AIPlatform`, with an active selection in `AIPlatform.defaultModel`.

### 12. Is model catalog local or server-driven?
The model catalog is currently **local**, with an on-demand dynamic refresh capability (`PlatformSettingsScreen._refreshPlatformModels`) that queries `GET /v1/models` on the target provider endpoint.

### 13. Does Lumen already implement BYOK?
**Yes**, but exclusively as **client-side BYOK** (user enters keys into the phone, and the phone calls providers directly).

### 14. Where are BYOK keys stored today?
In plaintext `SharedPreferences` on the device storage (`lib/services/settings_service.dart`).

### 15. Does Lumen call providers directly?
**Yes.** All outbound API calls currently egress directly from the mobile device to third-party provider servers (`api.openai.com`, `api.anthropic.com`, etc.).

### 16. Does it support custom base URLs?
**Yes.** `AIPlatform.endpoint` and `SettingsService.getApiEndpoint()` allow full customization of the API URL.

### 17. Does it already use a backend?
**No.** LumenFlow currently has zero backend server dependencies.

### 18. Can existing API abstraction target Tora/New-API?
**Yes.** `OtherProvider` can be duplicated and specialized into `ToraProvider` targeting `https://api.toraapi.com/v1/chat/completions` with zero architectural friction.

### 19. Can existing streaming consume New-API responses?
**Yes.** The SSE buffer and JSON delta extractor in `OtherProvider` match New-API’s streaming response format 100%.

### 20. What must change for server-managed BYOK?
Instead of storing third-party keys in local `SharedPreferences` and calling providers directly, LumenFlow will send the keys to New-API’s `/api/user_provider` endpoint (where they are encrypted with AES-256-GCM), and chat requests will route through New-API with a Tora user token.

### 21. What must change for subscription mode?
LumenFlow must fetch the user's active subscription status from `/api/user/self`, filter the model picker to permitted tiers, and handle HTTP 402 / quota errors with an upgrade prompt.

### 22. Is another backend/BFF actually necessary?
**No.** New-API already fulfills all gateway, authentication, billing, BYOK routing, and relay requirements. Adding a BFF would introduce latency and unnecessary complexity.

### 23. Where should provider selection occur?
Provider selection should occur **on the New-API backend** based on user identity, model requested, active subscription tier, and configured BYOK accounts.

### 24. Where should model selection occur?
Model selection should occur in the **mobile UI (ChatScreen / Model Selector)**, populated from New-API's `/v1/models` endpoint.

### 25. Where should entitlement be authoritative?
Entitlement must be strictly authoritative **on the New-API backend** (PostgreSQL `UserSubscription` and `User` balance tables).

### 26. How should mobile represent BYOK vs managed mode?
Mobile should represent them as distinct **Connection Profiles** or **Accounts** in the platform selector (e.g., "Tora AI Managed" vs. "My Anthropic BYOK").

### 27. What existing code should be reused?
- Entire UI layer (`ChatScreen`, `ConversationListScreen`, `MessageBubble`, `ChatInput`).
- SQLite storage layer (`ConversationDatabase`, `ConversationService`).
- SSE streaming and chunk accumulation logic (`HttpProviderBase`, `OtherProvider`).
- Attachment and multimodal base64 encoding (`FileService`, `Attachment`).
- Multi-language localization assets (`lib/l10n/`).

### 28. What existing code should eventually be removed/deprecated?
- Hardcoded direct provider calls (`ClaudeProvider`, `GeminiProvider`, `DeepSeekProvider`) can eventually be deprecated in favor of unified `ToraProvider` routing.
- Plaintext API key storage in `SettingsService`.
- Legacy single-platform `ApiSettingsScreen`.

### 29. What backend API gaps exist?
1. Need a streamlined mobile-friendly token issuance / login endpoint (`/api/user/login`).
2. Need an endpoint for mobile receipt validation if StoreKit / Google Play Billing is introduced.
3. Need clear error codes in `/v1/chat/completions` for quota exhaustion vs. subscription required.

### 30. What is the recommended implementation sequence?
1. **Phase 7B**: Mobile Secure Storage & Auth Client (`AuthService`, `flutter_secure_storage`, `LoginScreen`).
2. **Phase 7C**: `ToraProvider` implementation & `/v1/chat/completions` streaming verification.
3. **Phase 7D**: Server-side BYOK UI (`/api/user_provider`) and Subscription Status UI.
4. **Phase 7E**: End-to-end integration testing, receipt validation, and release readiness.

---

## 36. Findings / Risks

1. **Security Risk (High)**: Cleartext storage of API keys in `SharedPreferences`. Mitigated in Phase 7B by migrating to `flutter_secure_storage`.
2. **Architecture Finding (Medium)**: Discrepancy between `OpenAIProvider` (`/responses`) and `OtherProvider` (`/chat/completions`). Mitigated by basing the `ToraProvider` on `OtherProvider`.
3. **Licensing Isolation (Clean)**: Oriveo (AGPL-3.0-or-later) was audited for reference only and is completely isolated. LumenFlow (MIT) remains unencumbered.

---

## 37. Final Status & Conclusion

LumenFlow provides an outstanding, well-architected Flutter foundation that already possesses mature OpenAI-compatible networking, SSE stream buffering, multimodal support, and robust SQLite caching. Connecting LumenFlow directly to the hardened Tora/New-API backend requires no structural rework of the mobile app and requires no intermediate BFF gateway.

FINAL STATUS: READY FOR INTEGRATION DESIGN
