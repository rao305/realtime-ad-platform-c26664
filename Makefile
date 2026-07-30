.PHONY: tidy test test-go test-py up down logs demo report e2e invariants load lint

tidy:
	go mod tidy

test: test-go test-py

test-go:
	go test ./...

test-py:
	python3 -m pip install -q -r platform/processor/requirements.txt
	PYTHONPATH=platform/processor:platform/contracts/python python3 -m pytest platform/processor/tests platform/contracts/python -q

up:
	docker compose up -d --build

down:
	docker compose down -v

logs:
	docker compose logs -f --tail=200

demo:
	./scripts/demo.sh

report:
	./scripts/report.sh

e2e:
	./scripts/e2e.sh

invariants:
	./scripts/invariants.sh

load:
	./scripts/load_test.sh

lint:
	go vet ./...
	python3 -m pip install -q ruff
	ruff check platform/processor platform/cache_sync platform/analytics platform/contracts/python
