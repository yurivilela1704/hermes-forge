# Go for Laravel Developers

Welcome to Go! Coming from Laravel (PHP), you'll find Go to be a different beast—simpler in many ways, but stricter in others. This guide uses the **Hermes Forge** codebase to explain key Go concepts through the lens of what you already know.

## 1. No Classes, Just Structs and Methods

In Laravel, everything is a class. In Go, we use `structs` to hold data and `methods` to define behavior.

**Laravel:**
```php
class Store {
    protected $outputDir;
    public function saveNote($event, $content) { ... }
}
```

**Go (from `internal/storage/storage.go`):**
```go
type Store struct {
    OutputDir string
}

func (s *Store) SaveNote(event webhook.Event, content string) (NoteMetadata, error) {
    // 's' is like '$this'
    // 'SaveNote' is a method on the '*Store' pointer
}
```

## 2. Interfaces are Implicit

In PHP, a class must `implements ProviderInterface`. In Go, if a struct has the methods defined in an `interface`, it **automatically** implements it.

**Go (from `internal/llm/provider.go`):**
```go
type Provider interface {
    Generate(ctx context.Context, prompt string) (Generation, error)
    Model() string
}
```
Any struct (like `OpenAIProvider`) that has these two methods is a `Provider`. No `implements` keyword needed!

## 3. Error Handling: No Exceptions

Laravel uses `try-catch` blocks. Go returns errors as values. You'll see this pattern everywhere:

```go
res, err := store.SaveNote(event, content)
if err != nil {
    // Handle the error (log it, return it, etc.)
    return err
}
```
It feels verbose at first, but it makes the code very predictable. No "hidden" exceptions bubbling up.

## 4. Implicit Dependency Injection

Laravel has a powerful Service Container. In Go, we usually do "Poor Man's Dependency Injection" by passing dependencies manually (often in a "New" function).

**Go (from `cmd/server/main.go`):**
```go
func main() {
    store := initStore(cfg.OutputDir)
    gen := initGenerator(cfg.Template)
    
    // Pass them to handlers
    http.HandleFunc("/api/notes", listNotesHandler(cfg.UIToken, store))
}
```

## 5. Composition over Inheritance

Go doesn't have class inheritance. Instead, we "embed" structs.

```go
type SpecialStore struct {
    Store // Embeds Store's fields and methods
    ExtraField string
}
```

## 6. Public vs. Private (Casing)

Forget `public` and `private` keywords.
- Starts with **Uppercase**: Public (Exported) – can be used by other packages.
- Starts with **Lowercase**: Private (Unexported) – only used inside the same package.

Example: `SaveNote` is public, `slugify` is private.

## 7. Typing and Pointers

Go is statically typed. `*Store` means a "pointer to a Store".
- Pass by value: `func(s Store)` – makes a copy.
- Pass by pointer: `func(s *Store)` – shares the same memory (like PHP objects by default).

## 8. Modules and Packages

Laravel uses Composer. Go uses `go.mod`.
- A directory is a `package`.
- Files in the same directory must share the same `package name`.
- You import packages, not individual files.

## 9. Concurrency (Goroutines)

This is Go's superpower. You can run a function in the background by just typing `go`:

```go
go someBackgroundWork() // This doesn't block!
```
Laravel uses Queues (Redis/Database) for this. Go has them built-in (goroutines and channels).

## Summary Table

| Concept | Laravel (PHP) | Go |
| :--- | :--- | :--- |
| **Data Container** | Class | Struct |
| **Contract** | Interface (`implements`) | Interface (Implicit) |
| **Errors** | Exceptions (`throw/catch`) | Multi-value return (`val, err`) |
| **Dependencies** | Service Container / Type Hinting | Manual passing in constructors |
| **Background Tasks** | Queues / Workers | Goroutines |
| **Visibility** | `public`/`private`/`protected` | Uppercase / Lowercase names |
