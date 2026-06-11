
# Usage:
#   make test-all          Run tests for all services
#   make test-fastapi      Run FastAPI tests with coverage
#   make test-gin          Run Gin tests with coverage
#   make test-spring       Run Spring Boot tests with coverage
#   make clean             Remove all coverage artifacts
#

.PHONY: test-fastapi test-gin test-spring clean up down


test-fastapi:
	cd fastapi-service && py -m pytest --cov=. --cov-report=html
	@echo "Coverage report: fastapi-service/htmlcov/index.html"

test-gin:
	cd gin-service && go test ./... -coverprofile=coverage.out && go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: gin-service/coverage.html"

test-spring:
	cd spring-boot-service && gradlew.bat clean test jacocoTestReport
	@echo "Coverage report: spring-boot-service/build/reports/jacoco/test/html/index.html"

clean:
	@echo "Cleaning coverage artifacts..."
	rm -rf fastapi-service/htmlcov fastapi-service/.coverage
	rm -f gin-service/coverage.out gin-service/coverage.html
	rm -rf spring-boot-service/build/reports
	@echo "Done."

build:
	docker-compose up -d --build
up:
	docker-compose up -d
down:
	docker-compose down

temporal-run:
	temporal server start-dev --db-filename temporal-data.db