# Protohost Deploy - Makefile Include
# This file is auto-generated - do not edit manually

# Protohost global installation path
PROTOHOST_LIB := $(HOME)/.local/share/protohost-deploy

# Load configuration
include .protohost.config

# Load local overrides if they exist
-include .protohost.config.local

# Detect current branch
BRANCH := $(shell git branch --show-current 2>/dev/null || echo "main")
PROJECT_NAME := $(PROJECT_PREFIX)-$(BRANCH)

# Get dynamic ports from script
PORTS := $(shell python3 $(PROTOHOST_LIB)/lib/get_ports.py $(PROJECT_NAME))
WEB_PORT := $(word 1, $(PORTS))
MYSQL_PORT := $(word 2, $(PORTS))
REDIS_PORT := $(word 3, $(PORTS))

.PHONY: up down logs info restart list-all nginx-config nginx-enable nginx-disable nginx-list deploy update-protohost

up:
	@echo "Starting environment for project '$(PROJECT_NAME)'"
	@echo "  Web: http://localhost:$(WEB_PORT)"
	@echo "  MySQL: $(MYSQL_PORT)"
	@echo "  Redis: $(REDIS_PORT)"
	WEB_PORT=$(WEB_PORT) MYSQL_PORT=$(MYSQL_PORT) REDIS_PORT=$(REDIS_PORT) docker compose -p $(PROJECT_NAME) up -d --wait
	@mkdir -p $(REMOTE_BASE_DIR)/.ports
	@echo "WEB_PORT=$(WEB_PORT)" > $(REMOTE_BASE_DIR)/.ports/$(PROJECT_NAME)
	@echo "MYSQL_PORT=$(MYSQL_PORT)" >> $(REMOTE_BASE_DIR)/.ports/$(PROJECT_NAME)
	@echo "REDIS_PORT=$(REDIS_PORT)" >> $(REMOTE_BASE_DIR)/.ports/$(PROJECT_NAME)
	@echo "BRANCH=$(BRANCH)" >> $(REMOTE_BASE_DIR)/.ports/$(PROJECT_NAME)
	@echo "CREATED=$$(date -u +%Y-%m-%dT%H:%M:%SZ)" >> $(REMOTE_BASE_DIR)/.ports/$(PROJECT_NAME)
	@echo "EXPIRES=$$(date -u -d '+$(TTL_DAYS) days' +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || date -u -v+$(TTL_DAYS)d +%Y-%m-%dT%H:%M:%SZ)" >> $(REMOTE_BASE_DIR)/.ports/$(PROJECT_NAME)

down:
	docker compose -p $(PROJECT_NAME) down
	@rm -f $(REMOTE_BASE_DIR)/.ports/$(PROJECT_NAME)
	@rm -f $(REMOTE_BASE_DIR)/.nginx/$(PROJECT_NAME).conf

restart:
	WEB_PORT=$(WEB_PORT) MYSQL_PORT=$(MYSQL_PORT) REDIS_PORT=$(REDIS_PORT) docker compose -p $(PROJECT_NAME) restart

logs:
	docker compose -p $(PROJECT_NAME) logs -f

info:
	@echo "Environment Info:"
	@echo "  Branch:  $(BRANCH)"
	@echo "  Project: $(PROJECT_NAME)"
	@if [ -f $(REMOTE_BASE_DIR)/.ports/$(PROJECT_NAME) ]; then \
		. $(REMOTE_BASE_DIR)/.ports/$(PROJECT_NAME) && \
		echo "  Web:     http://localhost:$$WEB_PORT" && \
		echo "  MySQL:   $$MYSQL_PORT" && \
		echo "  Redis:   $$REDIS_PORT"; \
	else \
		echo "  Web:     http://localhost:$(WEB_PORT)"; \
		echo "  MySQL:   $(MYSQL_PORT)"; \
		echo "  Redis:   $(REDIS_PORT)"; \
	fi

list-all:
	@$(PROTOHOST_LIB)/lib/list_deployments.sh

nginx-config:
	@mkdir -p $(REMOTE_BASE_DIR)/.nginx
	@echo "Generating nginx config for $(PROJECT_NAME)..."
	@if [ ! -f $(REMOTE_BASE_DIR)/.ports/$(PROJECT_NAME) ]; then \
		echo "❌ Error: Deployment not running. Run 'make up' first."; \
		exit 1; \
	fi
	@. $(REMOTE_BASE_DIR)/.ports/$(PROJECT_NAME) && \
	echo "Using port: $$WEB_PORT" && \
	echo "server {" > $(REMOTE_BASE_DIR)/.nginx/$(PROJECT_NAME).conf
	@. $(REMOTE_BASE_DIR)/.ports/$(PROJECT_NAME) && \
	echo "    listen 443 ssl;" >> $(REMOTE_BASE_DIR)/.nginx/$(PROJECT_NAME).conf && \
	echo "    server_name $(PROJECT_NAME).$(REMOTE_HOST);" >> $(REMOTE_BASE_DIR)/.nginx/$(PROJECT_NAME).conf && \
	echo "" >> $(REMOTE_BASE_DIR)/.nginx/$(PROJECT_NAME).conf && \
	echo "    ssl_certificate /etc/letsencrypt/live/$(REMOTE_HOST)/fullchain.pem;" >> $(REMOTE_BASE_DIR)/.nginx/$(PROJECT_NAME).conf && \
	echo "    ssl_certificate_key /etc/letsencrypt/live/$(REMOTE_HOST)/privkey.pem;" >> $(REMOTE_BASE_DIR)/.nginx/$(PROJECT_NAME).conf && \
	echo "    include ssl-params.conf;" >> $(REMOTE_BASE_DIR)/.nginx/$(PROJECT_NAME).conf && \
	echo "" >> $(REMOTE_BASE_DIR)/.nginx/$(PROJECT_NAME).conf && \
	echo "    location / {" >> $(REMOTE_BASE_DIR)/.nginx/$(PROJECT_NAME).conf && \
	echo "        proxy_pass http://$(NGINX_PROXY_HOST):$$WEB_PORT;" >> $(REMOTE_BASE_DIR)/.nginx/$(PROJECT_NAME).conf && \
	echo "        proxy_http_version 1.1;" >> $(REMOTE_BASE_DIR)/.nginx/$(PROJECT_NAME).conf && \
	echo "        proxy_set_header Upgrade \$$http_upgrade;" >> $(REMOTE_BASE_DIR)/.nginx/$(PROJECT_NAME).conf && \
	echo "        proxy_set_header Connection \"upgrade\";" >> $(REMOTE_BASE_DIR)/.nginx/$(PROJECT_NAME).conf && \
	echo "        proxy_set_header Host \$$host;" >> $(REMOTE_BASE_DIR)/.nginx/$(PROJECT_NAME).conf && \
	echo "        proxy_set_header X-Real-IP \$$remote_addr;" >> $(REMOTE_BASE_DIR)/.nginx/$(PROJECT_NAME).conf && \
	echo "        proxy_set_header X-Forwarded-For \$$proxy_add_x_forwarded_for;" >> $(REMOTE_BASE_DIR)/.nginx/$(PROJECT_NAME).conf && \
	echo "        proxy_set_header X-Forwarded-Proto \$$scheme;" >> $(REMOTE_BASE_DIR)/.nginx/$(PROJECT_NAME).conf && \
	echo "        proxy_read_timeout 86400;" >> $(REMOTE_BASE_DIR)/.nginx/$(PROJECT_NAME).conf && \
	echo "        proxy_buffering off;" >> $(REMOTE_BASE_DIR)/.nginx/$(PROJECT_NAME).conf && \
	echo "    }" >> $(REMOTE_BASE_DIR)/.nginx/$(PROJECT_NAME).conf && \
	echo "}" >> $(REMOTE_BASE_DIR)/.nginx/$(PROJECT_NAME).conf && \
	echo "Nginx config written to $(REMOTE_BASE_DIR)/.nginx/$(PROJECT_NAME).conf" && \
	echo "  URL: https://$(PROJECT_NAME).$(REMOTE_HOST)" && \
	echo "  Port: $$WEB_PORT" && \
	echo "Deploying to nginx server..." && \
	scp $(REMOTE_BASE_DIR)/.nginx/$(PROJECT_NAME).conf $(NGINX_SERVER):/tmp/protohost-$(PROJECT_NAME).conf && \
	ssh $(NGINX_SERVER) "sudo mv /tmp/protohost-$(PROJECT_NAME).conf /etc/nginx/sites-enabled/protohost-$(PROJECT_NAME).conf && sudo service nginx restart" && \
	echo "✓ Nginx config deployed and nginx restarted"

nginx-enable:
	@$(PROTOHOST_LIB)/lib/nginx_manage.sh enable $(PROJECT_NAME)

nginx-disable:
	@$(PROTOHOST_LIB)/lib/nginx_manage.sh disable $(PROJECT_NAME)

nginx-list:
	@$(PROTOHOST_LIB)/lib/nginx_manage.sh list

deploy:
	@echo "Deploying to $(or $(HOST),$(REMOTE_HOST))..."
	@$(PROTOHOST_LIB)/lib/deploy.sh $(if $(RESET_DB),--reset-db) $(if $(NUKE),--nuke) $(if $(HOST),--host $(HOST))

update-protohost:
	@echo "🔄 Updating protohost-deploy to latest version..."
	@if [ -d /tmp/protohost-deploy ]; then \
		echo "  Updating existing clone..."; \
		cd /tmp/protohost-deploy && git pull origin main; \
	else \
		echo "  Cloning latest version..."; \
		git clone git@github.com:thatjpcsguy/protohost.git /tmp/protohost-deploy; \
	fi
	@echo "  Running global installer..."
	@/tmp/protohost-deploy/global-install.sh
	@echo "  Updating project Makefile.inc..."
	@cp $(PROTOHOST_LIB)/templates/Makefile.inc .protohost/Makefile.inc
	@echo "✅ Protohost-deploy updated successfully!"
	@echo "   Your .protohost.config and .protohost.config.local were preserved"
