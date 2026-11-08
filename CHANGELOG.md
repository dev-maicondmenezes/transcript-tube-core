# Changelog

Todas as mudanças notáveis deste projeto são documentadas neste arquivo.

O formato segue [Keep a Changelog](https://keepachangelog.com/pt-BR/1.1.0/)
e o projeto adere a [SemVer](https://semver.org/lang/pt-BR/).

## [Unreleased]

## [0.2.0] - 2026-11-08

Primeira versão pública do módulo independente `transcript-tube-core`,
migrada do monorepo original (`v0.1.0-core`).

### Added

- API pública congelada: `Extract`, `ExtractWithOptions`, `ToMarkdown`,
  `ValidateVideoID`, `VideoData`, `TranscriptSegment`, `Chapter` e `Options`.
- Metadados via YouTube Data API v3 com `part=snippet,statistics,contentDetails`.
- Transcrição em cascata: InnerTube → `timedtext` JSON3 → `timedtext` SRV3.
- Capítulos oficiais/automáticos via InnerTube `next`, com fallback para
  timestamps da descrição.
- Orquestração paralela com `sync.WaitGroup`, modo parcial e `errors.Join`.
- Markdown determinístico com capítulos `###` dentro de `## Transcrição`.
- Golden tests, fakes de rede e testes de integração atrás da build tag
  `integration`.
- Exemplo executável em `examples/core-demo`.

### Changed

- Módulo renomeado para `github.com/dev-maicondmenezes/transcript-tube-core`.
- Import path público em `github.com/dev-maicondmenezes/transcript-tube-core/core`.
- User-Agent das chamadas de rede atualizado para `transcript-tube-core/0.2.0`.
