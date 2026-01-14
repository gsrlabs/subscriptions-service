#!/bin/bash

GREEN='\033[0;32m'
NC='\033[0m' # No Color

echo -e "${GREEN}Настройка окружения...${NC}"

# Копируем .env если его еще нет
if [ ! -f .env ]; then
    cp .env.example .env
    echo -e "${GREEN}Файл .env создан из .env.example${NC}"
else
    echo "Файл .env уже существует, пропускаю копирование."
fi

# Проверка Docker
if ! command -v docker-compose &> /dev/null && ! command -v docker compose &> /dev/null; then
    echo "Ошибка: docker-compose не найден. Пожалуйста, установите Docker."
    exit 1
fi

echo -e "${GREEN}Запуск контейнеров через Docker Compose...${NC}"
docker compose up --build -d

echo -e "${GREEN}=============================================${NC}"
echo -e "${GREEN}Сервис успешно запущен!${NC}"
echo -e "Swagger: http://localhost:8090/swagger/index.html"
echo -e "${GREEN}=============================================${NC}"