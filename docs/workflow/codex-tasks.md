# Работа с задачами через OpenCode CLI

## Что OpenCode загружает автоматически

При запуске в корне репо OpenCode читает:

1. **`~/.config/opencode/AGENTS.md`** — глобальные правила (язык, стиль, B-Day/A-Day, git workflow)
2. **`AGENTS.md` (корень репо)** — архитектура проекта, tech stack, Go conventions
3. **Skills** (при вызове `/go-expert`, `/samber-do`) — детальные паттерны кода
4. **`docs/conventions/`** — ссылки из AGENTS.md, OpenCode может прочитать по запросу

OpenCode уже знает КАК писать код для этого проекта. Тебе нужно указать только ЧТО делать.

## Откуда брать задачи

### ROADMAP.md — последовательность по дням

Открой `ROADMAP.md`, найди текущий день. Задачи помечены чекбоксами:
- `[ ]` — не сделано, можно отдать
- `[x]` или `✅` — готово

Каждый день имеет тип (настроено в глобальном AGENTS.md):
- **A-Day (Пн, Ср, Пт)** — OpenCode пишет код, ты ревьюишь
- **B-Day (Вт, Чт)** — ты пишешь, OpenCode только ревьюит и задаёт вопросы

### MVP.md — полные спеки компонентов

12 компонентов с готовым кодом (секции `### 1.` через `### 12.`):

| # | Компонент | Секция в MVP.md |
|---|-----------|-----------------|
| 1 | HybridWorkload CRD | `### 1. HybridWorkload CRD` |
| 2 | Configuration | `### 2. Configuration` |
| 3 | AWS Pricing Client | `### 3. AWS Pricing Client` |
| 4 | Proxmox Metrics Client | `### 4. Proxmox Metrics Client` |
| 5 | Decision Engine | `### 5. Decision Engine` |
| 6 | Karpenter NodePool Manager | `### 6. Karpenter NodePool Manager` |
| 7 | Controller Reconciler | `### 7. Controller Reconciler` |
| 8 | Main Entry Point (DI) | `### 8. Main Entry Point` |
| 9 | Structured Error Types | `### 9. Structured Error Types` |
| 10 | VPN Health Checker | `### 10. VPN Health Checker` |
| 11 | Admission Webhooks | `### 11. Admission Webhooks` |
| 12 | Cost Savings CLI | `### 12. Cost Savings CLI` |

## Как формулировать промпт

### Реализация компонента (A-Day)

```
Реализуй компонент "Decision Engine" по спецификации из MVP.md, секция "### 5. Decision Engine".

Файлы:
- internal/scheduler/engine.go — бизнес-логика
- internal/scheduler/provider.go — samber/do провайдер

Используй /samber-do skill для provider.go.
```

### Написать тесты

```
Напиши table-driven unit тесты для internal/scheduler/engine.go.

Файл: test/unit/decision_engine_test.go

Покрой кейсы:
- low utilization → proxmox
- high priority → aws
- hysteresis (на AWS, proxmox < 70%) → return to proxmox
- VPN down → error
- budget exceeded → pending

Не используй samber/do в тестах — создавай сервисы напрямую через конструктор.
```

### Code review (B-Day)

```
Сделай review файла internal/scheduler/engine.go.

Проверь по /go-expert:
- Ошибки обёрнуты с контекстом (fmt.Errorf + %w)
- Нет raw go func() без panic recovery
- Structured logging через slog

Проверь по /samber-do:
- samber/do только в provider.go (Golden Rule)
- Нет os.Getenv в провайдерах
```

### Добавить provider.go

```
Создай provider.go для пакета internal/cost/.

Сервис: AWSPricingClient (уже реализован в aws_pricing_client.go).
Зависимости: *slog.Logger, *config.Config (для AWSRegion).

Используй /samber-do skill. Checklist:
- closure-based config
- component logger с "component"="aws-pricing"
- ошибки обёрнуты с контекстом
```

## Пошаговый workflow

1. **Выбери задачу** — открой ROADMAP.md, найди ближайший `[ ]`
2. **Найди спеку** — открой MVP.md, найди соответствующую секцию `### N.`
3. **Сформулируй промпт** — укажи: какой компонент, какие файлы, какую секцию MVP.md читать
4. **Вызови skill** — добавь `/go-expert` или `/samber-do` если нужны паттерны
5. **Запусти** — вставь промпт в OpenCode
6. **Проверь результат**:
   - `go build ./...` — компилируется?
   - `go test -race ./...` — тесты проходят?
   - `golangci-lint run ./...` — линтер доволен?
   - `samber/do` только в `provider.go`?
7. **Коммит** — `git add . && git commit -m "feat: implement component-name"`

## Советы

**Работает хорошо:**
- Давать конкретную секцию MVP.md — там уже есть готовый код
- Один компонент за раз (1 промпт = 1-2 файла)
- Указывать имена файлов явно
- Вызывать `/samber-do` при создании provider.go — skill знает все правила

**Не работает:**
- "Реализуй весь проект" — слишком абстрактно
- "Сделай как считаешь нужным" — нужна конкретика
- Несколько несвязанных компонентов в одном промпте
