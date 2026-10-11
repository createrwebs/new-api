#!/bin/bash
set -e

# Update packages and install curl
apt-get update -y
apt-get install -y curl ca-certificates

# Install Ollama
curl -fsSL https://ollama.com/install.sh | sh

# Configure Ollama to listen on all interfaces (0.0.0.0)
mkdir -p /etc/systemd/system/ollama.service.d
cat << 'EOF' > /etc/systemd/system/ollama.service.d/override.conf
[Service]
Environment="OLLAMA_HOST=0.0.0.0:11434"
Environment="OLLAMA_ORIGINS=*"
EOF

systemctl daemon-reload
systemctl restart ollama

# Wait for Ollama service to be ready
until curl -s http://127.0.0.1:11434/api/tags > /dev/null; do
    echo "Waiting for Ollama service..."
    sleep 2
done

# Pull Qwen models
ollama pull qwen2.5:3b
ollama pull qwen2.5:1.5b

echo "Setup completed at $(date)" > /var/log/toraapi_ai_setup.log
