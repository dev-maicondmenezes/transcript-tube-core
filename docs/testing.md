# Testes

## Convenções

- Testes padrão **nunca** acessam a rede.
- Table-driven sempre que possível.
- Fakes implementam as interfaces de injeção (`MetadataProvider`,
  `TranscriptProvider`, `ChapterProvider`).
- Funções puras (`ToMarkdown`, `parseISODuration`, `formatTimestamp`,
  `parseChapters`, `parseNextResponse`, `parseJSON3`, `parseSRV3`) são testadas
  diretamente.
- Golden files ficam em `core/testdata/` e são revisados manualmente ao mudar o
  template.
- Nenhum teste imprime ou registra API keys.

## Seams de testabilidade

| Seam | Uso |
| :--- | :--- |
| `core.NewExtractor(m, t)` | Substitui metadados e transcrição por fakes. |
| `core.NewExtractorWithChapters(m, t, c)` | Adiciona um provedor de capítulos fake. |
| `metadata.New(key, WithBaseURL, WithHTTPClient)` | Aponta para `httptest.Server`. |
| `transcript.New(WithInnerTubeFetcher, WithTimedTextFetcher)` | Simula a cascata. |
| `chapters.New(WithBaseURL, WithHTTPClient, WithClientVersion)` | Aponta para `httptest.Server`. |
| `core/testdata/*.md` | Golden tests. |
| `examples/core-demo` | Teste manual ponta a ponta. |

## Comandos

```bash
go test ./...
go test ./core/... -v
go test -race ./core/...
go test -cover ./core/...
```

Cobertura atual (Fase 1):

| Pacote | Cobertura |
| :--- | :--- |
| `core` | ~84% |
| `core/chapters` | ~85% |
| `core/internal/durfmt` | ~91% |
| `core/metadata` | ~88% |
| `core/transcript` | ~69% |

A cobertura é informativa nesta etapa; a meta posterior é ≥ 70% em `core` e
≥ 60% nos adaptadores.

## Integração

Testes reais ficam atrás da build tag `integration`:

```bash
YOUTUBE_API_KEY="$YOUTUBE_API_KEY" \
TEST_VIDEO_ID="$TEST_VIDEO_ID" \
go test -tags=integration ./core/... -run Integration -v
```

`TEST_VIDEO_ID` é opcional; o padrão é `dQw4w9WgXcQ`.

## Mapa dos testes mínimos

| ID | Teste | Arquivo |
| :--- | :--- | :--- |
| T1.1 | `ValidateVideoID` | `core/validate_test.go` |
| T1.2 | `ToMarkdown` golden | `core/markdown_test.go` |
| T1.3 | `formatTimestamp` | `core/duration_test.go` |
| T1.4 | `parseISODuration` | `core/duration_test.go` |
| T1.5 | Extração paralela com fakes | `core/extractor_test.go` |
| T1.6 | Falha de metadados | `core/extractor_test.go` |
| T1.7 | Falha de transcrição e modo parcial | `core/extractor_test.go` |
| T1.8 | `videos.list` com httptest | `core/metadata/client_test.go` |
| T1.9 | Cascata InnerTube → timedtext | `core/transcript/client_test.go` |
| T1.10 | Detecção de capítulos na descrição | `core/chapters_test.go` |
| T1.11 | Capítulos `###` no Markdown | `core/markdown_test.go` |
| T1.12 | Cliente InnerTube `next` com httptest | `core/chapters/client_test.go` |
| T1.13 | Prioridade e fallback do provedor de capítulos | `core/extractor_test.go` |

## Backlog para a etapa posterior

- Contexto cancelado durante metadados/transcrição.
- Campos ausentes (`likeCount` oculto, `duration` ausente).
- Markdown com Unicode, emojis e quebras de linha.
- Offsets fora de ordem ou negativos.
- Erros HTTP 403/429/500 do `timedtext`.
- Timeout por chamada.
- Concorrência com `-race`.
- Golden adicional para vídeo com mais de uma hora (já existe um caso básico).

## Critérios de aceite da Fase 1 — registro

Executado em 2026-11-07 com Go 1.27.3 (módulo declara Go 1.23.6):

- [x] `go build ./...` compila.
- [x] `go vet ./...` e `gofmt -l .` sem saída.
- [x] `go test ./...` passa offline (nenhum teste acessa a rede).
- [x] `go test -race ./core/...` passa.
- [x] `ToMarkdown` é byte a byte igual aos golden files.
- [x] `errors.Is` funciona para os cinco erros sentinela.
- [x] Metadados e transcrição rodam em paralelo (teste com barreira + `-race`).
- [x] `AllowPartialTranscript` retorna Markdown parcial quando só a transcrição falha.
- [x] Nenhum import direto de `cmd/`, Chrome, `os/exec` ou terminal em `core`.
      (`os/exec` aparece apenas na árvore transitiva de `google.golang.org/api`.)
- [ ] `examples/core-demo` com vídeo real: pendente de `YOUTUBE_API_KEY` no
      ambiente de execução.
- [x] Documentação da fase publicada.

