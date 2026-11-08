# Arquitetura

## Visão de camadas

```text
                 ┌────────────────────────────┐
                 │  Consumidores finos        │
                 │  cmd/cli   cmd/nm-host     │
                 │  examples/core-demo        │
                 └─────────────┬──────────────┘
                               │
                 ┌─────────────▼──────────────┐
                 │           core             │
                 │  Extract / ToMarkdown      │
                 │  ValidateVideoID           │
                 └──┬──────────┬──────────┬───┘
                    │          │          │
        ┌───────────▼──┐ ┌─────▼───────┐ ┌▼───────────────┐
        │core/metadata │ │core/transcr.│ │ core/chapters  │
        │Data API v3   │ │InnerTube -> │ │ InnerTube next │
        │              │ │timedtext    │ │                │
        └───────┬──────┘ └──────┬──────┘ └───────┬────────┘
                │               │                │
                ▼               ▼                ▼
        YouTube Data API   YouTube endpoints  YouTube InnerTube
```

A Fase 3 adiciona:

```text
Chrome extension (JS)
        │ chrome.runtime.connectNative()
        ▼
cmd/nm-host (Go, stdio framing)
        │
        ▼
      core
```

## Dependências permitidas

| Pacote | Pode importar | Não pode importar |
| :--- | :--- | :--- |
| `core` | `core/metadata`, `core/transcript`, `core/chapters`, `core/internal/*`, stdlib | `cmd/`, Chrome, `os/exec`, terminal, JS |
| `core/metadata` | stdlib, `google.golang.org/api/youtube/v3`, `core/internal/*` | `core`, `core/transcript`, `cmd/` |
| `core/transcript` | stdlib, biblioteca InnerTube, `core/internal/*` | `core`, `core/metadata`, `cmd/` |
| `core/chapters` | stdlib | `core`, `core/metadata`, `core/transcript`, `cmd/` |
| `core/internal/durfmt` | stdlib | `core`, adaptadores |
| `cmd/*` | `core`, stdlib | lógica de extração duplicada |

`core/metadata`, `core/transcript` e `core/chapters` não importam o pacote raiz
`core` para evitar ciclos. A orquestração converte os tipos dos adaptadores em
tipos do `core`.

## Fluxo de extração

1. `ExtractWithOptions` valida o `videoID` com `^[A-Za-z0-9_-]{11}$`.
2. Verifica `YOUTUBE_API_KEY`; se ausente retorna `ErrMissingAPIKey`.
3. Constrói `metadata.Client`, `transcript.Client` e `chapters.Client`.
4. `Extractor.Extract` dispara metadados, transcrição e capítulos **em
   paralelo** com `sync.WaitGroup`, sem `errgroup.WithContext`: a falha de um
   provedor não pode cancelar os demais. Só o `ctx` do chamador cancela o
   trabalho.
5. Metadados são obrigatórios. A transcrição é obrigatória, exceto com
   `AllowPartialTranscript`. Capítulos são opcionais.
6. Os capítulos vêm do InnerTube `next` (lista oficial do player, incluindo
   capítulos automáticos). Se o provedor falhar ou retornar vazio, o core usa
   os timestamps da descrição como fallback.
7. `ToMarkdown` renderiza o template canônico, inserindo os capítulos como
   subtítulos `###` dentro da seção de transcrição.

## Fluxo de transcrição

```text
opts.Lang (ou "pt", "pt-BR", "en")
        │
        ▼
InnerTube (biblioteca isolada em innerTubeFetcher)
        │ falha
        ▼
timedtext fmt=json3
        │ falha
        ▼
timedtext fmt=srv3 (XML)
        │ falha
        ▼
ErrTranscriptUnavailable
```

Quando o idioma preferido não está disponível, o InnerTube recebe uma lista
vazia de idiomas e devolve a primeira faixa disponível.

## Decisões

- **Metadados:** YouTube Data API v3 com
  `part=snippet,statistics,contentDetails`. `contentDetails` é incluído para que
  `VideoData.Duration` tenha uma fonte confiável, ao custo de 0 unidades extras.
- **Transcrição:** InnerTube com fallback para `timedtext`; sem chave de API.
- **Capítulos:** `POST https://www.youtube.com/youtubei/v1/next` com cliente
  `WEB` e sem API key. A resposta traz
  `playerOverlays…multiMarkersPlayerBarRenderer.markersMap[*].value.chapters[*].chapterRenderer`
  com `title` (`simpleText` ou `runs`) e `timeRangeStartMillis`. Essa é a mesma
  lista exibida pelo player, incluindo capítulos automáticos que não estão na
  descrição. O parser é recursivo para resistir a mudanças de aninhamento.
  Falhas são silenciosas e acionam o fallback para timestamps da descrição.
- **Fallback de capítulos:** timestamps `0:00 Título` da descrição. O parser
  exige pelo menos dois capítulos e o primeiro em `00:00` para evitar falsos
  positivos. Capítulos do provedor têm prioridade e são normalizados (ordenação,
  trim e remoção de offsets duplicados).
- **Contrato congelado:** o core expõe apenas `Extract`, `ExtractWithOptions`,
  `ToMarkdown`, `ValidateVideoID`, `VideoData`, `TranscriptSegment`, `Chapter`,
  `Options` e os erros sentinela.

## Limitações conhecidas

- A biblioteca InnerTube usa o HTML da página do vídeo para localizar as faixas
  de legenda. Esse comportamento é interno à dependência e isolado no adaptador
  `core/transcript`; o código do projeto não faz scraping.
- A biblioteca InnerTube não aceita `context.Context` em sua API pública. O
  adaptador roda a chamada em goroutine e retorna imediatamente quando o `ctx`
  do chamador é cancelado; a goroutine interna é limitada pelo timeout de 30s da
  própria biblioteca.
- O endpoint `timedtext` pode responder vazio ou bloquear requisições; nesse
  caso o erro é explícito e o consumidor pode optar pelo modo parcial.
- O endpoint InnerTube `next` é interno do YouTube e pode mudar sem aviso. Por
  isso o parsing é recursivo, o cliente usa `ErrUnavailable` como sentinela
  interna e qualquer falha simplesmente aciona o fallback para a descrição sem
  interromper a extração.
- A lista de capítulos do InnerTube depende do vídeo possuir marcadores no
  player. Vídeos sem capítulos retornam `ErrNoChapters` e permanecem sem
  subtítulos `###`.
- A biblioteca InnerTube imprime mensagens de diagnóstico em `stdout` em alguns
  caminhos de retry/erro. Isso não afeta a API do core, mas a Fase 2 deve
  garantir que o `stdout` da CLI contenha apenas Markdown; alternativas são
  capturar a saída durante a chamada ou substituir o adaptador por uma
  implementação própria do InnerTube.
- `go list -deps ./core/...` mostra `os/exec` na árvore transitiva por causa de
  `google.golang.org/api` (certificado/proxy de credenciais). Nenhum pacote do
  projeto importa `os/exec` diretamente; a verificação de imports diretos usa
  `go list -f '{{join .Imports " "}}' ./core/...`.
