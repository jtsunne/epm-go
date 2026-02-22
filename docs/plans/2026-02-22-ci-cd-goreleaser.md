# CI/CD, GoReleaser и Homebrew Tap для epm-go

## Overview
- **Problem**: Репозиторий `jtsunne/epm-go` создан на GitHub, но нет CI, автоматических релизов и дистрибуции через Homebrew. Модульный путь в `go.mod` устарел (`github.com/jtsunne/epm-go`).
- **Solution**: Настроить GitHub Actions (тесты на PR/push, релиз на теге), goreleaser для кросс-компиляции (Linux/macOS × amd64/arm64), и Homebrew tap для `brew install jtsunne/tap/epm`.
- **Целевые платформы**: 4 бинарника — Linux amd64, Linux arm64, macOS amd64, macOS arm64.
- **Начальная версия**: v0.1.0

## Context
- **go.mod**: `github.com/jtsunne/epm-go`, Go 1.25.0 — модульный путь нужно обновить на `github.com/jtsunne/epm-go`
- **Entry point**: `cmd/epm/main.go` — `var version = "dev"` (строка 19), инжектится через `-X main.version=...`
- **Makefile**: уже использует `git describe --tags --always --dirty` для ldflags
- **Чистый Go, без CGO** — кросс-компиляция тривиальна
- **Лицензия**: MIT
- **Нет**: `.github/`, `.goreleaser.yaml`, тегов, remote'ов, CI/CD
- **`.gitignore`**: нет `dist/` и `/epm` — нужно добавить
- **Стоковый бинарник `epm`** (10MB) в корне проекта — untracked, нужно удалить

## Development Approach
- **Testing approach**: Regular (код → тесты)
- Каждый таск завершается верификацией
- Чистый Go без CGO — кросс-компиляция безопасна
- CRITICAL: все тесты должны проходить перед переходом к следующему таску

## Implementation Steps

### Task 1: Обновить модульный путь `dm` → `jtsunne`
- [x] Обновить `module` в `go.mod` (строка 1): `github.com/dm/epm-go` → `github.com/jtsunne/epm-go`
- [x] Заменить import path во всех `.go` файлах (22 файла: `cmd/epm/main.go`, все `internal/**/*.go`)
- [x] Обновить URL в `README.md` (go install, git clone, releases — 3 вхождения)
- [x] Обновить URL в `docs/plans/completed/*.md` (для консистентности)
- [x] Выполнить `go build ./...` — должно скомпилироваться
- [x] Выполнить `go test -race -count=1 ./...` — все тесты должны пройти
- [x] Убедиться что `grep -r 'github.com/dm/epm-go'` не находит совпадений

### Task 2: Обновить `.gitignore`
- [x] Добавить `dist/` (выходная директория goreleaser)
- [x] Добавить `/epm` (бинарник в корне проекта)
- [x] Удалить бинарник `epm` из корня проекта

**Файл**: `.gitignore`

### Task 3: Создать `.goreleaser.yaml`
- [x] Создать файл `.goreleaser.yaml` в корне проекта (содержимое ниже в Technical Details)
- [x] Установить goreleaser локально: `go install github.com/goreleaser/goreleaser/v2@latest`
- [x] Выполнить `goreleaser check` — валидация конфига должна пройти

### Task 4: Создать GitHub Actions CI workflow
- [x] Создать директорию `.github/workflows/`
- [x] Создать `.github/workflows/ci.yml` (содержимое ниже в Technical Details)
- [x] 3 параллельных джоба: test, lint (staticcheck), goreleaser config check

### Task 5: Создать GitHub Actions Release workflow
- [x] Создать `.github/workflows/release.yml` (содержимое ниже в Technical Details)
- [x] Workflow: тесты → goreleaser release (только при пуше тега `v*`)

### Task 6: Пушнуть на GitHub и проверить CI
- [ ] Добавить remote: `git remote add origin git@github.com:jtsunne/epm-go.git`
- [ ] Пушнуть main ветку: `git push -u origin main`
- [ ] Проверить что CI workflow запустился и прошёл в GitHub Actions

### Task 7: Верификация
- [ ] `goreleaser check` — валидация конфига
- [ ] `goreleaser release --snapshot --clean` — локальный dry-run
- [ ] Проверить что в `dist/` есть 4 архива: `epm_*_darwin_amd64`, `epm_*_darwin_arm64`, `epm_*_linux_amd64`, `epm_*_linux_arm64`
- [ ] Каждый архив содержит: `epm` (бинарник), `LICENSE`, `README.md`
- [ ] Выполнить `make test` — финальная проверка
- [ ] Выполнить `make lint` — все чисто

## Technical Details

### `.goreleaser.yaml`

```yaml
version: 2
project_name: epm

before:
  hooks:
    - go mod tidy

builds:
  - id: epm
    main: ./cmd/epm
    binary: epm
    env:
      - CGO_ENABLED=0
    goos:
      - linux
      - darwin
    goarch:
      - amd64
      - arm64
    ldflags:
      - -s -w -X main.version={{.Version}}

archives:
  - id: default
    formats:
      - tar.gz
    name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"
    files:
      - LICENSE
      - README.md

checksum:
  name_template: checksums.txt

changelog:
  sort: asc
  filters:
    exclude:
      - "^docs:"
      - "^test:"
      - "^ci:"
      - "^chore:"

release:
  github:
    owner: jtsunne
    name: epm-go

brews:
  - repository:
      owner: jtsunne
      name: homebrew-tap
      token: "{{ .Env.HOMEBREW_TAP_GITHUB_TOKEN }}"
    directory: Formula
    name: epm
    homepage: "https://github.com/jtsunne/epm-go"
    description: "Terminal dashboard for Elasticsearch cluster performance monitoring"
    license: "MIT"
    install: |
      bin.install "epm"
    test: |
      system "#{bin}/epm", "--version"
    commit_msg_template: "chore: update epm formula to {{ .Tag }}"
```

**Решения**:
- `CGO_ENABLED=0` — чистый Go, безопасная кросс-компиляция
- `-s -w` — stripped бинарники (~30% меньше)
- `-X main.version={{.Version}}` → `var version = "dev"` в `cmd/epm/main.go:19`
- `HOMEBREW_TAP_GITHUB_TOKEN` — отдельный PAT (default `GITHUB_TOKEN` не имеет доступа к другим репо)
- Тесты НЕ в `before.hooks` — запускаются отдельным шагом в release workflow

### `.github/workflows/ci.yml`

```yaml
name: CI

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

permissions:
  contents: read

jobs:
  test:
    name: Test
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - run: go mod download
      - run: go mod verify
      - run: go vet ./...
      - run: go test -race -count=1 ./...

  lint:
    name: Lint
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - uses: dominikh/staticcheck-action@v1
        with:
          version: latest

  goreleaser-check:
    name: GoReleaser Config
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - uses: goreleaser/goreleaser-action@v6
        with:
          version: "~> v2"
          args: check
```

### `.github/workflows/release.yml`

```yaml
name: Release

on:
  push:
    tags:
      - "v*"

permissions:
  contents: write

jobs:
  test:
    name: Test
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - run: go test -race -count=1 ./...

  release:
    name: Release
    needs: test
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - uses: goreleaser/goreleaser-action@v6
        with:
          version: "~> v2"
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          HOMEBREW_TAP_GITHUB_TOKEN: ${{ secrets.HOMEBREW_TAP_GITHUB_TOKEN }}
```

**Ключевые моменты**:
- `fetch-depth: 0` — goreleaser нужна полная git история для changelog
- `needs: test` — релиз только после прохождения тестов
- `contents: write` — для создания GitHub Release
- Два токена: `GITHUB_TOKEN` (автоматический) + `HOMEBREW_TAP_GITHUB_TOKEN` (ручной секрет)

## Post-Completion

**Создание Homebrew tap репозитория** (ручные шаги):
1. Создать публичный репозиторий `jtsunne/homebrew-tap` на GitHub с README и директорией `Formula/`
2. GitHub Settings → Developer settings → Fine-grained tokens → Generate new token
   - Имя: `goreleaser-homebrew-tap`
   - Repository access: Only select repositories → `jtsunne/homebrew-tap`
   - Permissions: Contents → Read and write
3. `github.com/jtsunne/epm-go` → Settings → Secrets → New secret: `HOMEBREW_TAP_GITHUB_TOKEN`

**Первый релиз**:
```bash
git tag -a v0.1.0 -m "Initial release"
git push origin v0.1.0
# GitHub Actions → goreleaser → 4 бинарника + Homebrew формула
```

**Проверка Homebrew**:
```bash
brew install jtsunne/tap/epm
epm --version
# → epm version 0.1.0
```

**Артефакты релиза**:
```
epm_0.1.0_darwin_amd64.tar.gz    # macOS Intel
epm_0.1.0_darwin_arm64.tar.gz    # macOS Apple Silicon
epm_0.1.0_linux_amd64.tar.gz     # Linux x86_64
epm_0.1.0_linux_arm64.tar.gz     # Linux ARM64
checksums.txt                     # SHA256
```
