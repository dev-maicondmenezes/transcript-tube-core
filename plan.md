# Plano de Execução — `transcript_tube`

**Base:** `spec.md` v1.0
**Estratégia:** Core + Adaptadores, em **três repositórios independentes** (multi-repo)
**Data:** 2026-11-07 (revisado em 2026-11-08)
**Status:** Fase 1 concluída no monorepo original (`v0.1.0-core`, capítulos em `df5aa88`); migração para multi-repo e Fase 2 em andamento.
**CLI mnemonic:** `tyot` (abreviação de *Transcript YOuTube*) — binário, pacote `.deb`, formula Homebrew, man page e prefixo de env vars.
**Tooling:** `gh` (GitHub CLI) autenticado é usado para **criar, configurar e gerenciar** os três repositórios e seus workflows/secrets.

---

## 1. Visão geral do plano

O desenvolvimento será dividido em **3 fases independentes e verificáveis**, entregues em **repositórios Git separados** para permitir versionamento, distribuição e CI/CD independentes:

```
┌─────────────────────────────────────────────────────────────┐
│ transcript-tube-core  (biblioteca Go, contrato congelado)   │
└──────────────┬──────────────────────────────┬───────────────┘
               │                              │
               ▼                              ▼
┌──────────────────────────┐     ┌─────────────────────────────────┐
│ transcript-tube-cli      │     │ transcript-tube-extension       │
│ binário: tyot            │     │ (nm-host Go + extensão Chrome)  │
│ apt/brew/gh releases     │     │ Chrome Web Store (enhancement)  │
└──────────────────────────┘     └─────────────────────────────────┘
```

| Fase | Repositório | Escopo | Entregável verificável |
| :--- | :--- | :--- | :--- |
| **1 — Core** | `transcript-tube-core` | Tipos, validação, metadados, transcrição, capítulos, Markdown | Módulo importável, tag SemVer `v0.2.0` (migração de `v0.1.0-core`), CI verde |
| **2 — CLI** | `transcript-tube-cli` | Binário `tyot`, config flag/env/file, validação obrigatória de `YOUTUBE_API_KEY`, `--help` + man page, distribuição apt/brew/gh, CHANGELOG | `tyot` instalável via `apt`, `brew` e `gh release download`, com man page |
| **3 — Chrome Extension** | `transcript-tube-extension` | `nm-host` (Go) + extensão (JS/HTML/CSS), Native Messaging, popup, E2E | Extensão carregável, host instalável por script, E2E com CLI |

### 1.1 Arquitetura multi-repo

| Repositório | Conteúdo | Módulo Go | Versionamento |
| :--- | :--- | :--- | :--- |
| `transcript-tube-core` | `core/`, `examples/core-demo/`, `docs/architecture.md`, `docs/core.md`, `docs/testing.md`, `spec.md`, `plan.md` | `github.com/dev-maicondmenezes/transcript-tube-core` | SemVer estrito (`v0.x.y`) |
| `transcript-tube-cli` | `cmd/tyot/`, `internal/`, `docs/cli.md`, `docs/man/`, `.goreleaser.yaml`, `CHANGELOG.md` | `github.com/dev-maicondmenezes/transcript-tube-cli` | SemVer estrito; releases via tag |
| `transcript-tube-extension` | `cmd/nm-host/`, `extension/`, `scripts/`, `docs/extension.md` | `github.com/dev-maicondmenezes/transcript-tube-extension` | SemVer estrito; releases via tag |

**Regras de dependência:**

- `core` **não importa** `cli` nem `extension`.
- `cli` e `extension` importam `core` como módulo versionado (`require` no `go.mod`).
- Durante o desenvolvimento de `cli`/`extension`, é aceitável usar `replace` local; **antes da tag de release**, remover `replace` e fixar `require github.com/dev-maicondmenezes/transcript-tube-core vX.Y.Z`.
- Nenhum segredo (API key, credenciais) é versionado em qualquer repositório.

### 1.2 Uso do `gh` (GitHub CLI) como ferramenta central

Assume-se que `gh` está instalado e autenticado (`gh auth status` verde). Comandos usados ao longo do plano:

| Operação | Comando(s) |
| :--- | :--- |
| Criar repositórios | `gh repo create <owner>/<name> --public --description "..."` |
| Clonar | `gh repo clone <owner>/<name>` |
| Editar configurações | `gh repo edit --enable-issues=false --enable-wiki=false` |
| Proteger branch `main` | `gh api -X PUT repos/<owner>/<name>/branches/main/protection ...` |
| Configurar secrets | `gh secret set <NOME> --repo <owner>/<name>` |
| Configurar variáveis | `gh variable set <NOME> --repo <owner>/<name>` |
| Listar/observar workflows | `gh run list`, `gh run watch`, `gh workflow list` |
| Disparar workflow manual | `gh workflow run <file> --ref main` |
| Criar releases | `gh release create vX.Y.Z --generate-notes` |
| Upload de artefatos | `gh release upload vX.Y.Z <files>` |
| Consultar/badges | `gh api repos/<owner>/<name>` |
| Configurar Dependabot | `.github/dependabot.yml` versionado (gh não gerencia diretamente) |

Toda interação com o GitHub (criação, secrets, releases, branches protegidas) passa por `gh` neste plano. Nada depende de cliques na UI.

### 1.3 Contrato público do Core (congelado em `v0.2.0`)

A tag `v0.2.0` do `transcript-tube-core` congela o contrato abaixo. Alterações seguem SemVer:

- **`v0.2.x`** — correções compatíveis (bug fix).
- **`v0.3.0`** — adições compatíveis (novos campos, novas funções).
- **`v1.0.0`** — estabilidade declarada após as Fases 2 e 3.

```go
type VideoData struct {
    ID          string
    Title       string
    Channel     string
    Description string
    PublishedAt time.Time
    Duration    string
    ViewCount   uint64
    LikeCount   uint64
    Transcript  []TranscriptSegment
    Chapters    []Chapter
}

type TranscriptSegment struct {
    OffsetMs int64
    Text     string
}

type Chapter struct {
    OffsetMs int64
    Title    string
}

type Options struct {
    Lang                   string // ex.: "pt"; vazio = preferência "pt" com fallback
    AllowPartialTranscript bool
}

func Extract(ctx context.Context, videoID string) (*VideoData, error)
func ExtractWithOptions(ctx context.Context, videoID string, opts Options) (*VideoData, error)
func ToMarkdown(data *VideoData) string
func ValidateVideoID(videoID string) error
```

**Erros sentinela** (verificáveis com `errors.Is`):

```go
ErrInvalidVideoID
ErrMissingAPIKey
ErrVideoNotFound
ErrMetadataUnavailable
ErrTranscriptUnavailable
```

**Semântica do modo parcial:** com `AllowPartialTranscript: true`, quando a transcrição falha mas os metadados foram obtidos, a função retorna `VideoData` preenchido **e** um erro que envolve `ErrTranscriptUnavailable`.

### 1.4 Princípios de implementação

- `core` não importa `os/exec`, terminal, Chrome, JS ou pacotes de `cmd/`.
- Nenhum scraping do HTML do YouTube; somente YouTube Data API v3, endpoints
  InnerTube/timedtext e o endpoint InnerTube `next` para capítulos.
- Toda chamada de rede recebe `context.Context`.
- Erros sempre contextualizados (`metadata: ...`, `transcript: ...`).
- Dependências externas encapsuladas atrás de interfaces/adaptadores.
- Nenhum segredo em código, extensão, logs ou arquivos versionados.
- **A API key do YouTube é lida exclusivamente da variável de ambiente `YOUTUBE_API_KEY`** (sem flag `--api-key`, sem campo em config file).

### 1.5 Definition of Done global

Uma fase só é considerada concluída quando:

- [ ] `go build ./...`, `go vet ./...` e `gofmt -l .` estão limpos no repositório da fase.
- [ ] `go test ./...` passa **offline** no repositório da fase.
- [ ] CI/CD do repositório está verde (verificado com `gh run list`).
- [ ] Critérios de aceite específicos foram executados e registrados.
- [ ] Documentação da fase foi criada/atualizada (`README.md` do repositório da fase).
- [ ] `CHANGELOG.md` atualizado (a partir da Fase 2).
- [ ] Tag SemVer (`vX.Y.Z`) criada e release publicada (GitHub Releases, via `gh release view`).
- [ ] Branch `main` protegida (`gh api repos/<owner>/<name>/branches/main/protection`).
- [ ] Nenhuma fase anterior sofreu regressão.

---

## 2. Fase 1 — Core (concluída; migração em curso)

### 2.1 Objetivo e escopo

Construir a fundação do projeto e o pacote `core` completo: tipos, validação de `videoID`, cliente de metadados, cliente de transcrição em cascata, capítulos (InnerTube `next` + fallback da descrição), orquestração paralela e formatação Markdown.

**Status:** concluída em 2026-11-08 no monorepo original (`v0.1.0-core`; capítulos em `df5aa88`). A migração para o repositório `transcript-tube-core` está descrita na seção 3.2.

### 2.2 Entregas da Fase 1 (registro)

- `core`: `Extract`, `ExtractWithOptions`, `ToMarkdown`, `ValidateVideoID`,
  `VideoData`, `TranscriptSegment`, `Chapter`, `Options` e erros sentinela.
- `core/metadata`: YouTube Data API v3 com `part=snippet,statistics,contentDetails`.
- `core/transcript`: cascata InnerTube → `timedtext` JSON3 → `timedtext` SRV3.
- `core/chapters`: cliente InnerTube `next` para capítulos oficiais/automáticos.
- `core/chapters.go`: parser de capítulos por timestamps da descrição (fallback).
- Orquestração paralela com `sync.WaitGroup`, modo parcial e `errors.Join`.
- Markdown determinístico com capítulos `###` dentro de `## Transcrição`.
- `examples/core-demo`, golden tests, testes com `httptest`/fakes e documentação.
- Cobertura: `core` ~84%, `core/chapters` ~85%, `durfmt` ~91%, `metadata` ~88%, `transcript` ~69%.

Detalhes históricos completos da Fase 1 (descobertas, decisões, commits) estão preservados no repositório `transcript-tube-core` em `docs/`.

### 2.3 Commits e tags de referência

| Referência | Conteúdo |
| :--- | :--- |
| `2c8a9d8` (tag `v0.1.0-core`) | Fase 1 inicial |
| `e2ad652` | Capítulos a partir dos timestamps da descrição |
| `df5aa88` | Fonte prioritária de capítulos via InnerTube `next` |

> Essas tags existem no **monorepo original**. Na migração para `transcript-tube-core`, o mesmo conteúdo é re-tagueado como `v0.2.0`.

---

## 3. Fase 2 — CLI (`tyot`)

### 3.1 Objetivo e escopo

Criar o repositório independente **`transcript-tube-cli`**, que produz o binário **`tyot`**, com:

- consumo de `github.com/dev-maicondmenezes/transcript-tube-core` como dependência versionada;
- configuração via **flags** + **variáveis de ambiente** + **arquivo de configuração** (sem API key no arquivo — ver 3.5.5);
- **validação robusta** de `YOUTUBE_API_KEY` em qualquer invocação que não seja do tipo "help" (ver 3.5.5);
- `--help` resumido e **man page completa** (padrão Linux);
- distribuição via **APT** (Debian/Ubuntu), **Homebrew** (macOS) e **GitHub Releases**;
- `CHANGELOG.md` gerado a partir de Conventional Commits;
- **CI/CD próprio** (GitHub Actions), criado e gerenciado via `gh`.

**Fora de escopo:** extensão Chrome, Native Messaging e mudanças no contrato do `core`.

**Pré-requisitos:**

- Fase 1 concluída e `transcript-tube-core` publicado com tag SemVer `v0.2.0` (ver 3.2).
- `gh` instalado e autenticado (`gh auth status`).
- Repositório `homebrew-tap` criável via `gh repo create`.
- GPG key (opcional) para assinar pacotes APT.
- Ferramentas locais: Go 1.23, GoReleaser v2, `dpkg-deb`, `lintian`, `man`, `git-cliff` (opcional).

### 3.2 Migração do core para repositório próprio (via `gh`)

**Por que migrar antes da Fase 2:** o `go.mod` do CLI precisa apontar para um módulo importável com versão SemVer válida. Tags como `v0.1.0-core` não são reconhecidas pelo `go get`.

**Passos:**

1. Criar o repositório no GitHub com `gh`:
   ```bash
   gh repo create dev-maicondmenezes/transcript-tube-core \
     --public \
     --description "Core library: extract YouTube metadata and transcript to Markdown." \
     --disable-wiki \
     --confirm
   ```
2. Clonar ou copiar do monorepo original:
   ```bash
   gh repo clone dev-maicondmenezes/transcript-tube-core
   cd transcript-tube-core
   # copiar do monorepo: core/, examples/core-demo/, docs/, spec.md, plan.md,
   # Makefile, .gitignore, LICENSE, .env.example, scripts/check-env.sh
   ```
3. Atualizar `go.mod`:
   ```bash
   go mod edit -module github.com/dev-maicondmenezes/transcript-tube-core
   grep -rl "github.com/dev-maicondmenezes/transcript_tube" --include='*.go' . \
     | xargs sed -i 's#github.com/dev-maicondmenezes/transcript_tube#github.com/dev-maicondmenezes/transcript-tube-core#g'
   ```
4. Reexecutar `make check` e `go test -race ./core/...` para confirmar paridade.
5. Configurar branch protection via `gh`:
   ```bash
   gh api -X PUT repos/dev-maicondmenezes/transcript-tube-core/branches/main/protection \
     -F required_status_checks='{"strict":true,"contexts":["ci/test"]}' \
     -F enforce_admins=true \
     -F required_pull_request_reviews='{"required_approving_review_count":1}' \
     -F restrictions=null
   ```
6. Commitar, taguear e publicar:
   ```bash
   git tag v0.2.0
   git push origin main --tags
   gh release view v0.2.0 --repo dev-maicondmenezes/transcript-tube-core
   ```
7. Arquivar o monorepo original (via `gh`):
   ```bash
   gh repo archive dev-maicondmenezes/transcript_tube --yes
   # ou, se preferir deletar: gh repo delete dev-maicondmenezes/transcript_tube --yes
   ```

**Versionamento na migração:** primeira tag pública do módulo é **`v0.2.0`** (conteúdo de `df5aa88`). Não criar `v0.1.0` retroativo.

### 3.3 Setup do ambiente

#### Ferramentas

| Ferramenta | Versão mínima | Instalação |
| :--- | :--- | :--- |
| Go | 1.23 | sistema |
| Git | 2.x | sistema |
| **`gh`** | **2.40+** | **sistema; `gh auth status` deve estar verde** |
| Make | 4.x | sistema (ou comandos Go equivalentes) |
| GoReleaser | v2 | `go install github.com/goreleaser/goreleaser/v2@latest` |
| git-cliff | 2.x (opcional) | binário de [git-cliff.org](https://git-cliff.org) |
| dpkg-deb, lintian | — | `apt install dpkg-dev lintian` |
| man | — | `apt install man-db` |
| Node.js | 20 | apenas para testes E2E manuais (opcional) |

#### Variáveis de ambiente de desenvolvimento

`.env.example` (no repo `transcript-tube-cli`, não versionar `.env`):

```bash
YOUTUBE_API_KEY=               # única fonte da API key (obrigatória para extração)
GITHUB_TOKEN=                  # usado pelo GoReleaser em release local (opcional)
HOMEBREW_TAP_GITHUB_TOKEN=     # PAT com write no repo homebrew-tap
```

#### Inicialização do repositório (via `gh`)

```bash
# 1. Criar o repositório no GitHub
gh repo create dev-maicondmenezes/transcript-tube-cli \
  --public \
  --description "tyot — extract YouTube metadata and transcript to Markdown (CLI)." \
  --disable-wiki \
  --confirm

# 2. Clonar
gh repo clone dev-maicondmenezes/transcript-tube-cli
cd transcript-tube-cli

# 3. Inicializar o módulo
go mod init github.com/dev-maicondmenezes/transcript-tube-cli
go get github.com/dev-maicondmenezes/transcript-tube-core@v0.2.0

mkdir -p cmd/tyot cmd/gen-man internal/cli internal/config internal/url

# 4. Proteger a branch main
gh api -X PUT repos/dev-maicondmenezes/transcript-tube-cli/branches/main/protection \
  -F required_status_checks='{"strict":true,"contexts":["ci/test"]}' \
  -F enforce_admins=true \
  -F required_pull_request_reviews='{"required_approving_review_count":1}' \
  -F restrictions=null
```

> Durante o desenvolvimento, é aceitável:
> ```bash
> go mod edit -replace github.com/dev-maicondmenezes/transcript-tube-core=../transcript-tube-core
> ```
> **Antes da tag de release**, remover: `go mod edit -dropreplace github.com/dev-maicondmenezes/transcript-tube-core` e fixar `vX.Y.Z`.

#### Configuração do OpenCode + DeepSeek

Reaproveitar `AGENTS.md` do core como base, adicionando:

- Regra: **nenhum `replace` local no `go.mod` na branch `main`**.
- Regra: **o CLI nunca loga em `stdout`**; `stdout` é reservado para o Markdown.
- Regra: **toda flag deve ter help curto e menção no man page**.
- Regra: **o binário se chama `tyot`; o pacote principal fica em `cmd/tyot/`**.
- Regra: **nenhuma configuração de API key fora de `YOUTUBE_API_KEY`** (sem flag, sem config file).
- Comandos de verificação: `make check`, `make release-dry-run`.

### 3.4 Estrutura de arquivos do repositório CLI

```text
transcript-tube-cli/
├── go.mod
├── go.sum
├── cmd/
│   ├── tyot/
│   │   └── main.go            # os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
│   └── gen-man/
│       └── main.go            # gera docs/man/tyot.1 + completions
├── internal/
│   ├── cli/
│   │   ├── root.go            # cobra root + PersistentPreRunE (validação de env)
│   │   ├── extract.go         # subcomando padrão (extração)
│   │   ├── version.go         # --version
│   │   ├── config.go          # carregamento flag → env → file
│   │   ├── env.go             # validação robusta de YOUTUBE_API_KEY
│   │   ├── env_test.go
│   │   ├── output.go          # stdout vs arquivo (atômico)
│   │   └── exit.go            # códigos de saída
│   ├── config/
│   │   ├── config.go
│   │   └── config_test.go
│   └── url/
│       ├── url.go
│       └── url_test.go
├── docs/
│   ├── cli.md
│   ├── man/
│   │   └── tyot.1
│   ├── completions/
│   │   ├── tyot.bash
│   │   ├── tyot.zsh
│   │   ├── _tyot
│   │   └── tyot.fish
│   └── examples/
│       └── config.yaml
├── .github/
│   ├── workflows/
│   │   ├── ci.yml
│   │   └── release.yml
│   └── dependabot.yml
├── .goreleaser.yaml
├── .git-cliff.toml
├── CHANGELOG.md
├── README.md
├── LICENSE
├── Makefile
└── .env.example
```

### 3.5 Features do CLI

#### 3.5.1 Configuração (precedência flag → env → file → default)

**Flags globais:**

| Flag | Curto | Tipo | Default | Descrição |
| :--- | :--- | :--- | :--- | :--- |
| `--help` | `-h` | bool | — | Mostra ajuda resumida. |
| `--version` | `-V` | bool | — | Mostra versão, commit e data de build. |
| `--config` | — | path | `$XDG_CONFIG_HOME/tyot/config.yaml` | Arquivo de configuração. |
| `--lang` | `-l` | string | `pt` | Idioma preferido da transcrição. |
| `--output` | `-o` | path | `stdout` | Arquivo de saída (Markdown). |
| `--timeout` | — | duration | `60s` | Timeout total da extração. |
| `--allow-partial` | — | bool | `false` | Retorna Markdown parcial se a transcrição falhar. |
| `--no-chapters` | — | bool | `false` | Desabilita headings de capítulos no Markdown. |
| `--quiet` | `-q` | bool | `false` | Suprime logs de progresso em `stderr`. |

> **Não existe `--api-key`.** A API key é configurada apenas via variável de ambiente `YOUTUBE_API_KEY` (ver 3.5.5).

**Variáveis de ambiente:**

| Variável | Precedência | Uso |
| :--- | :--- | :--- |
| `YOUTUBE_API_KEY` | **única fonte** | API key da YouTube Data API v3. Obrigatória para qualquer extração. |
| `TYOT_LANG` | 2 | Equivale a `--lang`. |
| `TYOT_TIMEOUT` | 2 | Equivale a `--timeout`. |
| `TYOT_CONFIG` | 2 | Caminho alternativo de config. |
| `NO_COLOR` | — | Desabilita cores. |

**Arquivo de configuração (YAML):**

Localização (via `os.UserConfigDir()`):

- Linux: `$XDG_CONFIG_HOME/tyot/config.yaml` (`~/.config/tyot/...` por padrão).
- macOS: `~/Library/Application Support/tyot/config.yaml`.
- Windows: `%AppData%\tyot\config.yaml`.

Estrutura:

```yaml
# Exemplo: docs/examples/config.yaml
default_lang: "pt"
default_timeout: "60s"
allow_partial: false
no_chapters: false
```

> **Não existe campo `youtube_api_key`.** O arquivo de config **não pode** conter a API key; se um campo `youtube_api_key` for encontrado, o loader retorna erro explícito (`exit 4`) instruindo a remover o campo e usar `YOUTUBE_API_KEY`.

Regras:

- Arquivo ausente não é erro; apenas aplica defaults.
- Campos desconhecidos: **erro** (fail-fast).
- O loader **rejeita explicitamente** qualquer chave relacionada a API key (`youtube_api_key`, `api_key`, `yt_api_key`) para evitar configuração insegura.

#### 3.5.2 `--help` resumido

```text
tyot — extrai metadados e transcrição do YouTube para Markdown

Uso:
  tyot <URL-ou-ID> [flags]
  tyot --help
  tyot --version

Exemplos:
  tyot dQw4w9WgXcQ
  tyot "https://youtu.be/dQw4w9WgXcQ" -o out/rick.md
  tyot "https://www.youtube.com/watch?v=dQw4w9WgXcQ" --lang en

Flags:
  -l, --lang string           Idioma da transcrição (default "pt")
  -o, --output string         Arquivo de saída (default: stdout)
      --timeout duration      Timeout total (default 60s)
      --allow-partial         Retorna Markdown parcial se transcrição falhar
      --no-chapters           Desabilita capítulos no Markdown
  -q, --quiet                 Suprime logs de progresso
      --config string         Arquivo de configuração
  -h, --help                  Mostra esta ajuda
  -V, --version               Mostra versão

Requisitos:
  Variável de ambiente YOUTUBE_API_KEY definida.
  Como obter: https://console.cloud.google.com/apis/credentials

Documentação completa: man tyot
Reportar bugs: https://github.com/dev-maicondmenezes/transcript-tube-cli/issues
```

#### 3.5.3 Man page completa

Gerada por `cobra/doc.GenManTree` a partir do comando raiz `tyot` (fonte: `cmd/gen-man/main.go`). O arquivo gerado é `docs/man/tyot.1`.

Instalação nos pacotes:
- `.deb`: `/usr/share/man/man1/tyot.1` (via `nfpm`).
- `brew`: `man1.install "docs/man/tyot.1"` no `install` da formula.

Regeneração local:

```bash
make man
# → docs/man/tyot.1
man -l docs/man/tyot.1
```

#### 3.5.4 Códigos de saída

```go
const (
    exitOK             = 0   // sucesso
    exitArgument       = 1   // URL/ID/flag inválidos
    exitExtraction     = 2   // API key ausente, rede, API, transcrição, timeout
    exitFileWrite      = 3   // falha ao criar diretório/arquivo
    exitConfig         = 4   // config file inválido (inclui tentativa de API key no arquivo)
)
```

#### 3.5.5 Validação robusta de `YOUTUBE_API_KEY` (obrigatória)

**Regra de ouro:** qualquer invocação que **não seja do tipo "help"** deve validar `YOUTUBE_API_KEY` **antes** de qualquer outra lógica de negócio.

**O que é "help-type" (isentos da validação):**

- `tyot --help` / `tyot -h`
- `tyot --version` / `tyot -V`
- `tyot completion <shell>` (se implementado como subcomando)
- `tyot help` (se implementado; opcional)
- Geração local de man page/completions (`cmd/gen-man` — não é o binário de usuário)

**O que NÃO é "help-type" (validação sempre executada):**

- `tyot <URL>` / `tyot <ID>`
- `tyot` sem argumentos (mostra help, mas ainda valida para alertar o usuário antes)
- `tyot -o file.md <URL>`
- Qualquer combinação de flags que leve à execução de extração
- Erros de argumento (URL inválida, flag desconhecida) **também** passam primeiro pela validação de env

**Onde acontece:** `PersistentPreRunE` do comando raiz em `internal/cli/root.go`, com a lógica isolada em `internal/cli/env.go`.

**Contrato da função:**

```go
// ValidateEnv verifica pré-requisitos de ambiente para execução real do CLI.
// Retorna nil quando OK ou um *EnvError com mensagem amigável e código de saída.
func ValidateEnv(getenv func(string) string) error
```

**Comportamento:**

1. Lê `YOUTUBE_API_KEY` do ambiente.
2. Se **vazia ou só espaços** → retorna erro com mensagem completa (ver abaixo) e exit code `2`.
3. Se **presente** → retorna `nil` (segue o fluxo normal).
4. Opcional (pós-`v0.1.0`): validar formato mínimo (ex.: tamanho >= 20 caracteres) e emitir **warning** em `stderr` sem bloquear; o erro real ainda virá do `core` (`ErrMissingAPIKey` / 403 da API).

**Mensagem de erro (padrão, exibida em `stderr`):**

```text
erro: variável de ambiente YOUTUBE_API_KEY não está definida.

O tyot precisa da chave da YouTube Data API v3 para extrair metadados.
Defina-a antes de executar:

  export YOUTUBE_API_KEY="sua-chave"
  tyot <URL-ou-ID>

Como obter uma chave:
  1. Acesse https://console.cloud.google.com/apis/credentials
  2. Ative a YouTube Data API v3 no projeto.
  3. Crie uma credencial do tipo "Chave de API".
  4. Restrinja a chave à YouTube Data API v3.

A ajuda (tyot --help) e a versão (tyot --version) funcionam sem a chave.

Código de saída: 2
```

**Regras adicionais:**

- A mensagem **nunca** imprime o valor da variável (mesmo que exista).
- A mensagem é **localizável** (i18n pode vir em versão futura); PT-BR como padrão.
- Se o usuário tentar rodar sem env **e** com argumentos inválidos, a mensagem de env vence (fail-fast na causa raiz mais comum).
- A validação **não** é executada dentro do `core`; o `core` mantém seu próprio `ErrMissingAPIKey`, mas o CLI já filtra antes, dando mensagem mais rica.

**Teste obrigatório (T2.9 expandido):**

- `tyot dQw4w9WgXcQ` sem env → exit 2, mensagem contém "YOUTUBE_API_KEY" e link do console.
- `tyot --help` sem env → exit 0, mostra help (não valida).
- `tyot --version` sem env → exit 0, mostra versão (não valida).
- `tyot -o file.md` sem env → exit 2 (valida antes de reclamar de URL ausente).
- `tyot dQw4w9WgXcQ` com env definida → segue o fluxo normal.
- `tyot dQw4w9WgXcQ` com env vazia (`export YOUTUBE_API_KEY=""`) → exit 2.
- `tyot dQw4w9WgXcQ` com env só com espaços → exit 2.

### 3.6 Tarefas de implementação

#### C1 — Repositório e esqueleto (via `gh`)

```bash
gh repo create dev-maicondmenezes/transcript-tube-cli \
  --public \
  --description "tyot — extract YouTube metadata and transcript to Markdown (CLI)." \
  --disable-wiki --confirm
gh repo clone dev-maicondmenezes/transcript-tube-cli
cd transcript-tube-cli
go mod init github.com/dev-maicondmenezes/transcript-tube-cli
go get github.com/dev-maicondmenezes/transcript-tube-core@v0.2.0
```

- `Makefile`, `.gitignore`, `.env.example`, `LICENSE` (MIT).
- `README.md` inicial.
- Branch `main` protegida via `gh api` (ver 3.3).

#### C2 — Framework de CLI

Adotar **`github.com/spf13/cobra`**:

- Gera `--help` padrão, subcomandos e flags tipadas.
- Gera man page (`doc.GenManTree`) e shell completions.
- Padrão do ecossistema.

O comando raiz é `tyot`; `cmd/tyot/main.go` apenas chama `cli.Execute()`.

#### C3 — Configuração (flag → env → file)

- `internal/config/config.go`: struct `Config` com os campos de 3.5.1 (**sem** campo de API key).
- Loader:
  1. defaults;
  2. arquivo (YAML, via `gopkg.in/yaml.v3`) — **rejeita** qualquer chave relacionada a API key;
  3. env vars;
  4. flags.
- Retornar erro claro (`exitConfig = 4`) para YAML inválido, campo desconhecido ou campo de API key.

#### C4 — Validação de env (`internal/cli/env.go`)

- Implementar `ValidateEnv(getenv func(string) string) error` com `EnvError`.
- Registrar em `PersistentPreRunE` do root, checando se o comando atual está na lista de isenções (help, version, completion).
- Mensagem em `stderr` conforme 3.5.5.
- Sempre retornar exit 2 em falha.

#### C5 — Subcomando `extract` (padrão)

- Aceitar `URL`/`ID` como argumento posicional.
- Se nenhum argumento e `stdin` for TTY: mostrar ajuda e sair com `exitArgument`.
- Se `stdin` **não** for TTY: ler URL do `stdin`.
- Validação de URL via `internal/url`.
- Chamar `core.ExtractWithOptions(ctx, id, core.Options{Lang, AllowPartialTranscript})`.
- Respeitar `--no-chapters`.
- Timeout: `context.WithTimeout(ctx, cfg.Timeout)`.

#### C6 — Saída

- `stdout`: Markdown puro. **Nenhum log em stdout.**
- `-o arquivo`: escrita atômica (arquivo temporário + `os.Rename`), `MkdirAll(dir, 0o755)`, `0o644`.
- `-q`: suprime logs de progresso em `stderr`.

#### C7 — Tratamento de erros

- `errors.Is` sobre sentinelas do core → mensagens humanizadas em `stderr`.
- `ErrMissingAPIKey` do core (se chegar até aqui): mensagem consistente com 3.5.5, exit 2.
- `ErrTranscriptUnavailable` com `--allow-partial`: Markdown parcial + aviso em `stderr` + saída 0.
- `ErrTranscriptUnavailable` sem `--allow-partial`: exit 2.

#### C8 — Testes unitários

Interface interna para injeção de fake:

```go
type extractor interface {
    Extract(ctx context.Context, id string, opts core.Options) (*core.VideoData, error)
}
```

Cobertura mínima:

| ID | Caso |
| :--- | :--- |
| T2.1 | `ParseVideoID` (ID puro, watch, youtu.be, subdomínios, parâmetros) |
| T2.2 | `ParseVideoID` rejeita playlist, canal, ID curto/longo, vazio |
| T2.3 | Precedência config: flag > env > file > default |
| T2.4 | Config file YAML inválido → exit 4 com mensagem |
| T2.5 | Campo desconhecido no YAML → exit 4 |
| T2.6 | Campo `youtube_api_key` no YAML → exit 4 com mensagem específica |
| T2.7 | `--help` sem API key → exit 0, mostra uso |
| T2.8 | `--version` sem API key → exit 0, mostra versão |
| T2.9 | `ValidateEnv` — env vazia/ausente/só espaços → exit 2 com mensagem contendo "YOUTUBE_API_KEY" e URL do console |
| T2.10 | `ValidateEnv` — env presente → nil |
| T2.11 | `tyot <URL>` sem env → exit 2 (validação vence argumento) |
| T2.12 | `tyot -o file.md` sem env → exit 2 |
| T2.13 | `run` com sucesso escreve Markdown puro em `stdout` |
| T2.14 | `-o` cria diretórios e escreve atomicamente |
| T2.15 | Erro de extração → exit 2 |
| T2.16 | `-o` em caminho inválido → exit 3 |
| T2.17 | `--no-chapters` remove headings do Markdown |
| T2.18 | `--allow-partial` + `ErrTranscriptUnavailable` → exit 0 com aviso em `stderr` |
| T2.19 | `--version` mostra `version`, `commit`, `date` |
| T2.20 | Leitura de URL via `stdin` quando não é TTY |
| T2.21 | `gen-man` gera man page válida (`man --warnings`) |

#### C9 — Man page e completions

- `cmd/gen-man/main.go`: gera `docs/man/tyot.1` e `docs/completions/` a partir da cobra root.
- Commitar `docs/man/tyot.1` e `docs/completions/` no repo.

#### C10 — CHANGELOG

- Configurar `git-cliff` com `.git-cliff.toml`.
- `make changelog` prepende a entrada ao `CHANGELOG.md`.
- Release workflow regenera o `CHANGELOG.md` antes de publicar.

#### C11 — GoReleaser (`.goreleaser.yaml`)

```yaml
version: 2
project_name: tyot

builds:
  - id: tyot
    main: ./cmd/tyot
    binary: tyot
    env: [CGO_ENABLED=0]
    goos: [linux, darwin, windows]
    goarch: [amd64, arm64]
    goarm: [7]
    ldflags:
      - -s -w
      - -X main.version={{.Version}}
      - -X main.commit={{.Commit}}
      - -X main.date={{.Date}}

archives:
  - formats: [tar.gz]
    format_overrides:
      - goos: windows
        formats: [zip]
    files: [LICENSE, README.md, CHANGELOG.md, docs/man/tyot.1]

checksum:
  name_template: checksums.txt

changelog:
  use: github
  sort: asc
  filters:
    exclude: ["^docs:", "^test:", "^chore:"]

nfpm:
  - id: deb
    package_name: tyot
    vendor: maicondmenezes
    homepage: https://github.com/dev-maicondmenezes/transcript-tube-cli
    maintainer: Maicon Menezes <maicon@example.com>
    description: Extract YouTube video metadata and transcript to Markdown.
    license: MIT
    formats: [deb]
    contents:
      - { src: docs/man/tyot.1, dst: /usr/share/man/man1/tyot.1, file_info: { mode: 0644 } }
      - { src: LICENSE, dst: /usr/share/doc/tyot/copyright }
      - { src: docs/completions/tyot.bash, dst: /usr/share/bash-completion/completions/tyot }
      - { src: docs/completions/tyot.zsh, dst: /usr/share/zsh/vendor-completions/_tyot }
      - { src: docs/completions/tyot.fish, dst: /usr/share/fish/vendor_completions.d/tyot.fish }

brews:
  - name: tyot
    repository:
      owner: maicondmenezes
      name: homebrew-tap
      token: "{{ .Env.HOMEBREW_TAP_GITHUB_TOKEN }}"
    homepage: https://github.com/dev-maicondmenezes/transcript-tube-cli
    description: Extract YouTube video metadata and transcript to Markdown.
    license: MIT
    install: |
      bin.install "tyot"
      man1.install "docs/man/tyot.1"
      bash_completion.install "docs/completions/tyot.bash" => "tyot"
      zsh_completion.install "docs/completions/tyot.zsh" => "_tyot"
      fish_completion.install "docs/completions/tyot.fish"
    test: |
      system "#{bin}/tyot", "--version"

release:
  github:
    owner: maicondmenezes
    name: transcript-tube-cli
  draft: false
  prerelease: auto
```

#### C12 — GitHub Actions (criadas e gerenciadas via `gh`)

**`.github/workflows/ci.yml`:**

```yaml
name: ci
on:
  push: { branches: [main] }
  pull_request: { branches: [main] }
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.23' }
      - run: go build ./...
      - run: go vet ./...
      - run: test -z "$(gofmt -l .)"
      - run: go test -race ./...
      - run: go run ./cmd/gen-man
      - run: git diff --exit-code docs/man/ || (echo "man page desatualizada; rode 'make man'" && exit 1)
  release-dry-run:
    runs-on: ubuntu-latest
    needs: test
    steps:
      - uses: actions/checkout@v4
        with: { fetch-depth: 0 }
      - uses: actions/setup-go@v5
        with: { go-version: '1.23' }
      - uses: goreleaser/goreleaser-action@v6
        with:
          distribution: goreleaser
          version: '~> v2'
          args: release --snapshot --clean
```

**`.github/workflows/release.yml`:**

```yaml
name: release
on:
  push:
    tags: ['v*']
permissions:
  contents: write
jobs:
  goreleaser:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with: { fetch-depth: 0 }
      - uses: actions/setup-go@v5
        with: { go-version: '1.23' }
      - name: Update CHANGELOG
        run: |
          go install github.com/orhun/git-cliff/cmd/git-cliff@latest
          git-cliff --unreleased --prepend CHANGELOG.md
          git config user.name "github-actions"
          git config user.email "actions@github.com"
          git add CHANGELOG.md
          git commit -m "chore(changelog): update for ${{ github.ref_name }}" || true
          git push origin HEAD:main
      - uses: goreleaser/goreleaser-action@v6
        with:
          distribution: goreleaser
          version: '~> v2'
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          HOMEBREW_TAP_GITHUB_TOKEN: ${{ secrets.HOMEBREW_TAP_GITHUB_TOKEN }}
```

**Configuração via `gh`:**

```bash
# Secrets
gh secret set HOMEBREW_TAP_GITHUB_TOKEN \
  --repo dev-maicondmenezes/transcript-tube-cli \
  --body "$HOMEBREW_TAP_GITHUB_TOKEN"

# Verificar workflows
gh workflow list --repo dev-maicondmenezes/transcript-tube-cli
gh run list --repo dev-maicondmenezes/transcript-tube-cli --limit 10

# Disparar CI manualmente
gh workflow run ci.yml --repo dev-maicondmenezes/transcript-tube-cli --ref main

# Observar a última execução
gh run watch --repo dev-maicondmenezes/transcript-tube-cli

# Logs de um run específico em caso de falha
gh run view <run-id> --repo dev-maicondmenezes/transcript-tube-cli --log
```

#### C13 — APT: `.deb` + repositório (enhancement)

**Primário (obrigatório na Fase 2):** `.deb` em GitHub Releases.

```bash
gh release download --repo dev-maicondmenezes/transcript-tube-cli \
  --pattern 'tyot_*_amd64.deb'
sudo apt install ./tyot_*.deb
```

**Enhancement (se houver tempo):** apt repo em GitHub Pages.

Workflow adicional `.github/workflows/apt-repo.yml` usa `aptly`/`reprepro` em CI para gerar `Packages.gz` + `Release` + `InRelease` assinados e faz push dos artefatos para `gh-pages` (via `git push` no job, que já tem permissão de escrita).

#### C14 — Homebrew (via `gh`)

- Criar o tap se ainda não existir:
  ```bash
  gh repo create dev-maicondmenezes/homebrew-tap --public \
    --description "Homebrew tap for maicondmenezes tools" --confirm
  ```
- GoReleaser gera `tyot.rb` em cada release.
- Usuário instala com:
  ```bash
  brew tap dev-maicondmenezes/tap
  brew install tyot
  man tyot
  ```

#### C15 — GitHub Releases (via `gh`)

- GoReleaser publica automaticamente (chamado pelo workflow).
- Verificar release:
  ```bash
  gh release view v0.1.0 --repo dev-maicondmenezes/transcript-tube-cli
  gh release download v0.1.0 --repo dev-maicondmenezes/transcript-tube-cli
  ```
- Histórico completo em `https://github.com/dev-maicondmenezes/transcript-tube-cli/releases`.

### 3.7 Testes

- `go test ./...` offline (fakes).
- `go test -race ./...`.
- Teste de man page: `man --warnings -l docs/man/tyot.1`.
- Teste de `.deb`:
  ```bash
  goreleaser release --snapshot --clean
  dpkg-deb --info dist/tyot_*_amd64.deb
  sudo dpkg -i dist/tyot_*_amd64.deb
  tyot --version
  man tyot
  sudo dpkg -r tyot
  ```
- Teste E2E manual (com API key real): `--output`, stdout, `--no-chapters`, `--allow-partial`, `--lang`.
- Teste de validação de env (ver 3.5.5): todas as combinações da tabela T2.9–T2.12.

### 3.8 Documentação da fase

| Arquivo | Conteúdo |
| :--- | :--- |
| `README.md` (repo CLI) | Instalação (apt/brew/gh), uso básico, link para `docs/cli.md`, requisito de `YOUTUBE_API_KEY`, comandos de build/desenvolvimento, contribuição. |
| `docs/cli.md` | Manual completo: todas as flags, formatos de URL, config file (YAML), precedência, **seção dedicada sobre `YOUTUBE_API_KEY` (obtenção, validação, mensagens de erro)**, exit codes, exemplos, troubleshooting. |
| `docs/man/tyot.1` | Man page gerada; versionada. |
| `CHANGELOG.md` | Gerado por `git-cliff`. |
| `AGENTS.md` | Regras do repo CLI. |

### 3.9 Critérios de aceite da Fase 2

- [ ] Repositório `transcript-tube-cli` público com `main` protegida (verificado com `gh api .../branches/main/protection`).
- [ ] `go.mod` referencia `github.com/dev-maicondmenezes/transcript-tube-core v0.2.0` sem `replace`.
- [ ] `go build ./...`, `go vet ./...`, `gofmt -l .` limpos.
- [ ] `go test ./...` e `go test -race ./...` passam offline.
- [ ] CI verde (`gh run list` confirma `ci.yml` no último push).
- [ ] `make man` regenera a man page; `git diff --exit-code docs/man/tyot.1` passa no CI.
- [ ] Precedência de config (flag > env > file > default) coberta por teste.
- [ ] Config file com `youtube_api_key` é rejeitado (exit 4) com mensagem específica.
- [ ] **Todas as invocações não-help validam `YOUTUBE_API_KEY`** (T2.9–T2.12 passam).
- [ ] `tyot` funciona com ID puro, `watch?v=`, `youtu.be`.
- [ ] `-o` grava arquivo e cria diretórios.
- [ ] Sem `-o`, Markdown puro em `stdout` (testado em pipe).
- [ ] Exit codes 0, 1, 2, 3, 4 reproduzíveis.
- [ ] `--help` e `--version` funcionam sem API key.
- [ ] `man tyot` funciona após instalação do `.deb`.
- [ ] Release `v0.1.0` publicada (verificada com `gh release view v0.1.0`) com binários `tyot` (linux/darwin/windows; amd64/arm64), `.deb` (amd64/arm64), `checksums.txt`, CHANGELOG, formula em `homebrew-tap`.
- [ ] `apt install ./tyot_*.deb` funciona em Debian/Ubuntu limpos.
- [ ] `brew tap dev-maicondmenezes/tap && brew install tyot` funciona em macOS.
- [ ] `gh release download` funciona para a última e para uma versão anterior.
- [ ] `CHANGELOG.md` inclui `v0.1.0` com `Added`/`Changed`/`Fixed`.

### 3.10 Comandos de verificação (fase isolada)

```bash
source .env

# build e testes
make check
go test -race ./...
make man

# release dry-run
make release-dry-run

# validação local do .deb
dpkg-deb --info dist/tyot_*_amd64.deb
sudo dpkg -i dist/tyot_*_amd64.deb
tyot --version
man tyot
sudo dpkg -r tyot

# teste funcional real (env presente)
tyot dQw4w9WgXcQ > /tmp/cli.md
tyot "https://youtu.be/dQw4w9WgXcQ" -o /tmp/out/rick.md

# validação de env (todas devem falhar com exit 2 e mensagem clara)
env -u YOUTUBE_API_KEY tyot dQw4w9WgXcQ; echo "exit=$?"
env -u YOUTUBE_API_KEY tyot -o /tmp/x.md; echo "exit=$?"
YOUTUBE_API_KEY="" tyot dQw4w9WgXcQ; echo "exit=$?"
YOUTUBE_API_KEY="   " tyot dQw4w9WgXcQ; echo "exit=$?"

# help/version devem funcionar SEM a env
env -u YOUTUBE_API_KEY tyot --help; echo "exit=$?"     # 0
env -u YOUTUBE_API_KEY tyot --version; echo "exit=$?"  # 0

# argumento inválido sem env → validação de env vence
env -u YOUTUBE_API_KEY tyot id-invalido; echo "exit=$?" # 2

# release (a partir de uma tag)
git tag v0.1.0
git push origin main --tags
gh run watch --repo dev-maicondmenezes/transcript-tube-cli
gh release view v0.1.0 --repo dev-maicondmenezes/transcript-tube-cli
```

### 3.11 Riscos e mitigação

| Risco | Mitigação |
| :--- | :--- |
| `replace` local esquecido no `go.mod` publicado | CI falha se `go mod edit -json` contiver `Replace` com path local. |
| Man page desatualizada após mudar flags | CI regenera e compara com `git diff --exit-code`. |
| Formula do Homebrew quebrada | `goreleaser release --snapshot` no CI a cada PR; `brew audit` em workflow semanal. |
| `.deb` com dependências faltando | `lintian` no CI sobre o `.deb` do snapshot. |
| `.deb` de arch errada para o usuário | Releases com `amd64`, `arm64`, `armv7` explícitos nos nomes. |
| Usuário esquece de definir `YOUTUBE_API_KEY` | Validação robusta em `PersistentPreRunE` com mensagem rica (ver 3.5.5). |
| Usuário tenta colocar API key no config file | Loader rejeita explicitamente com `exit 4` e mensagem instruindo a usar env. |
| Tag Git empurrada sem release | `release.yml` escuta `push: tags: ['v*']`; `gh run watch` no checklist de release. |
| Vazamento de API key em logs de CI | Nunca logar env; `gh secret set` para tokens; `core` não loga env. |

### 3.12 Decisões e justificativas

| Decisão | Alternativas | Justificativa |
| :--- | :--- | :--- |
| **Multi-repo** (core, cli, extension) | Monorepo com submódulos | Distribuição, CI/CD e versionamento independentes. |
| **`gh` para provisionamento** | Cliques na UI | Reprodutibilidade, versionamento das operações, auditoria. |
| **Cobra** para o CLI | `flag` stdlib, `urfave/cli` | Man page e completions nativas. |
| **YAML** para config | TOML, JSON | Legibilidade; `gopkg.in/yaml.v3` maduro. |
| **Config + env + flags** com precedência | Só env + flags | Setup persistente + override por sessão + override pontual. |
| **`YOUTUBE_API_KEY` só via env** | `--api-key` flag, campo no config | Evita vazamento em `ps`, histórico de shell e config files versionados; simplifica modelo de segurança. |
| **Validação obrigatória em `PersistentPreRunE`** | Validação tardia no `run` | Fail-fast, mensagem única e consistente, cobertura de todas as rotas de execução real. |
| **GoReleaser v2** | Makefile manual | Padrão da indústria; `.deb`, brew, checksums, changelog, anexos no GitHub. |
| **nfpm** para `.deb` | `dpkg-deb` manual | Integrado ao GoReleaser; inclui man page, completions, copyright. |
| **`.deb` em GitHub Releases** como distribuição primária | apt repo dedicado | Custo zero; atende "instalável via apt". |
| **`git-cliff`** para CHANGELOG | `goreleaser changelog` nativo | Formato Keep a Changelog; versionado no repo. |
| **SemVer estrito** | CalVer | Previsibilidade. |
| **`v0.2.0`** como primeira tag do core | `v0.1.0` retroativo | `v0.1.0-core` não é versão válida de módulo Go. |
| **Binário `tyot`** | `transcript_tube` | Mnemônico curto e memorável. |
| **Pacote principal em `cmd/tyot/`** | Main no root | Permite `go install .../cmd/tyot@latest` → binário `tyot`, mantendo repo descritivo. |

---

## 4. Fase 3 — Chrome Extension (atualizado para multi-repo)

### 4.1 Mudanças em relação ao plano original

A Fase 3 passa a viver no repositório **`transcript-tube-extension`** com módulo `github.com/dev-maicondmenezes/transcript-tube-extension`. O conteúdo das seções 4.2–4.9 do plano original permanece válido, com os seguintes ajustes:

1. **Criação do repo via `gh`:**
   ```bash
   gh repo create dev-maicondmenezes/transcript-tube-extension \
     --public \
     --description "Chrome extension + native messaging host for transcript_tube." \
     --disable-wiki --confirm
   ```
2. **`go.mod` próprio** com `require github.com/dev-maicondmenezes/transcript-tube-core vX.Y.Z` (sem `replace` na `main`).
3. **Distribuição da extensão:** `.zip`/`.crx` anexado às GitHub Releases, com instruções de instalação em `chrome://extensions`. Publicação na Chrome Web Store fica como enhancement.
4. **Distribuição do `nm-host`:** binário Go por OS/arch anexado às GitHub Releases + `scripts/install-native-host.sh`. GoReleaser no repo da extensão.
5. **CI/CD próprio** (`.github/workflows/ci.yml` e `release.yml`), com jobs Go (host) e JS (extensão).
6. **CHANGELOG.md** próprio.
7. **Dependabot** para o core (`.github/dependabot.yml`).

O restante da Fase 3 (tarefas C1–C6 de `nm-host`, C1–C6 da extensão, testes, critérios, riscos) mantém-se conforme o plano original, com caminhos e URLs atualizados.

### 4.2 Critérios de aceite da Fase 3 (delta)

- [ ] Repositório `transcript-tube-extension` público com `main` protegida (via `gh api`).
- [ ] `go.mod` referencia `transcript-tube-core vX.Y.Z` sem `replace`.
- [ ] Release `v0.1.0` publicada (`gh release view v0.1.0`) com `nm-host` (linux/darwin/windows), extensão `.zip`, `checksums.txt`, CHANGELOG.
- [ ] Dependabot configurado para o core.
- [ ] E2E CLI ↔ extensão (mesmo Markdown para o mesmo vídeo).

---

## 5. Itens transversais

### 5.1 Documentação consolidada por repositório

| Documento | Repositório | Conteúdo |
| :--- | :--- | :--- |
| `README.md` | `transcript-tube-core` | Visão geral, quickstart Go, API pública. |
| `docs/architecture.md` | `transcript-tube-core` | Diagramas, camadas, dependências. |
| `docs/core.md` | `transcript-tube-core` | API, erros, opções. |
| `docs/testing.md` | `transcript-tube-core` | Convenções, fakes, golden, tags. |
| `spec.md`, `plan.md` | `transcript-tube-core` | Especificação e plano. |
| `README.md` | `transcript-tube-cli` | Instalação (apt/brew/gh), uso, link para `docs/cli.md`. |
| `docs/cli.md` | `transcript-tube-cli` | Manual completo (flags, config, exit codes, **seção de `YOUTUBE_API_KEY`**). |
| `docs/man/tyot.1` | `transcript-tube-cli` | Man page (gerada, versionada). |
| `CHANGELOG.md` | `transcript-tube-cli` | Gerado por `git-cliff`. |
| `README.md` | `transcript-tube-extension` | Instalação da extensão e do host. |
| `docs/extension.md` | `transcript-tube-extension` | Guia completo, protocolo NM, troubleshooting. |
| `CHANGELOG.md` | `transcript-tube-extension` | Gerado por `git-cliff`. |

### 5.2 Estratégia de testes

| Camada | Ferramenta | Executado no CI do repo |
| :--- | :--- | :--- |
| Unitário Go core | `go test` | `transcript-tube-core` |
| Corrida core | `go test -race` | `transcript-tube-core` |
| Unitário Go CLI | `go test` | `transcript-tube-cli` |
| Snapshot de release | `goreleaser release --snapshot` + `dpkg-deb --info` | `transcript-tube-cli` |
| Unitário Go nm-host | `go test` | `transcript-tube-extension` |
| Funções puras JS | `node --test` | `transcript-tube-extension` |
| Sintaxe JS | `node --check` | `transcript-tube-extension` |
| Integração real (YouTube) | `go test -tags=integration` | manual/agendado |

### 5.3 CI/CD por repositório (todas gerenciadas via `gh`)

| Repositório | Workflow | Trigger |
| :--- | :--- | :--- |
| `transcript-tube-core` | `ci.yml` | push/PR em `main` |
| `transcript-tube-core` | `release.yml` | tag `v*` → tag módulo Go + GitHub Release |
| `transcript-tube-cli` | `ci.yml` | push/PR em `main` |
| `transcript-tube-cli` | `release.yml` | tag `v*` → goreleaser + `.deb` + brew + GitHub Release |
| `transcript-tube-cli` | `apt-repo.yml` (enhancement) | tag `v*` → push em `gh-pages` |
| `transcript-tube-extension` | `ci.yml` | push/PR em `main` |
| `transcript-tube-extension` | `release.yml` | tag `v*` → goreleaser (nm-host) + zip da extensão + GitHub Release |

**Comandos `gh` de operação** (aplicados a qualquer repo):

```bash
# Forçar execução de um workflow
gh workflow run ci.yml --repo dev-maicondmenezes/<repo> --ref main

# Observar runs recentes
gh run list --repo dev-maicondmenezes/<repo> --limit 20
gh run watch --repo dev-maicondmenezes/<repo>

# Ver log de um run com falha
gh run view <run-id> --repo dev-maicondmenezes/<repo> --log

# Configurar/atualizar secrets
gh secret set <NOME> --repo dev-maicondmenezes/<repo> --body "$VALOR"
gh secret list --repo dev-maicondmenezes/<repo>

# Configurar variáveis (valores não sensíveis)
gh variable set <NOME> --repo dev-maicondmenezes/<repo> --body "valor"
```

### 5.4 Ordem de execução e marcos

| Marco | Fase | Gate |
| :--- | :--- | :--- |
| **M1 — concluído** | Core | `make check` + teste real via `examples/core-demo` + contrato congelado |
| **M1.5** | Migração multi-repo | `transcript-tube-core` publicado em `v0.2.0` (via `gh repo create`); monorepo arquivado |
| **M2** | CLI | `make check` + `make release-dry-run` + `.deb` validado + release `v0.1.0` publicada + `brew install tyot` funcional + `man tyot` + validação de env completa |
| **M3** | Extension | `make check-extension` + extensão carregada + E2E com CLI |
| **M4** | Integração final | CLI e extensão geram o mesmo Markdown; documentação revisada em todos os repos |

### 5.5 Checklist de encerramento do projeto

- [ ] Três repositórios no GitHub com CI verde (`gh run list` em cada um) e releases `vX.Y.Z`.
- [ ] Branches `main` protegidas em todos os repos (verificado com `gh api repos/.../branches/main/protection`).
- [ ] `transcript-tube-core` com `docs/` completo e `spec.md`/`plan.md` atualizados.
- [ ] `transcript-tube-cli` instalável via apt, brew e GitHub Releases; `man tyot` funcional; CHANGELOG versionado; validação de `YOUTUBE_API_KEY` cobrindo 100% das invocações não-help.
- [ ] `transcript-tube-extension` com `nm-host` binário por OS e extensão `.zip`; E2E com CLI.
- [ ] Nenhum segredo versionado em nenhum repo (`git grep -i "api_key\|YOUTUBE_API_KEY"`).
- [ ] `.env.example` presente em todos os repos.
- [ ] Secrets configurados via `gh secret set` (`HOMEBREW_TAP_GITHUB_TOKEN` no `transcript-tube-cli`).
- [ ] Limitações conhecidas documentadas (limite de 1 MiB, API key obrigatória via env, idiomas, apt repo enhancement).
- [ ] `AGENTS.md` em cada repo refletindo as regras da fase (incluindo "sem `--api-key`" e "validação obrigatória de `YOUTUBE_API_KEY`").