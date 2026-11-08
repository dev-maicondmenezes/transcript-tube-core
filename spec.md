# Especificação da Aplicação: `transcript_tube`

**Versão:** 1.0  
**Status:** Rascunho para implementação  
**Ambiente de desenvolvimento:** OpenCode + DeepSeek  
**Linguagem:** Go (monorepo)

---

## 1. Visão Geral

`transcript_tube` é uma ferramenta que extrai metadados e a transcrição completa de um vídeo do YouTube e os formata como um documento Markdown estruturado (header + corpo). O projeto adota uma arquitetura **Core + Adaptadores**: toda a lógica de extração e formatação reside em um pacote Go compartilhado, e cada interface (CLI, extensão Chrome, MCP) é um consumidor fino desse núcleo.

Esta especificação cobre **duas interfaces**:

1.  **CLI** (`transcript_tube`): binário de linha de comando para uso direto no terminal.
2.  **Extensão Chrome**: interface gráfica que se integra à página do YouTube e delega a extração a um host de Native Messaging escrito em Go.

---

## 2. Arquitetura

```
transcript_tube/
├── go.mod
├── core/                       # Lógica compartilhada (sem I/O de interface)
│   ├── extractor.go            # Orquestra metadados + transcrição
│   ├── metadata/               # Cliente YouTube Data API v3
│   ├── transcript/             # Cliente de transcrição (InnerTube / timedtext)
│   └── markdown.go             # Formata VideoData em Markdown
├── cmd/
│   ├── cli/                    # Binário do CLI
│   └── nm-host/                # Host de Native Messaging (Go)
└── extension/                  # Código da extensão Chrome (JS/HTML/CSS)
    ├── manifest.json           # Manifest V3
    ├── service-worker.js       # Orquestra chamadas ao host nativo
    └── popup/                  # UI do popup (opcional)
```

O pacote `core` **não importa** nenhuma dependência do Chrome ou de terminal. Ele expõe funções puras que recebem um contexto e um identificador de vídeo, e retornam uma estrutura `VideoData` ou um erro.

---

## 3. Especificação do Core (`core`)

### 3.1 Tipo `VideoData`

```go
type VideoData struct {
    ID          string
    Title       string
    Channel     string
    Description string
    PublishedAt time.Time
    Duration    string        // ex.: "12:34"
    ViewCount   uint64
    LikeCount   uint64
    Transcript  []TranscriptSegment
}

type TranscriptSegment struct {
    OffsetMs int64   // milissegundos desde o início
    Text     string
}
```

### 3.2 Função de Extração

```go
func Extract(ctx context.Context, videoID string) (*VideoData, error)
```

**Comportamento:**

1.  Valida o `videoID` (11 caracteres alfanuméricos).
2.  Em paralelo, chama o cliente de metadados e o cliente de transcrição.
3.  Se qualquer uma das chamadas falhar, retorna erro com contexto (qual etapa falhou).
4.  Retorna `VideoData` preenchido.

### 3.3 Metadados (`core/metadata`)

Usa a **YouTube Data API v3** oficial, método `videos.list` com `part=snippet,statistics`.

- **Autenticação:** API key (variável de ambiente `YOUTUBE_API_KEY`).
- **Custo:** 1 unidade por chamada; cota padrão de 10.000 unidades/dia é suficiente para uso individual.
- **Biblioteca:** `google.golang.org/api/youtube/v3`.

### 3.4 Transcrição (`core/transcript`)

**Estratégia em cascata:**

| Prioridade | Método | Biblioteca / Endpoint |
| :--- | :--- | :--- |
| 1 | InnerTube API | `github.com/horiagug/youtube-transcript-api-go` ou equivalente |
| 2 | `timedtext` público | `https://www.youtube.com/api/timedtext` (sem chave) |
| 3 | Erro explícito | Se ambas falharem, `ErrTranscriptUnavailable` |

A biblioteca `github.com/horiagug/youtube-transcript-api-go` suporta múltiplos idiomas e retorna segmentos com timestamps, o que atende ao requisito.

### 3.5 Formatação Markdown (`core/markdown.go`)

```go
func ToMarkdown(data *VideoData) string
```

**Template de saída:**

```markdown
# {Title}

**Canal:** {Channel}
**URL:** https://www.youtube.com/watch?v={ID}
**Publicado em:** {PublishedAt formatado}
**Duração:** {Duration}
**Visualizações:** {ViewCount}
**Likes:** {LikeCount}

---

## Descrição

{Description}

---

## Transcrição

[{mm:ss}] {Texto do primeiro segmento}
[{mm:ss}] {Texto do segundo segmento}
...
```

O tempo de cada segmento é formatado como `mm:ss` (ou `hh:mm:ss` se a duração exceder 1 hora).

---

## 4. Interface CLI (`cmd/cli`)

### 4.1 Nome do Binário

`transcript_tube`

### 4.2 Uso

```
transcript_tube <URL-ou-ID> [--output <arquivo.md>] [--lang <código>]
```

| Flag | Descrição | Padrão |
| :--- | :--- | :--- |
| `--output`, `-o` | Caminho do arquivo Markdown de saída. Se omitido, imprime no `stdout`. | `stdout` |
| `--lang` | Código de idioma preferido para a transcrição (ex.: `pt`, `en`). | `pt` (se disponível), senão o primeiro disponível |

### 4.3 Comportamento

1.  Extrai o `videoID` da URL (formatos aceitos: `youtube.com/watch?v=`, `youtu.be/`, ou o próprio ID).
2.  Chama `core.Extract(ctx, videoID)`.
3.  Chama `core.ToMarkdown(data)`.
4.  Se `--output` foi fornecido, escreve o Markdown no arquivo (cria diretórios se necessário). Caso contrário, imprime no `stdout`.

### 4.4 Códigos de Saída

| Código | Significado |
| :--- | :--- |
| 0 | Sucesso |
| 1 | Erro de argumento (URL/ID inválido) |
| 2 | Erro de extração (API, rede, transcrição indisponível) |
| 3 | Erro de escrita de arquivo |

### 4.5 Dependência de Ambiente

Requer `YOUTUBE_API_KEY` configurada. Se ausente, o CLI exibe uma mensagem de erro clara e encerra com código 2.

---

## 5. Interface Extensão Chrome

### 5.1 Visão Geral da Arquitetura

A extensão **não executa Go diretamente**. Ela usa o protocolo **Native Messaging** do Chrome para se comunicar com um pequeno binário Go instalado na máquina do usuário (`nm-host`). Esse host é o único componente que importa o pacote `core`.

```
[Chrome Extension (JS)]
        │  chrome.runtime.connectNative()
        ▼
[Native Messaging Host (Go)] ──► [core] ──► YouTube APIs
```

### 5.2 Host de Native Messaging (`cmd/nm-host`)

**Responsabilidades:**

- Ler mensagens do `stdin` no formato de Native Messaging (comprimento de 4 bytes em little-endian + JSON).
- O JSON recebido contém: `{ "videoId": "..." }`.
- Chamar `core.Extract` e `core.ToMarkdown`.
- Responder no `stdout` com: `{ "markdown": "..." }` ou `{ "error": "..." }`.

**Manifesto do host** (`com.transcript_tube.host.json`):

```json
{
  "name": "com.transcript_tube.host",
  "description": "transcript_tube Native Messaging Host",
  "path": "/caminho/absoluto/para/nm-host",
  "type": "stdio",
  "allowed_origins": [
    "chrome-extension://<ID_DA_EXTENSAO>/"
  ]
}
```

O arquivo de manifesto deve ser colocado no diretório de configuração do Chrome específico do SO (ex.: `~/.config/google-chrome/NativeMessagingHosts/` no Linux). A referência `jfarleyx/chrome-native-messaging-golang` documenta o padrão de comunicação.

### 5.3 Extensão (Manifest V3)

#### `manifest.json`

```json
{
  "manifest_version": 3,
  "name": "transcript_tube",
  "version": "1.0",
  "description": "Extrai metadados e transcrição de vídeos do YouTube para Markdown.",
  "permissions": ["activeTab", "nativeMessaging", "scripting"],
  "host_permissions": ["https://www.youtube.com/*"],
  "background": {
    "service_worker": "service-worker.js"
  },
  "action": {
    "default_popup": "popup/popup.html"
  },
  "content_scripts": [
    {
      "matches": ["https://www.youtube.com/watch*"],
      "js": ["content.js"]
    }
  ]
}
```

#### `content.js` (content script)

- Detecta a URL do vídeo na página (`window.location.href`).
- Extrai o `videoId` via regex.
- Expõe o `videoId` para o popup ou service worker mediante mensagem.

#### `service-worker.js`

- Recebe uma mensagem do popup com o `videoId`.
- Abre uma porta de Native Messaging: `chrome.runtime.connectNative("com.transcript_tube.host")`.
- Envia `{ "videoId": "..." }`.
- Aguarda a resposta.
- Retorna o Markdown (ou erro) ao popup.

#### `popup/popup.html` + `popup.js`

Interface mínima com:

1.  Botão **"Extrair para Markdown"**.
2.  Área de texto (readonly) exibindo o Markdown gerado (ou uma mensagem de erro).
3.  Botão **"Copiar"** (copia o conteúdo da área de texto para a área de transferência).
4.  Botão **"Baixar .md"** (cria um `Blob` e dispara o download).

### 5.4 Fluxo do Usuário

1.  Usuário abre um vídeo no YouTube.
2.  Clica no ícone da extensão na barra de ferramentas.
3.  O popup exibe o título do vídeo detectado e o botão de extração.
4.  Usuário clica em extrair.
5.  O popup exibe o Markdown gerado e oferece as opções de copiar ou baixar.

### 5.5 Tratamento de Erros

| Cenário | Comportamento |
| :--- | :--- |
| Host nativo não instalado | Popup exibe mensagem: "Host nativo não encontrado. Instale o `nm-host` e registre o manifesto." |
| `YOUTUBE_API_KEY` ausente no ambiente do host | Host retorna erro descritivo; popup exibe a mensagem. |
| Transcrição indisponível | Host retorna erro `ErrTranscriptUnavailable`; popup exibe aviso, mas ainda permite copiar o Markdown parcial (sem transcrição), se desejado. |
| Vídeo não é uma página de watch | Popup exibe: "Abra um vídeo do YouTube para extrair." |

---

## 6. Configuração do Ambiente de Desenvolvimento com OpenCode + DeepSeek

### 6.1 Pré-requisitos

- Go 1.23 ou superior.
- OpenCode v2 (a versão v1.18.x não é mais suportada pela configuração recomendada).
- Chave de API da DeepSeek (`platform.deepseek.com/api_keys`).

### 6.2 Configuração do OpenCode

O OpenCode usa `opencode.jsonc` para configuração de modelos e `AGENTS.md` para regras globais do projeto.

**`opencode.jsonc` (trecho relevante):**

```jsonc
{
  "model": "deepseek/deepseek-flash",
  "providers": {
    "deepseek": {
      "apiKey": "env:DEEPSEEK_API_KEY"
    }
  }
}
```

**`AGENTS.md`** deve declarar, no mínimo:

- Stack: Go, monorepo com `core` + `cmd` + `extension`.
- Princípios: o pacote `core` não importa dependências de Chrome ou terminal.
- Anti-padrões: não fazer scraping do HTML do YouTube; usar as APIs especificadas.
- Comandos de verificação: `go build ./...`, `go vet ./...`, `go test ./...`.

A configuração de referência `znlgis/my-opencode-deepseek-config` demonstra o uso de `AGENTS.md` para disciplina de contexto e roteamento de tarefas entre agentes.

### 6.3 Variáveis de Ambiente Necessárias

```bash
export YOUTUBE_API_KEY="sua-chave"
export DEEPSEEK_API_KEY="sua-chave-deepseek"
```

---

## 7. Plano de Implementação Sugerido

| Fase | Escopo | Entregável |
| :--- | :--- | :--- |
| **1** | Core: tipos, cliente de metadados, cliente de transcrição, formatação Markdown | `core/` compilando com testes unitários |
| **2** | CLI: parsing de argumentos, integração com o core, escrita em arquivo/stdout | Binário `transcript_tube` funcional |
| **3** | Host de Native Messaging: protocolo stdin/stdout, registro do manifesto | `nm-host` testável via eco de mensagens |
| **4** | Extensão Chrome: manifest, service worker, popup, content script | Extensão carregável em modo desenvolvedor |
| **5** | Integração e testes ponta a ponta | CLI + extensão operacionais com a mesma chave de API |

---

## 8. Decisões Técnicas Consolidadas

| Componente | Decisão | Justificativa |
| :--- | :--- | :--- |
| **Metadados** | YouTube Data API v3 (`videos.list`) | Estável, oficial, cota generosa (1 unidade/chamada) |
| **Transcrição** | InnerTube com fallback para `timedtext` | Sem chave de API; biblioteca Go dedicada disponível |
| **Extensão ↔ Go** | Native Messaging | Mantém o core em Go; protocolo suportado nativamente pelo Chrome |
| **Manifest** | V3 | Requisito atual do Chrome Web Store |
| **Arquitetura** | Core + adaptadores | Evita duplicação entre CLI e extensão |

---

Esta especificação está pronta para ser usada como entrada no OpenCode. O próximo passo prático é criar o `go.mod`, a estrutura de diretórios e implementar o pacote `core` com testes unitários, antes de avançar para o CLI e a extensão.