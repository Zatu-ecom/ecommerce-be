.PHONY: help build build-dev run run-dev stop clean logs test migrate docker-up docker-down docker-restart

# Default target
help:
	@echo "🛍️  Zatu E-commerce Backend - Docker Commands"
	@echo ""
	@echo "📦 Build Commands:"
	@echo "  make build          - Build production Docker image"
	@echo ""
	@echo "🚀 Run Commands:"
	@echo "  make up             - Start all services"
	@echo "  make down           - Stop all services"
	@echo "  make restart        - Restart all services"
	@echo ""
	@echo "🔧 Utility Commands:"
	@echo "  make logs           - View application logs"
	@echo "  make logs-all       - View all services logs"
	@echo "  make migrate        - Run database migrations"
	@echo "  make shell          - Open shell in app container"
	@echo "  make db-shell       - Open PostgreSQL shell"
	@echo "  make redis-cli      - Open Redis CLI"
	@echo ""
	@echo "🧹 Cleanup Commands:"
	@echo "  make clean          - Stop and remove containers"
	@echo "  make clean-all      - Stop, remove containers and volumes (⚠️  deletes data)"
	@echo "  make prune          - Remove unused Docker resources"
	@echo ""
	@echo "🧪 Test Commands:"
	@echo "  make test           - Run tests in Docker"
	@echo "  make test-local     - Run tests locally"
	@echo ""
	@echo "📊 Monitoring Commands:"
	@echo "  make ps             - Show running containers"
	@echo "  make stats          - Show container resource usage"
	@echo "  make health         - Check service health"

# Build production image
build:
	@echo "🏗️  Building production image..."
	DOCKER_BUILDKIT=1 docker build -t ecommerce-backend:latest --target runtime .
	@echo "✅ Build complete!"



# Start all services (production)
up:
	@echo "🚀 Starting services (production)..."
	docker-compose up -d
	@echo "✅ Services started!"
	@echo "📝 Run 'make migrate' to set up the database"
	@echo "🌐 API available at http://localhost:8080"



# Stop all services
down:
	@echo "🛑 Stopping services..."
	docker-compose down
	@echo "✅ Services stopped!"

# Restart all services
restart:
	@echo "🔄 Restarting services..."
	docker-compose restart
	@echo "✅ Services restarted!"

# View application logs
logs:
	docker-compose logs -f app

# View all logs
logs-all:
	docker-compose logs -f

# Run database migrations
migrate:
	@echo "🗄️  Running database migrations..."
	docker-compose --profile tools run --rm migrate
	@echo "✅ Migrations complete!"

# Open shell in app container
shell:
	docker-compose exec app sh

# Open PostgreSQL shell
db-shell:
	docker-compose exec postgres psql -U postgres -d ecommerce

# Open Redis CLI
redis-cli:
	docker-compose exec redis redis-cli

# Show running containers
ps:
	docker-compose ps

# Show container stats
stats:
	docker stats --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.NetIO}}"

# Check service health
health:
	@echo "🏥 Checking service health..."
	@docker-compose ps --format json | jq -r '.[] | "\(.Service): \(.State) - \(.Health)"'

# Clean up containers
clean:
	@echo "🧹 Cleaning up containers..."
	docker-compose down
	@echo "✅ Cleanup complete!"

# Clean up everything including volumes
clean-all:
	@echo "⚠️  This will delete all data!"
	@read -p "Are you sure? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		docker-compose down -v; \
		echo "✅ All containers and volumes removed!"; \
	else \
		echo "❌ Cancelled"; \
	fi

# Prune unused Docker resources
prune:
	@echo "🧹 Removing unused Docker resources..."
	docker system prune -f
	@echo "✅ Prune complete!"

# Run tests in Docker
test:
	@echo "🧪 Running tests in Docker..."
	docker-compose --profile dev run --rm app-dev go test ./test/integration/... -v

# Run tests locally
test-local:
	@echo "🧪 Running tests locally..."
	go test ./test/integration/... -v

# Run all tests with summary (failed tests shown at end)
test-all:
	@echo "🧪 Running all integration tests..."
	@go test ./test/integration/... -v 2>&1 | tee /tmp/test_output.txt; \
	EXIT_CODE=$$?; \
	echo ""; \
	echo "=========================================="; \
	echo "           📊 TEST SUMMARY"; \
	echo "=========================================="; \
	echo ""; \
	FAILED=$$(grep -c "^    --- FAIL:\|^--- FAIL:" /tmp/test_output.txt 2>/dev/null || echo "0"); \
	PASSED=$$(grep -c "^    --- PASS:\|^--- PASS:" /tmp/test_output.txt 2>/dev/null || echo "0"); \
	TOTAL=$$((PASSED + FAILED)); \
	echo "📈 Total:  $$TOTAL"; \
	echo "✅ Passed: $$PASSED"; \
	echo "❌ Failed: $$FAILED"; \
	echo ""; \
	if [ "$$FAILED" -gt 0 ]; then \
		echo "=========================================="; \
		echo "           ❌ FAILED TESTS"; \
		echo "=========================================="; \
		grep "^    --- FAIL:\|^--- FAIL:" /tmp/test_output.txt | sed 's/^    //' | sort -u; \
	else \
		echo "🎉 All tests passed!"; \
	fi; \
	exit $$EXIT_CODE

# Run tests with JSON output for CI/CD
test-json:
	@echo "🧪 Running tests with JSON output..."
	go test ./test/integration/... -json 2>&1 | tee test-results.json

# Re-run only failed tests from the last test-all run
test-failed:
	@echo "🔄 Re-running failed tests..."
	@if [ ! -f /tmp/test_output.txt ]; then \
		echo "❌ No previous test run found. Run 'make test-all' first."; \
		exit 1; \
	fi; \
	FAILED_TESTS=$$(grep "^--- FAIL:" /tmp/test_output.txt | sed 's/--- FAIL: //' | sed 's/ (.*//' | tr '\n' '|' | sed 's/|$$//'); \
	if [ -z "$$FAILED_TESTS" ]; then \
		echo "✅ No failed tests to re-run!"; \
	else \
		echo "Running: $$FAILED_TESTS"; \
		go test ./test/integration/... -v -run "$$FAILED_TESTS" 2>&1 | tee /tmp/test_output.txt; \
		echo ""; \
		echo "=========================================="; \
		echo "           📊 RE-RUN SUMMARY"; \
		echo "=========================================="; \
		FAILED=$$(grep -c "^--- FAIL:" /tmp/test_output.txt 2>/dev/null || echo "0"); \
		PASSED=$$(grep -c "^--- PASS:" /tmp/test_output.txt 2>/dev/null || echo "0"); \
		echo "✅ Passed: $$PASSED"; \
		echo "❌ Failed: $$FAILED"; \
		if [ "$$FAILED" -gt 0 ]; then \
			echo ""; \
			echo "=========================================="; \
			echo "           ❌ STILL FAILING"; \
			echo "=========================================="; \
			grep "^--- FAIL:" /tmp/test_output.txt; \
		fi; \
	fi

# Quick start (build + up + migrate)
quickstart: build up migrate
	@echo "🎉 Quick start complete!"
	@echo "🌐 API available at http://localhost:8080"
