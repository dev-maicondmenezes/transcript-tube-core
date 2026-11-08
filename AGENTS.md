# AGENTS.md

Instruções para agentes que trabalham no repositório `transcript-tube-core`.

## Stack

- Go 1.23.6+.
- Biblioteca `core/` — extração, capítulos e formatação Markdown.
- `examples/core-demo/` — consumidor mínimo para testes manuais.
- Sem `cmd/`, sem Chrome, sem dependências de terminal.

## Regras de arquitetura

- `core` **não importa** Chrome, `os/exec`, terminal ou lógica de interface.
- `core/metadata` usa exclusivamente a YouTube Data API v3.
- `core/transcript` tenta InnerTube primeiro e `timedtext` como fallback.
- `core/chapters` busca capítulos no InnerTube `next`; a descrição é o fallback.
  Não há scraping de HTML.
- Toda chamada de rede recebe `context.Context`.
- Dependências externas ficam atrás de interfaces/adaptadores testáveis.
- Erros de domínio são sentinelas e devem ser verificáveis com `errors.Is`.
- O contrato público do core está congelado em `v0.2.0`; mudanças exigem
  justificativa, atualização dos consumidores e respeito ao SemVer.

## Anti-padrões

- Fazer scraping do HTML do YouTube para metadados.
- Colocar API keys no código, em arquivos versionados ou em logs.
- Carregar `.env` automaticamente dentro do core.
- Introduzir dependências de UI ou de terminal no pacote `core`.
- Adicionar `replace` local no `go.mod` da branch `main`.

## Comandos de verificação

```bash
go build ./...
go vet ./...
gofmt -l .
go test ./...
go test -race ./core/...
make check
```

Testes de integração (rede/API key, nunca no `go test ./...`):

```bash
YOUTUBE_API_KEY="$YOUTUBE_API_KEY" TEST_VIDEO_ID="$TEST_VIDEO_ID" \
  go test -tags=integration ./core/... -run Integration -v
```

## Política de testes

- Table-driven por padrão.
- Fakes para rede; a suíte padrão nunca acessa a internet.
- Golden files em `core/testdata/` revisados manualmente ao mudar o template.
- Testes de integração atrás da build tag `integration`.
- Metas: ≥ 70% em `core`; ≥ 60% nos adaptadores.

## Release

- A versão fica em `VERSION` (SemVer sem o prefixo `v`).
- O merge de um PR em `main` dispara `.github/workflows/release.yml`, que cria a
  tag `vX.Y.Z` e publica a GitHub Release.
- Nunca criar tags manualmente antes do merge, exceto em recuperação documentada.

Consulte [`README.md`](README.md), [`docs/`](docs/) e [`RELEASING.md`](RELEASING.md).
