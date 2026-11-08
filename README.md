# transcript-tube-core

Biblioteca Go que extrai metadados, capítulos e transcrição de vídeos do YouTube
e formata o resultado como Markdown determinístico.

Este é o núcleo compartilhado pelos adaptadores `tyot` (CLI) e `transcript-tube-extension`
(extensão Chrome + Native Messaging Host). O contrato público está congelado em
`v0.2.0` e segue SemVer estrito.

## Recursos

- Metadados via YouTube Data API v3 (`snippet`, `statistics`, `contentDetails`).
- Transcrição em cascata: InnerTube → `timedtext` JSON3 → `timedtext` SRV3.
- Capítulos via InnerTube `next` (incluindo capítulos automáticos), com fallback
  para os timestamps da descrição.
- Orquestração paralela com `sync.WaitGroup` e modo de transcrição parcial.
- Markdown determinístico com golden tests.
- Nenhum scraping do HTML do YouTube.

## Instalação

```bash
go get github.com/dev-maicondmenezes/transcript-tube-core@v0.2.0
```

O pacote público fica no subdiretório `core`:

```go
import core "github.com/dev-maicondmenezes/transcript-tube-core/core"
```

## Uso

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    core "github.com/dev-maicondmenezes/transcript-tube-core/core"
)

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
    defer cancel()

    data, err := core.ExtractWithOptions(ctx, "dQw4w9WgXcQ", core.Options{
        Lang:                   "pt",
        AllowPartialTranscript: false,
    })
    if err != nil {
        log.Fatal(err)
    }

    fmt.Print(core.ToMarkdown(data))
}
```

### Variável de ambiente

`YOUTUBE_API_KEY` é a única fonte da API key da YouTube Data API v3:

```bash
export YOUTUBE_API_KEY="sua-chave"
```

Obtenha uma chave em <https://console.cloud.google.com/apis/credentials> e
restrinja-a à YouTube Data API v3. O core nunca carrega `.env` automaticamente.

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

Erros sentinela verificáveis com `errors.Is`: `ErrInvalidVideoID`,
`ErrMissingAPIKey`, `ErrVideoNotFound`, `ErrMetadataUnavailable` e
`ErrTranscriptUnavailable`.

Com `AllowPartialTranscript: true`, uma falha de transcrição devolve `VideoData`
com metadados **e** um erro que envolve `ErrTranscriptUnavailable`.

## Documentação

- [`docs/core.md`](docs/core.md) — API, erros e opções.
- [`docs/architecture.md`](docs/architecture.md) — camadas e dependências.
- [`docs/testing.md`](docs/testing.md) — convenções e fakes.
- [`spec.md`](spec.md) e [`plan.md`](plan.md) — especificação e plano do projeto.

## Desenvolvimento

Requisitos: Go 1.23.6+.

```bash
make setup
make check          # gofmt, vet, build e test
make test-race
make cover
```

Testes de integração (rede + API key) ficam atrás da build tag `integration` e
nunca rodam na suíte padrão:

```bash
YOUTUBE_API_KEY="$YOUTUBE_API_KEY" TEST_VIDEO_ID="$TEST_VIDEO_ID" \
  go test -tags=integration ./core/... -run Integration -v
```

## Releases

A versão fica em [`VERSION`](VERSION). O workflow `release.yml` roda em pushes
para `main`, cria a tag `vX.Y.Z` (se ainda não existir) e publica a GitHub
Release correspondente. Consulte [`RELEASING.md`](RELEASING.md).

## Licença

MIT — veja [`LICENSE`](LICENSE).
