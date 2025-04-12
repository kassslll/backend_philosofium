# backend_philosofium

/root
│
├── /backend                # Основная папка backend
│   ├── /config
│   │   └── config.go       # Конфигурация приложения
│   │
│   ├── /controllers
│   │   ├── auth_controller.go
│   │   ├── courses_controller.go
│   │   ├── tests_controller.go
│   │   └── ...             # Другие контроллеры
│   │
│   ├── /middleware
│   │   ├── auth.go
│   │   └── logging.go
│   │
│   ├── /migrations
│   │   └── 001_init_schema.up.sql
│   │
│   ├── /models
│   │   ├── analytics.go
│   │   ├── courses.go
│   │   ├── tests.go
│   │   └── ...             # Другие модели
│   │
│   ├── /routes
│   │   └── routes.go       # Маршруты API
│   │
│   ├── /utils
│   │   ├── database.go
│   │   ├── jwt.go
│   │   └── logger.go
│   │
│   ├── main.go             # Точка входа
│   └── go.mod              # Go модули
│
├── /tests                  # Тесты
│   ├── auth_test.go
│   ├── courses_test.go
│   ├── tests_test.go
│   └── main_test.go        # Главный тестовый файл
│
├── .env                    # Переменные окружения (в .gitignore)
├── .env.example            # Пример переменных окружения
├── Dockerfile              # Конфигурация Docker
├── docker-compose.yml      # Production конфигурация
├── docker-compose.dev.yml  # Development конфигурация
├── go.sum                 # Зависимости
└── README.md              # Документация проекта