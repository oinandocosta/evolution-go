# Linux ARM64 e imagens multiarch

A imagem `nandocosta/evolution-go:latest` está publicada para `linux/amd64` e `linux/arm64`, após aprovação dos gates canônicos em runners nativos separados. A tag fixa nc.6 AMD64 foi preservada e o índice inclui a variante ARM64 aprovada. A adição de arquitetura não altera VERSION, RELEASE.json, dependências, configurações ou sessões existentes.

`latest` e `0.7.2-nc.6-multiarch` apontam para o mesmo índice com as duas variantes. O Docker seleciona a variante compatível com o host. ARM64 exige sistema operacional de 64 bits; ARM de 32 bits (`linux/arm/v7`) permanece fora do escopo.

## Homologação nativa e publicação

A [execução 41948461407](https://github.com/oinandocosta/evolution-go/actions/runs/41948461407) concluiu com sucesso no commit [`5481e977006ab0b3c11ce2c976f78941f2ffa0fb`](https://github.com/oinandocosta/evolution-go/commit/5481e977006ab0b3c11ce2c976f78941f2ffa0fb). Os jobs AMD64 e ARM64 passaram no Dockerfile canônico, incluindo testes Go, vet e race. Na imagem ARM final, passaram 43 testes de runtime, sem falhas ou SKIP, incluindo WebP, JPEG, orientação EXIF e miniatura PDF.

A CI rodou em modo build-only. O tar ARM validado foi publicado posteriormente, sem recompilar, com Image ID `sha256:f901a1689ff5b1c4b56f754c1692ba376c1bb7e42bdebe21c187ac7e3e23f4cf`. O novo build AMD64 da CI foi usado para validação; o índice publicado reutiliza o manifest AMD64 já existente.

| Tag | Plataformas e finalidade |
| --- | --- |
| `nandocosta/evolution-go:latest` | Linux AMD64 e ARM64; índice promovido após validação. |
| `nandocosta/evolution-go:0.7.2-nc.6-multiarch` | O mesmo índice aprovado das duas arquiteturas. |
| `nandocosta/evolution-go:0.7.2-nc.6` | Tag fixa AMD64 existente, preservada. |
| `nandocosta/evolution-go:0.7.2-nc.6-arm64` | Variante ARM64 nativa aprovada. |

| Identidade | Digest |
| --- | --- |
| Tag fixa nc.6 e filho AMD64 preservado | `sha256:17689c858288a334407f3003e5b46a2f106fb854ce76ad0873d343dce2b7e3ea` |
| Tag nc.6-arm64 e filho ARM64 | `sha256:b95241edd9c86a6add88bc448f6460bc53afe96df79454c1c6f04bb58e3df668` |
| Índice nc.6-multiarch e latest | `sha256:e75b89deb9c43157602c93e42d75d26d4aa01d0ef3db20ea9265f760bb663121` |

Pulls anônimos de `latest` em ambas as plataformas retornaram os Image IDs esperados e passaram no smoke de runtime. A imagem ARM usa o loader `/lib/ld-musl-aarch64.so.1`. O [registro do artefato](evidence/arm64/artifact.json) contém fonte Git, hashes, jobs e verificações. Não houve redeploy em produção; configurações, bancos, volumes e conexões existentes foram preservados.

QR live, PostgreSQL opt-in e migração entre arquiteturas não rodaram nessa CI nativa. O histórico local abaixo registra esses testes separadamente, com dados sintéticos e sem pareamento. A execução nativa em VM hospedada também não representa homologação de todo equipamento ARM de destino ou teste prolongado de entrega real.

## Histórico local de 4 de novembro de 2026 — QEMU

A execução ARM64 emulada passou na suíte Go e no vet. Uma candidata funcional, construída com um Dockerfile temporário e ignorado apenas para concluir a investigação, passou em 43 testes do pacote de envio dentro das bibliotecas da imagem final, sem SKIP, incluindo WebP, JPEG, orientação EXIF e PDF. Também passaram o teste de PostgreSQL com 64 inicializações concorrentes e o handshake real que gerou seis QR Codes, sem pareamento.

Três sessões sintéticas criadas em AMD64 com o schema 14 da dependência antiga foram lidas e migradas em ARM64 para o schema 16 da dependência atual. A verificação de APIs do store, configurações, chaves, sessões, novos campos, reabertura e 64 leituras concorrentes preservou o checksum dos dados legados. O teste usou banco descartável, sem conexões ou dados de produção. Isso comprova o cenário sintético testado; envio e recebimento reais continuam exigindo homologação própria.

O build canônico foi interrompido pelo detector de corridas antes de executar os testes instrumentados: `ThreadSanitizer: unsupported VMA range; Found 47 - Supported 48`. É uma limitação do runtime de detecção Go/LLVM nesse ambiente QEMU com endereçamento de 47 bits. A correção upstream específica é o [LLVM PR 220040](https://github.com/llvm/llvm-project/pull/220040); os runtimes oficiais Go 1.26.10 e 1.27.3 consultados ainda usam o detector anterior.

O Dockerfile principal manteve o gate race obrigatório. Aquela candidata funcional sem race ficou somente local e não foi publicada. Na etapa posterior, o build canônico passou em ARM64 nativo; a imagem publicada veio dessa execução nativa, conforme o registro acima.

## Compilar e testar sem publicar

O Dockerfile principal compila com CGO e executa testes Go, vet e detector de corridas antes do estágio final. As bibliotecas C precisam corresponder à arquitetura-alvo. Não basta definir GOARCH=arm64 usando o compilador C de amd64.

Os exemplos abaixo usam shell POSIX e tags locais separadas das releases publicadas. Criar um builder isolado se ele ainda não existir; não é necessário torná-lo o padrão:

```sh
docker buildx create --name evolution-nc-arm-check --driver docker-container --bootstrap
docker buildx inspect evolution-nc-arm-check
```

Confirmar que o builder oferece `linux/arm64` e que os índices das imagens-base fixadas no Dockerfile contêm essa variante. Em máquina amd64, o build e a execução ARM64 usam emulação QEMU; em host ARM64, podem executar nativamente. Emulação pode ser consideravelmente mais lenta e não equivale a testar em equipamento ARM físico. Os comandos canônicos abaixo exigem que o gate race também passe; o ambiente emulado local descrito acima continua bloqueado nesse ponto.

```sh
docker buildx build --builder evolution-nc-arm-check --platform linux/arm64 --target verification --load -t evolution-go-arm64-verify:local .
docker buildx build --builder evolution-nc-arm-check --platform linux/arm64 --load -t evolution-go-arm64:local .
docker image inspect evolution-go-arm64:local --format '{{.Os}}/{{.Architecture}}'
docker run --rm --platform linux/arm64 evolution-go-arm64:local -h
docker run --rm --platform linux/arm64 --entrypoint sh evolution-go-arm64:local -c 'ldd /app/server && ffmpeg -version && pdftoppm -v'
```

Compilar o executável de testes com a mesma arquitetura e executá-lo nas bibliotecas da imagem final. As duas tags locais acima foram carregadas no Docker Engine. Para usá-las em FROM, o exemplo seguinte usa o builder `desktop-linux` do Docker Desktop, com driver `docker`; conferir o driver com inspect antes de continuar. Esse nome é do ambiente local; os runners Linux do workflow usam o builder `default`, também com driver `docker`. Um builder com driver `docker-container` não acessa automaticamente essas tags locais. Nesse caso, disponibilizar as imagens em um registro de testes e informar suas referências nos argumentos, sem sobrescrever as tags de release.

```sh
docker buildx inspect desktop-linux
docker buildx build --builder desktop-linux --platform linux/arm64 -f nandocosta/Dockerfile.runtime-tests --build-arg VERIFY_IMAGE=evolution-go-arm64-verify:local --build-arg RUNTIME_IMAGE=evolution-go-arm64:local --load -t evolution-go-arm64-runtime-tests:local .
docker run --rm --platform linux/arm64 evolution-go-arm64-runtime-tests:local -test.v
```

Sem `-test.run`, o último comando executa todos os testes compilados do pacote de envio. Confirmar especialmente a codificação/decodificação WebP, conversão JPEG, orientação EXIF e renderização de PDF. O teste de PDF pode ser omitido quando pdftoppm não existe; para aprovar esta imagem, a ferramenta deve existir e o teste deve passar, sem SKIP.

Validar também inicialização, persistência e retomada usando bancos descartáveis e sessões sintéticas, sem dados de produção. QR sem pareamento, testes de mídia e testes unitários não comprovam entrega real: pareamento, envio e recebimento precisam de homologação própria. Repetir os testes de runtime para amd64 quando houver alteração que afete seu artefato.

## Publicar preservando o artefato amd64

O workflow manual preparado usa `ubuntu-24.04` para AMD64 e `ubuntu-24.04-arm` para ARM64, sem QEMU. Cada job exige a arquitetura nativa e o driver Docker local, executa o Dockerfile canônico com testes Go, vet e race e depois todos os testes do pacote de envio na imagem final, com pelo menos 43 testes e nenhum SKIP. A execução real desse workflow no GitHub foi aprovada no [run 41948461407](https://github.com/oinandocosta/evolution-go/actions/runs/41948461407), em modo build-only. QR live, PostgreSQL opt-in e migração entre arquiteturas não são repetidos por esse workflow; seus metadados declaram essa cobertura separadamente.

Por padrão, `publish` é falso: cada arquitetura exporta apenas a imagem final em um arquivo tar, acompanhado de SHA256SUMS, metadados, logs e inspect das imagens final e verification como artifact. A imagem verification permanece somente no runner para os testes. Esse modo não exige credenciais do Docker Hub e permite baixar, conferir os checksums e carregar a imagem com `docker load` antes de publicar localmente com as credenciais já existentes.

Quando `publish` está marcado, o guard exige que o repositório Docker Hub já seja público e que a tag canônica e os dois sufixos de arquitetura ainda não existam. Apenas depois de ambos os jobs nativos passarem, um job separado confere e carrega os artifacts, publica os sufixos e monta a nova tag de versão com exatamente dois manifests por digest. Uma falha de push pode deixar um sufixo disponível; ela não autoriza a criação do índice sem as duas variantes. O workflow não promove `latest`. Para futuras releases, atualizar VERSION e RELEASE.json conforme o processo existente e usar uma nova tag fixa; não recompilar e sobrescrever uma release já publicada apenas para adicionar arquitetura.

Procedimento usado para acrescentar ARM64 à `latest` sem reconstruir o amd64 validado:

1. Registrar o digest amd64 atualmente aprovado e o conteúdo de `latest` antes da promoção.
2. Concluir e registrar os testes ARM64. Publicar essa mesma imagem em uma tag própria de versão e arquitetura, sem substituir as tags amd64 existentes.
3. Inspecionar os dois manifests por digest e conferir `linux/amd64` e `linux/arm64`, versão, conteúdo e avisos de licença. Usar os manifests das imagens, sem confundir índices ou atestações com variantes executáveis.
4. Criar primeiro uma tag de índice candidata com `docker buildx imagetools create`, referenciando explicitamente os dois digests aprovados, sem `--append`. Inspecionar esse índice e exigir exatamente duas plataformas únicas, `linux/amd64` e `linux/arm64`, confirmando que o digest filho amd64 permaneceu idêntico ao registrado. Dois manifests diferentes de AMD64 podem gerar entradas duplicadas se forem combinados inadvertidamente.
5. Verificar pull anônimo e seleção das variantes por plataforma, incluindo o funcionamento do ARM64 após pull. Somente então promover o mesmo índice validado para `latest`.
6. Manter a tag fixa amd64 e os digests anteriores disponíveis para retorno. Não alterar a stack de produção nem suas configurações por causa da publicação ARM64.

A promoção muda o digest do índice `latest`; isso é esperado mesmo mantendo o mesmo manifest filho amd64. Containers em execução não são atualizados automaticamente. Aplicar uma nova imagem no Portainer continua exigindo pull e redeploy.

## Referências

- [Builds para múltiplas plataformas](https://docs.docker.com/build/building/multi-platform/).
- [Runners nativos do GitHub Actions](https://docs.github.com/en/actions/reference/runners/github-hosted-runners).
- [Driver Docker e acesso às imagens locais](https://docs.docker.com/build/builders/drivers/docker/).
- [Detector de corridas da distribuição Go 1.27.3](https://raw.githubusercontent.com/golang/go/go1.27.3/src/runtime/race/README).
- [Criação de índices com imagetools](https://docs.docker.com/reference/cli/docker/buildx/imagetools/create/).
