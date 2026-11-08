# Core

Pacote: `github.com/dev-maicondmenezes/transcript-tube-core/core`

## API pública

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
    Lang                   string
    AllowPartialTranscript bool
}

func Extract(ctx context.Context, videoID string) (*VideoData, error)
func ExtractWithOptions(ctx context.Context, videoID string, opts Options) (*VideoData, error)
func ToMarkdown(data *VideoData) string
func ValidateVideoID(videoID string) error
```

### Construtores e injeção

Para consumidores que precisam substituir os adaptadores:

```go
type MetadataProvider interface {
    Fetch(ctx context.Context, videoID string) (*metadata.Video, error)
}

type TranscriptProvider interface {
    Fetch(ctx context.Context, videoID, lang string) ([]TranscriptSegment, error)
}

type ChapterProvider interface {
    Fetch(ctx context.Context, videoID string) ([]Chapter, error)
}

func NewExtractor(m MetadataProvider, t TranscriptProvider) *Extractor
func NewExtractorWithChapters(m MetadataProvider, t TranscriptProvider, c ChapterProvider) *Extractor

func (e *Extractor) Extract(ctx context.Context, videoID string, opts Options) (*VideoData, error)
```

## Erros sentinela

Todos os erros são contextualizados e verificáveis com `errors.Is`:

| Sentinela | Quando ocorre |
| :--- | :--- |
| `ErrInvalidVideoID` | O ID não tem 11 caracteres válidos. |
| `ErrMissingAPIKey` | `YOUTUBE_API_KEY` está vazia/ausente. |
| `ErrVideoNotFound` | `videos.list` retornou zero itens ou HTTP 404. |
| `ErrMetadataUnavailable` | Falha de API/rede/parse de metadados. |
| `ErrTranscriptUnavailable` | Todas as estratégias de transcrição falharam. |

Quando metadados e transcrição falham, o erro combina as duas sentinelas com
`errors.Join`.

## Idiomas

- `Options.Lang` informado: tentado primeiro.
- Sempre tenta `pt`, `pt-BR` e `en`.
- Se nada corresponder no InnerTube, usa a primeira faixa disponível.
- `timedtext` só recebe códigos concretos (nunca vazio).

## Modo parcial

```go
data, err := core.ExtractWithOptions(ctx, id, core.Options{
    AllowPartialTranscript: true,
})
if errors.Is(err, core.ErrTranscriptUnavailable) {
    // data está preenchido; data.Transcript está vazio
    fmt.Print(core.ToMarkdown(data))
}
```

Sem `AllowPartialTranscript`, qualquer falha de transcrição retorna `nil` e erro.

## Capítulos

A fonte prioritária é o endpoint InnerTube `next`, que devolve a mesma lista de
capítulos exibida pelo player do YouTube — incluindo capítulos gerados
automaticamente que **não** aparecem na descrição.

- Requisição: `POST https://www.youtube.com/youtubei/v1/next?prettyPrint=false`
  com `context.client.clientName = "WEB"`; não exige API key.
- O cliente percorre a resposta recursivamente procurando `chapterRenderer` e lê
  `title` (`simpleText` ou `runs`) e `timeRangeStartMillis`.
- Falhas de rede, HTTP ou parse são silenciosas para o consumidor: a extração
  continua e usa o fallback da descrição.

Fallback: timestamps da descrição do vídeo (`0:00 Título`), que também é uma
fonte oficial da YouTube Data API v3.

Regras do parser da descrição:

- A linha precisa começar com `mm:ss` ou `hh:mm:ss` (com espaço opcional,
  colchetes, bullet `-`, `*` ou `•` e separador `-`, `–`, `—`, `:`, `|`).
- São necessários pelo menos **2** capítulos válidos.
- O primeiro capítulo precisa começar em `00:00`.
- Timestamps inválidos são ignorados.
- A lista é ordenada por offset e offsets duplicados são descartados.

`VideoData.Chapters` é preenchido automaticamente por
`Extract`/`ExtractWithOptions`. Capítulos do InnerTube têm prioridade; se o
provedor falhar ou retornar vazio, o resultado da descrição é mantido.
Consumidores também podem montar `Chapters` manualmente; `ToMarkdown` normaliza
a lista antes de renderizar.

No Markdown, cada capítulo vira um subtítulo `###` dentro de
`## Transcrição`, ancorado no primeiro segmento cujo offset é maior ou igual ao
início do capítulo:

```markdown
## Transcrição

### [00:00] Introdução

[00:00] Primeiro segmento
[00:59] Segundo segmento

### [01:23] Go

[01:23] Terceiro segmento
```

Se a transcrição estiver indisponível, nenhum capítulo é renderizado e a seção
mantém `_Transcrição indisponível._`.

## Formato Markdown

O template é renderizado byte a byte conforme a especificação:

```markdown
# {Title}

**Canal:** {Channel}
**URL:** https://www.youtube.com/watch?v={ID}
**Publicado em:** 25/10/2009 07:00 UTC
**Duração:** 03:33
**Visualizações:** 1000000
**Likes:** 12345

---

## Descrição

{Description}

---

## Transcrição

[00:00] Primeiro segmento
[00:59] Segundo segmento
```

Detalhes:

- `PublishedAt` é formatado em UTC como `02/01/2006 15:04 UTC`; zero vira
  `desconhecida`.
- `Duration` vazia vira `desconhecida`; se os metadados não trouxerem duração e
  houver transcrição, o extrato usa o offset do último segmento como
  aproximação.
- Timestamps usam `mm:ss` e passam para `hh:mm:ss` a partir de uma hora.
- Offsets negativos são normalizados para `00:00`.
- Transcrição vazia mantém a seção com `_Transcrição indisponível._`.
- Capítulos são renderizados como `### [mm:ss] Título` dentro da seção de
  transcrição, na ordem cronológica.
- Números são decimais puros, sem separador de milhar.
- A saída termina com exatamente um newline.

## Exemplo de uso

```go
ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
defer cancel()

data, err := core.ExtractWithOptions(ctx, "dQw4w9WgXcQ", core.Options{Lang: "pt"})
if err != nil {
    log.Fatal(err)
}

fmt.Print(core.ToMarkdown(data))
```

## Decisão sobre `contentDetails`

A especificação cita `part=snippet,statistics`, mas `VideoData.Duration` não
teria fonte. Incluímos `contentDetails` para ler `duration` em ISO 8601
(`PT12M34S`). O custo continua sendo 1 unidade de cota por chamada.
