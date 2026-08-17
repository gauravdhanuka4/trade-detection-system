




env:
	@if [ ! -f .env ]; then \
		echo "Creating .env file from .env.example..."; \
		cp .env.example .env; \
		echo ".env file created successfully."; \
	else \
		echo ".env file already exists."; \
	fi