# Processo de release

Este repositório usa um fluxo de release acionado pelo merge de um PR em `main`.

## Pré-requisitos

- CI verde no PR (`ci/test`).
- `VERSION` atualizado no PR (SemVer sem o prefixo `v`).
- `CHANGELOG.md` atualizado.
- `go.mod` sem `replace` local.

## Fluxo

1. Abra um PR de uma branch (por exemplo `release/v0.2.0`) para `main`.
2. Aguarde `ci/test`.
3. Faça o merge.
4. O workflow `release.yml` roda em `push` para `main`:
   - valida build, vet e testes;
   - lê `VERSION`;
   - se a tag `vX.Y.Z` ainda não existir, cria a tag anotada e a envia;
   - cria a GitHub Release com notas.
5. Para consumir a versão:

   ```bash
   go get github.com/dev-maicondmenezes/transcript-tube-core@v0.2.0
   ```

## Idempotência

Se a tag já existir, o workflow não recria a release. Para republicar uma versão,
incremente `VERSION` em um novo PR; nunca reutilize uma tag publicada.

## Ordem no projeto

O repositório `transcript-tube-cli` depende deste módulo. Faça o merge e aguarde
a tag `vX.Y.Z` do core antes de fazer o merge do PR do CLI.
