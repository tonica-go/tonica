package docker_compose

const dockerComposeTemplate = `version: "3"
services:
{{- if .AppDB }}
  app-db:
    image: postgres:15-alpine
    container_name: ps-app-db
    environment:
      POSTGRES_USER: app
      POSTGRES_PASSWORD: app
      POSTGRES_DB: app
    ports:
      - "5434:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U app"]
      interval: 5s
      timeout: 5s
      retries: 10
    volumes:
      - app-db-data:/var/lib/postgresql/data
    networks:
      - ps-local
{{- end }}
{{- if .Temporal }}

  temporal-db:
    image: postgres:15-alpine
    container_name: ps-temporal-db
    environment:
      POSTGRES_USER: temporal
      POSTGRES_PASSWORD: temporal
      POSTGRES_DB: temporal
    ports:
      - "5433:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U temporal"]
      interval: 5s
      timeout: 5s
      retries: 10
    volumes:
      - temporal-db-data:/var/lib/postgresql/data
    networks:
      - ps-local

  temporal:
    image: temporalio/auto-setup:latest
    container_name: ps-temporal
    depends_on:
      temporal-db:
        condition: service_healthy
    environment:
      DB: postgres12
      DB_PORT: 5432
      DBNAME: temporal
      POSTGRES_USER: temporal
      POSTGRES_PWD: temporal
      POSTGRES_HOST: temporal-db
      POSTGRES_SEEDS: temporal-db
      TEMPORAL_UI_PORT: 8233
    ports:
      - "7233:7233"
      - "8233:8233"
    networks:
      - ps-local

  temporal-admin-tools:
    image: temporalio/admin-tools:latest
    container_name: ps-temporal-admin
    depends_on:
      temporal:
        condition: service_started
    environment:
      TEMPORAL_TLS_ENABLE: "false"
      TEMPORAL_CLI_ADDRESS: temporal:7233
    entrypoint: ["/bin/sh", "-c", "sleep infinity"]
    networks:
      - ps-local

  temporal-ui:
    image: temporalio/ui:latest
    container_name: ps-temporal-ui
    depends_on:
      temporal:
        condition: service_started
    environment:
      TEMPORAL_ADDRESS: temporal:7233
      TEMPORAL_CORS_ORIGINS: http://localhost:5173,http://localhost:5174
      TEMPORAL_TLS_DISABLE_HOST_VERIFICATION: "true"
    ports:
      - "8234:8080"
    networks:
      - ps-local
{{- end }}
{{- if .Redpanda }}

  redpanda:
    image: docker.redpanda.com/redpandadata/redpanda:v24.1.9
    container_name: ps-redpanda
    command:
      - redpanda
      - start
      - --overprovisioned
      - --smp=1
      - --memory=1G
      - --reserve-memory=0M
      - --node-id=0
      - --check=false
      - --kafka-addr=internal://0.0.0.0:9092,external://0.0.0.0:19092
      - --advertise-kafka-addr=internal://redpanda:9092,external://localhost:19092
    ports:
      - "9092:9092"
      - "19092:19092"
    healthcheck:
      test: ["CMD-SHELL", "rpk cluster info -X brokers=localhost:9092 >/dev/null 2>&1"]
      interval: 10s
      timeout: 5s
      retries: 10
    networks:
      - ps-local

  redpanda-console:
    image: docker.redpanda.com/redpandadata/console:latest
    container_name: ps-redpanda-console
    depends_on:
      - redpanda
    environment:
      - KAFKA_BROKERS=redpanda:9092
      - SERVER_LISTEN_ADDR=0.0.0.0:8080
    ports:
      - "8081:8080"
    networks:
      - ps-local
{{- end }}
{{- if .Dragonfly }}

  dragonfly:
    image: 'docker.dragonflydb.io/dragonflydb/dragonfly'
    container_name: ps-dragonfly
    ulimits:
      memlock: -1
    ports:
      - "6379:6379"
    volumes:
      - dragonfly-data:/data
    networks:
      - ps-local
{{- end }}
{{- if .Mailhog }}

  mailhog:
    image: mailhog/mailhog:v1.0.1
    container_name: ps-mailhog
    ports:
      - "1025:1025"
      - "8025:8025"
    networks:
      - ps-local
{{- end }}
{{- if .Jaeger }}

  jaeger:
    image: jaegertracing/all-in-one:1.74.0
    container_name: ps-jaeger
    environment:
      - COLLECTOR_OTLP_ENABLED=true
    ports:
      - "16686:16686"
      - "4317:4317"
    networks:
      - ps-local
{{- end }}

networks:
  ps-local:
    driver: bridge
{{- if .HasVolumes }}

volumes:
{{- if .AppDB }}
  app-db-data:
{{- end }}
{{- if .Temporal }}
  temporal-db-data:
{{- end }}
{{- if .Dragonfly }}
  dragonfly-data:
{{- end }}
{{- end }}
`
