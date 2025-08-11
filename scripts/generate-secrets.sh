#!/bin/bash

# SA3D Security Secrets Generation Script
# This script generates secure secrets for the SA3D application

set -euo pipefail

echo "🔐 SA3D Security Secrets Generation"
echo "================================="

# Function to generate a secure random string
generate_secret() {
    local length=${1:-32}
    openssl rand -base64 $length | tr -d "=+/" | cut -c1-$length
}

# Function to generate a strong password
generate_password() {
    local length=${1:-16}
    openssl rand -base64 $length | tr -d "=+/" | head -c $length
}

# Create .env file if it doesn't exist
if [ ! -f .env ]; then
    cp .env.example .env
    echo "✅ Created .env file from .env.example"
else
    echo "⚠️  .env file already exists. Updating secrets..."
fi

# Generate JWT secret (64 characters for extra security)
JWT_SECRET=$(generate_secret 64)
echo "✅ Generated JWT secret (64 characters)"

# Generate database passwords
DB_PASSWORD=$(generate_password 20)
REDIS_PASSWORD=$(generate_password 16)
echo "✅ Generated database passwords"

# Generate application secrets
APP_SECRET_KEY=$(generate_secret 32)
ENCRYPTION_KEY=$(generate_secret 32)
echo "✅ Generated application secrets"

# Update .env file
{
    echo "# Generated on $(date)"
    echo "# Environment Configuration"
    echo "ENVIRONMENT=development"
    echo "LOG_LEVEL=info"
    echo ""
    echo "# Database Configuration (CRITICAL: Secure these in production)"
    echo "DB_HOST=localhost"
    echo "DB_PORT=5432"
    echo "DB_USER=sa3d"
    echo "DB_PASSWORD=$DB_PASSWORD"
    echo "DB_NAME=sa3d_db"
    echo "DB_SSL_MODE=require"
    echo ""
    echo "# Redis Configuration"
    echo "REDIS_HOST=localhost"
    echo "REDIS_PORT=6379"
    echo "REDIS_PASSWORD=$REDIS_PASSWORD"
    echo "REDIS_DB=0"
    echo ""
    echo "# Kafka Configuration"
    echo "KAFKA_BROKERS=localhost:9092"
    echo "KAFKA_GROUP_ID=sa3d-services"
    echo "KAFKA_TOPIC_ANALYSIS=analysis-events"
    echo "KAFKA_TOPIC_METRICS=metrics-events"
    echo "KAFKA_TOPIC_COLLABORATION=collaboration-events"
    echo ""
    echo "# Service Ports"
    echo "API_GATEWAY_PORT=8080"
    echo "ANALYSIS_SERVICE_PORT=8081"
    echo "VISUALIZATION_SERVICE_PORT=8082"
    echo "COLLABORATION_SERVICE_PORT=8083"
    echo "METRICS_SERVICE_PORT=8084"
    echo ""
    echo "# JWT Configuration (CRITICAL: Keep this secret)"
    echo "JWT_SECRET=$JWT_SECRET"
    echo "JWT_EXPIRY=24h"
    echo ""
    echo "# Application Security"
    echo "APP_SECRET_KEY=$APP_SECRET_KEY"
    echo "ENCRYPTION_KEY=$ENCRYPTION_KEY"
    echo ""
    echo "# CORS Configuration"
    echo "CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:5173"
    echo "CORS_ALLOWED_METHODS=GET,POST,PUT,DELETE,OPTIONS"
    echo "CORS_ALLOWED_HEADERS=Content-Type,Authorization,X-Request-ID"
    echo ""
    echo "# Rate Limiting"
    echo "RATE_LIMIT_REQUESTS_PER_MINUTE=60"
    echo "RATE_LIMIT_BURST=10"
    echo ""
    echo "# Analysis Configuration"
    echo "MAX_FILE_SIZE_MB=100"
    echo "MAX_PROJECT_SIZE_GB=10"
    echo "ANALYSIS_TIMEOUT_MINUTES=30"
    echo "WORKER_POOL_SIZE=0"
    echo ""
    echo "# WebSocket Configuration"
    echo "WS_READ_BUFFER_SIZE=1024"
    echo "WS_WRITE_BUFFER_SIZE=1024"
    echo "WS_MAX_MESSAGE_SIZE=512000"
} > .env.tmp

mv .env.tmp .env
chmod 600 .env  # Secure permissions

echo ""
echo "🎉 Secrets generated successfully!"
echo ""
echo "📋 Summary:"
echo "  - JWT Secret: 64 characters (secure)"
echo "  - DB Password: 20 characters (strong)"
echo "  - Redis Password: 16 characters (strong)"
echo "  - App Secrets: 32 characters each"
echo ""
echo "🔒 Security Notes:"
echo "  - All secrets are cryptographically secure"
echo "  - .env file permissions set to 600 (owner read/write only)"
echo "  - Never commit .env file to version control"
echo "  - Rotate secrets regularly in production"
echo ""
echo "⚡ Next Steps:"
echo "  1. Review the generated .env file"
echo "  2. For production: Store secrets in a secure vault"
echo "  3. Run 'docker-compose up --build' to test"
echo ""

# Generate a secrets backup for ops team (without actual values)
cat > .env.template << EOF
# SA3D Environment Template
# Copy this to .env and fill in the values

# Database Configuration
DB_HOST=localhost
DB_PORT=5432
DB_USER=sa3d
DB_PASSWORD=<GENERATE_20_CHAR_PASSWORD>
DB_NAME=sa3d_db
DB_SSL_MODE=require

# Redis Configuration  
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=<GENERATE_16_CHAR_PASSWORD>
REDIS_DB=0

# JWT Configuration (CRITICAL)
JWT_SECRET=<GENERATE_64_CHAR_SECRET>
JWT_EXPIRY=24h

# Application Security
APP_SECRET_KEY=<GENERATE_32_CHAR_SECRET>
ENCRYPTION_KEY=<GENERATE_32_CHAR_SECRET>
EOF

echo "📄 Created .env.template for deployment reference"
echo ""
echo "🛡️  Security Implementation Complete!"