# Умный кредитный кабинет предпринимателя

Go API: Gin, PostgreSQL, GORM, JWT и bcrypt. Все демонстрационные данные синтетические.

```sh
cp .env.example .env
docker compose up --build -d
curl http://localhost:8080/healthz
```

Перед развертыванием задайте собственные `JWT_SECRET` и `DB_PASSWORD` в `.env`.
Демо: `client1`, `client2`, `client3`, `officer1`; пароль `123`.

Подробная архитектура, контракт API, запуск и проверка: [BACKEND_API.md](BACKEND_API.md).
