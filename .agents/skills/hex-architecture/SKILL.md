---
name: hex-architecture
description: Guide for designing and implementing business logic using Hexagonal Architecture (Ports and Adapters) in the Sattchel repository. Use when creating new components or adding business features.
---

# Hexagonal Architecture Guidelines

All business logic in Sattchel follows **Hexagonal Architecture (Ports and Adapters)**. Domain logic must remain isolated from driving adapters (CLI/UI/APIs) and driven adapters (files, databases, external services).

---

## Component Folder Layout

Each functional area (e.g., `tracker`, `optimizely`) is self-contained under `internal/<component>/`:

```text
internal/<component>/
├── core/
│   ├── domain.go      # Entities and core domain models
│   ├── ports.go       # Abstract interfaces (driven repositories & driving ports)
│   ├── service.go     # Service layer implementing business use cases
│   └── *_test.go      # Unit tests for core logic
└── adapters/
    ├── driving/       # CLI commands, HTTP API handlers, TUI integration
    └── driven/        # Persistence adapters (file storage, database, external APIs)
```

> **Note on Go Package Flattening:** Keep the `core` folder flat (`package core` inside `internal/<component>/core`) to prevent circular import dependencies in Go.

---

## Core Rules & Responsibilities

### 1. Core Domain (`internal/<component>/core/`)

- Contains domain structs, business validation, and core entities.
- **No business logic in adapters:** Driving adapters (CLI/UI) and driven adapters (file/API) must not contain business rules.
- **Zero dependencies on adapters:** `core` must never import anything from `adapters/`.

### 2. Ports (`internal/<component>/core/ports.go`)

- Ports are Go interfaces defined inside `core/`.
- Group interfaces by **architectural responsibility**, not per individual domain model.
- Example Driven Port:

  ```go
  type Repository interface {
      SaveProject(ctx context.Context, project *Project) error
      GetProject(ctx context.Context, id string) (*Project, error)
  }
  ```

### 3. Service Layer (`internal/<component>/core/service.go`)

- The `Service` struct holds dependencies on driven ports (interfaces) and implements use cases.
- Operates like transaction scripts executing specific user/system actions.
- Example:

  ```go
  type Service struct {
      repo Repository
  }

  func NewService(repo Repository) *Service {
      return &Service{repo: repo}
  }
  ```

### 4. Adapters (`internal/<component>/adapters/`)

- **Driving Adapters:** CLI commands (Cobra), HTTP handlers, or TUI components. They accept user input, call `Service` methods, and format output.
- **Driven Adapters:** Implement `core/ports.go` interfaces (e.g., file storage in `adapters/driven/file.go`).

### 5. DTOs and Data Boundaries

- Do not expose or leak internal adapter structures into `core`.
- If the UI layer needs aggregated data or complex view states, return explicit DTOs from `service.go` or keep logic inside `core` rather than adding logic to CLI handlers.

### 6. Error Naming Convention

- **Mandatory:** Error variables MUST always be named `err` (e.g., `if err != nil`). Never use custom names like `runErr`, `getGoalsErr`, or `saveErr`.
