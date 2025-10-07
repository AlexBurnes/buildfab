# Git Integration: buildfab + pre-push

**Дата обновления**: 7 октября 2025

## Обзор интеграции

buildfab предоставляет Git интеграцию через **два способа использования**:

1. **Standalone CLI** - Прямое использование buildfab для запуска стадий
2. **Library API** - Встраивание в другие утилиты (например, pre-push)

## pre-push Utility

**pre-push** - это отдельная утилита для автоматической валидации проекта перед git push.

- **Проект**: https://github.com/AlexBurnes/pre-push
- **Версия**: v1.8.2 (Latest, Oct 7, 2025)
- **Статус**: Production-ready

### Ключевые особенности

- ✅ Встраивает buildfab как библиотеку
- ✅ Использует тот же `.project.yml` конфигурационный файл
- ✅ Автоматически запускается при `git push`
- ✅ Всегда выполняет stage `pre-push`
- ✅ Простая установка: `pre-push install`

## Архитектура

```
┌─────────────────────────────────────────────────────┐
│                    Git Push                         │
└──────────────────────┬──────────────────────────────┘
                       │
                       ↓
┌─────────────────────────────────────────────────────┐
│              .git/hooks/pre-push                    │
│              (installed by pre-push)                │
└──────────────────────┬──────────────────────────────┘
                       │
                       ↓
┌─────────────────────────────────────────────────────┐
│              pre-push utility (CLI)                 │
│          Embeds buildfab as library                 │
└──────────────────────┬──────────────────────────────┘
                       │
                       ↓
┌─────────────────────────────────────────────────────┐
│              buildfab library                       │
│          (pkg/buildfab API)                         │
└──────────────────────┬──────────────────────────────┘
                       │
                       ↓
┌─────────────────────────────────────────────────────┐
│         Execute "pre-push" stage                    │
│       from .project.yml configuration               │
└──────────────────────┬──────────────────────────────┘
                       │
                       ↓
┌─────────────────────────────────────────────────────┐
│     Run actions: git checks, tests, builds, etc.    │
└─────────────────────────────────────────────────────┘
```

## Shared Configuration

**Оба инструмента используют один файл**: `.project.yml`

```yaml
project:
  name: "my-project"
  modules: ["my-app"]

actions:
  - name: git-untracked
    uses: git@untracked  # Check for untracked files
  
  - name: version-check
    uses: version@check   # Validate version format
  
  - name: test
    run: go test ./...    # Run tests

stages:
  pre-push:  # ← Special stage for Git hooks
    steps:
      - action: git-untracked
      - action: version-check
      - action: test
```

## Два способа запуска

### 1. Автоматический (через pre-push)

```bash
# Установка Git hook (один раз)
pre-push install

# Теперь при git push автоматически:
git push origin main
# → Триггерит .git/hooks/pre-push
# → Запускает pre-push utility
# → Выполняет buildfab library
# → Запускает stage "pre-push"
```

### 2. Ручной (через buildfab CLI)

```bash
# Ручной запуск той же стадии
buildfab run pre-push

# С опциями
buildfab run pre-push --verbose
buildfab run pre-push --dry-run
```

## Установка

### 1. Установить pre-push utility

**macOS (Homebrew)**:
```bash
brew install AlexBurnes/tap/pre-push
```

**Windows (Scoop)**:
```bash
scoop bucket add AlexBurnes https://github.com/AlexBurnes/scoop-bucket
scoop install pre-push
```

**Linux (Direct)**:
```bash
curl -sSL https://github.com/AlexBurnes/pre-push/releases/latest/download/install-linux.sh | bash
```

### 2. Установить Git hook

```bash
# В директории проекта
cd /path/to/project

# Установить hook
pre-push install
```

### 3. Создать конфигурацию

Создать `.project.yml` в корне проекта:

```yaml
project:
  name: "my-project"

actions:
  - name: git-checks
    uses: git@untracked
  
  - name: test
    run: go test ./...

stages:
  pre-push:
    steps:
      - action: git-checks
      - action: test
```

### 4. Протестировать

```bash
# Тест вручную (через buildfab)
buildfab run pre-push

# Тест вручную (через pre-push)
pre-push test

# Автоматический тест (при git push)
git push origin main
```

## Ключевые преимущества

### ✅ Единая конфигурация

- **Один файл** `.project.yml` для обоих инструментов
- **Нет дублирования** настроек
- **Легкое обслуживание** - изменения в одном месте

### ✅ Гибкость использования

- **Автоматическое выполнение** через Git hook (pre-push)
- **Ручное выполнение** через CLI (buildfab)
- **Тестирование** перед коммитом

### ✅ Library-first архитектура

- **buildfab** - основная библиотека
- **pre-push** - специализированная обертка
- **Любые другие инструменты** могут использовать buildfab API

### ✅ Стандартный workflow

- **Знакомый паттерн** Git hooks
- **Современная конфигурация** YAML
- **Профессиональный вывод** с цветами и статусами

## Built-in Actions для Git

buildfab предоставляет встроенные actions для Git проверок:

```yaml
actions:
  # Проверка untracked файлов
  - name: check-untracked
    uses: git@untracked
  
  # Проверка uncommitted изменений
  - name: check-uncommitted
    uses: git@uncommitted
  
  # Проверка modified файлов (warning only)
  - name: check-modified
    uses: git@modified
  
  # Проверка формата версии
  - name: check-version
    uses: version@check
  
  # Проверка что версия - наибольшая
  - name: check-version-greatest
    uses: version@check-greatest
```

## Сравнение с альтернативами

| Аспект | buildfab + pre-push | GitHub Actions | Husky | pre-commit |
|--------|---------------------|----------------|-------|------------|
| **Конфигурация** | ✅ YAML (.project.yml) | ✅ YAML | ⚙️ JS | ⚙️ YAML |
| **Локальное выполнение** | ✅ Полная поддержка | ❌ Только облако | ✅ Да | ✅ Да |
| **Git hooks** | ✅ pre-push install | ❌ Нет | ✅ Да | ✅ Да |
| **DAG execution** | ✅ Полная | ✅ Да | ❌ Нет | ⚙️ Частичная |
| **Matrix builds** | ✅ Локально | ✅ Облако | ❌ Нет | ❌ Нет |
| **Containers** | ✅ Docker/Podman | ✅ Services | ❌ Нет | ⚙️ Частичная |
| **Library API** | ✅ Go embeddable | ❌ Нет | ❌ Нет | ❌ Нет |
| **Ручной запуск** | ✅ buildfab run | ❌ Нет | ⚙️ Через npm | ✅ Да |
| **Кросс-платформа** | ✅ Linux/Win/macOS | ⚙️ Runners | ✅ Да | ✅ Да |

## Реальное использование

### Пример 1: Базовая валидация

```yaml
# .project.yml
project:
  name: "go-service"

actions:
  - name: git-untracked
    uses: git@untracked
  
  - name: test
    run: go test ./...

stages:
  pre-push:
    steps:
      - action: git-untracked
      - action: test
```

**Workflow**:
```bash
# Установка
pre-push install

# Теперь каждый git push проверяет:
git push origin main
# → Нет untracked файлов?
# → Тесты проходят?
# → Push разрешен/запрещен
```

### Пример 2: Комплексная валидация

```yaml
# .project.yml
project:
  name: "production-service"

actions:
  - name: git-checks
    uses: git@untracked
  
  - name: version-check
    uses: version@check
  
  - name: lint
    run: golangci-lint run
  
  - name: test
    run: go test ./... -race
  
  - name: build
    run: go build ./...

stages:
  pre-push:
    steps:
      - action: git-checks
      - action: version-check
      - action: lint
      - action: test
        require: [lint]
      - action: build
        require: [test]
```

### Пример 3: С матрицами и контейнерами

```yaml
# .project.yml
project:
  name: "multi-platform-app"
  max_parallel: 2

actions:
  - name: test-platform
    container:
      image:
        from: golang:1.23-alpine
      run: go test ./...

stages:
  pre-push:
    steps:
      - action: test-platform
        matrix:
          values:
            platform: ["linux", "windows"]
          strategy:
            max_parallel: 2
            fail_fast: true
```

## Troubleshooting

### Hook не запускается

```bash
# Проверить установку
ls -la .git/hooks/pre-push

# Переустановить
pre-push install
```

### Тесты проходят локально, но не в hook

```bash
# Запустить вручную для отладки
buildfab run pre-push --verbose

# Или через pre-push
pre-push test --verbose
```

### Пропустить hook временно

```bash
# Использовать --no-verify
git push --no-verify origin main
```

## Дополнительная информация

- **buildfab проект**: https://github.com/AlexBurnes/buildfab
- **pre-push проект**: https://github.com/AlexBurnes/pre-push
- **buildfab документация**: [docs/](../docs/)
- **Comparison документ**: [docs/Comparison-with-others.md](Comparison-with-others.md)

---

**Статус**: ✅ Production-ready  
**Обновлено**: 7 октября 2025  
**Версия buildfab**: v0.20.0  
**Версия pre-push**: v1.8.2

