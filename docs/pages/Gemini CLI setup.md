# Gemini CLI setup

Connect the official Gemini CLI to ToraApi with your API key through the native Gemini API.

## 1. Create an API key

Create a ToraApi API key.

## 2. Install Gemini CLI

~~~bash
npm install -g @google/gemini-cli@latest
gemini --version
~~~

## 3. Configure ToraApi

Replace `YOUR_ToraApi_API_KEY` with your complete ToraApi API key.

**Windows PowerShell**

~~~powershell
$env:GEMINI_API_KEY="YOUR_ToraApi_API_KEY"
$env:GOOGLE_GEMINI_BASE_URL="https://toraapi.com"
~~~

**Linux or macOS**

~~~bash
export GEMINI_API_KEY="YOUR_ToraApi_API_KEY"
export GOOGLE_GEMINI_BASE_URL="https://toraapi.com"
~~~

## 4. Start and test

Replace `YOUR_MODEL_ID` with a model ID from the ToraApi model catalog. In the same terminal, open your project directory and run:

~~~bash
gemini --model YOUR_MODEL_ID
~~~

If prompted, select **Use Gemini API key**, then send:

~~~text
Reply with one sentence confirming the connection is working.
~~~

A normal reply means setup is complete.