#!/bin/bash

# Docker setup script for Weather API
# This script starts Docker containers and applies migrations

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🐳 Weather API Docker Setup${NC}"
echo ""

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo -e "${RED}❌ Docker is not running!${NC}"
    echo ""
    echo "Please start Docker Desktop or OrbStack and try again."
    echo ""
    echo "If you're using:"
    echo "  - Docker Desktop: Open the Docker Desktop application"
    echo "  - OrbStack: Run 'open -a OrbStack'"
    echo ""
    exit 1
fi

echo -e "${GREEN}✅ Docker is running${NC}"
echo ""

# Load environment variables
if [ -f .env ]; then
    export $(cat .env | grep -v '^#' | xargs)
fi

# Start containers
echo -e "${BLUE}📦 Starting Docker containers...${NC}"
docker compose up -d

# Wait for databases to be healthy
echo ""
echo -e "${YELLOW}⏳ Waiting for databases to be ready...${NC}"

# Function to check if database is ready
wait_for_db() {
    local container=$1
    local max_attempts=30
    local attempt=1
    
    while [ $attempt -le $max_attempts ]; do
        if docker exec $container pg_isready > /dev/null 2>&1; then
            echo -e "${GREEN}✅ $container is ready${NC}"
            return 0
        fi
        echo -n "."
        sleep 1
        attempt=$((attempt + 1))
    done
    
    echo -e "${RED}❌ $container failed to start${NC}"
    return 1
}

wait_for_db "weather_postgres"
wait_for_db "weather_postgres_test"

echo ""
echo -e "${GREEN}✅ All databases are ready${NC}"
echo ""

# Check if golang-migrate is installed
if ! command -v migrate &> /dev/null; then
    echo -e "${YELLOW}⚠️  golang-migrate is not installed${NC}"
    echo ""
    echo "To install it:"
    echo "  - macOS: brew install golang-migrate"
    echo "  - Or via Go: go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest"
    echo ""
    echo "Skipping migrations..."
    echo ""
else
    # Apply migrations
    echo -e "${BLUE}🔄 Applying migrations...${NC}"
    echo ""
    
    echo -e "${YELLOW}Development database:${NC}"
    export DATABASE_URL="postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@localhost:${POSTGRES_PORT}/${POSTGRES_DB}?sslmode=disable"
    migrate -path migrations -database "$DATABASE_URL" up
    echo -e "${GREEN}✅ Dev migrations applied${NC}"
    echo ""
    
    echo -e "${YELLOW}Test database:${NC}"
    export DATABASE_URL="postgres://${POSTGRES_USER_TEST}:${POSTGRES_PASSWORD_TEST}@localhost:${POSTGRES_PORT_TEST}/${POSTGRES_DB_TEST}?sslmode=disable"
    migrate -path migrations -database "$DATABASE_URL" up
    echo -e "${GREEN}✅ Test migrations applied${NC}"
    echo ""
fi

# Show container status
echo -e "${BLUE}📊 Container Status:${NC}"
docker compose ps
echo ""

# Show connection info
echo -e "${GREEN}🎉 Setup complete!${NC}"
echo ""
echo -e "${BLUE}Connection Information:${NC}"
echo ""
echo -e "${YELLOW}Development Database:${NC}"
echo "  Host:     localhost"
echo "  Port:     ${POSTGRES_PORT:-5432}"
echo "  Database: ${POSTGRES_DB:-weather_db}"
echo "  User:     ${POSTGRES_USER:-weather_user}"
echo "  Password: ${POSTGRES_PASSWORD:-weather_password}"
echo ""
echo -e "${YELLOW}Test Database:${NC}"
echo "  Host:     localhost"
echo "  Port:     ${POSTGRES_PORT_TEST:-5433}"
echo "  Database: ${POSTGRES_DB_TEST:-weather_db_test}"
echo "  User:     ${POSTGRES_USER_TEST:-weather_user_test}"
echo "  Password: ${POSTGRES_PASSWORD_TEST:-weather_password_test}"
echo ""
echo -e "${BLUE}Next Steps:${NC}"
echo "  1. Run the application: make run"
echo "  2. Run tests: make test"
echo "  3. View logs: make docker-logs"
echo "  4. Restart containers: make docker-restart"
echo "  5. Stop containers: make docker-down"
echo ""
