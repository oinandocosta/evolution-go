# Transparência da distribuição nandocosta e manutenção em relação à Evolution GO

**Estado em 4 de novembro de 2026:** a nc.6 e `latest` estão publicadas para Linux AMD64. Os testes funcionais locais ARM64 passaram por emulação, mas o detector de corridas não conseguiu inicializar nesse ambiente. O workflow nativo está preparado; sua execução e a publicação ARM64 continuam pendentes. `latest` permanece na imagem AMD64 existente. Os caminhos de código neste documento partem da raiz do repositório.

Navegação: [notas da distribuição](README.md), [ARM64](ARM64.md), [manifesto canônico](RELEASE.json) e [futuras atualizações oficiais](#como-incorporar-futuras-atualizações-oficiais).

## Origem, escopo e atribuição

Esta é uma distribuição independente da Evolution GO, baseada no [código da Evolution Foundation](https://github.com/evolution-foundation/evolution-go), no commit [`9337afc47e10b86cc896a6f432240e40fee95dd1`](https://github.com/evolution-foundation/evolution-go/commit/9337afc47e10b86cc896a6f432240e40fee95dd1). O identificador `nandocosta` distingue os artefatos desta distribuição; não indica versão oficial, aprovação da Foundation ou aprovação dos autores das propostas citadas.

O módulo Go e o console mantêm a identidade de origem. `LICENSE`, `NOTICE` e `TRADEMARKS.md` foram preservados; a imagem passou a incluí-los explicitamente, junto com `/app/nandocosta-release.json`. A verificação de licença/ativação permanece. Não foram removidos mecanismos de ativação ou alteradas as condições do produto original. Os patches arquivados mantêm a identificação dos autores das contribuições. As referências aos PRs abaixo reconhecem essas origens sem atribuir aos seus autores as adaptações e decisões posteriores deste fork.

O inventário compara a base acima com o código desta distribuição e as preparações ARM64 ainda não publicadas. A versão pública do fork parte do commit upstream `9337afc47e10b86cc896a6f432240e40fee95dd1`, preserva seu histórico original e consolida as alterações posteriores. O histórico operacional local usado nas releases nc.2–nc.6 não será enviado ao repositório público. Identificadores de commits locais presentes em notas ou evidências antigas registram aquele processo de build; não são commits navegáveis deste fork nem provas de merge dos PRs oficiais.

O cadastro canônico dos commits de origem, partes selecionadas, dependências e schema é o [RELEASE.json](RELEASE.json). Os históricos [nc.2](RELEASE-nc2.md), [nc.3](RELEASE-nc3.md), [nc.4](RELEASE-nc4.md), [nc.5](RELEASE-nc5.md), as [notas atuais](README.md), o [diagnóstico do PR 199](PR-199.md) e as [evidências por release](evidence/README.md) complementam esse cadastro. A rastreabilidade pública usa o manifesto, os PRs/commits upstream citados, o código consolidado e essas evidências; não depende de publicar o histórico operacional local. Veja também a [procedência e sanitização dos patches](upstream-patches/README.md). Este documento explica o conjunto, sem criar uma segunda lista normativa de versões ou digests.

## Como interpretar a incorporação das contribuições

- **Integração funcional documentada:** o comportamento descrito foi incorporado; a entrada em `applied_prs` não deve ser interpretada como prova de merge integral ou igualdade byte a byte com todo o PR.
- **Integração adaptada ou parcial:** a distribuição aproveita uma proposta, mas conserva sua arquitetura, seleciona partes ou acrescenta validações/testes próprios. Os recortes explícitos estão descritos abaixo e no manifesto.
- **Correção própria:** complemento desenvolvido nesta distribuição, sem PR de origem específico comprovado para aquela implementação.
- **Não incorporado:** proposta examinada ou relacionada ao diagnóstico, mas não aplicada integralmente. Patch arquivado serve como referência e não prova, sozinho, inclusão na imagem.

Não afirmamos aplicação integral de um PR apenas porque seu patch está em `upstream-patches`. Para uma auditoria de igualdade exata, é necessário comparar a revisão registrada do PR com o diff da release relevante, considerando as alterações posteriores sobre os mesmos arquivos.

## Propostas da comunidade e recortes

| Origem | Integração nesta distribuição e ressalvas | Código e testes de referência |
| --- | --- | --- |
| [Issue 225](https://github.com/evolution-foundation/evolution-go/issues/225), [PR 227](https://github.com/evolution-foundation/evolution-go/pull/227), [PR 229](https://github.com/evolution-foundation/evolution-go/pull/229), [PR 230](https://github.com/evolution-foundation/evolution-go/pull/230) | Adaptação do handshake: aplicar a revisão Web ao `UserAgent.AppVersion` e hash de registro por `store.SetWAVersion`, antes de criar o cliente; timeout, validação, sincronização e proteção contra downgrade automático. Não adotado fallback ou piso de versão estático. O manifesto registra esses PRs como fontes conceituais, sem um head fixado para cada um. | `pkg/whatsmeow/service/handshake_version.go`, `whatsmeow.go`, `handshake_version_test.go`, `handshake_fetch_test.go`. |
| [PR 220](https://github.com/evolution-foundation/evolution-go/pull/220) | Salto inicial limitado do WhatsMeow para `a2381054887e`, que reconhece stanzas de status; depois substituído pela atualização maior da nc.5. | `go.mod`, `go.sum`; `nandocosta/upstream-patches/220-status-ack.patch`. |
| [PR 221](https://github.com/evolution-foundation/evolution-go/pull/221) | Store compartilhado, inicializado sob mutex e reaproveitando o pool PostgreSQL já aberto. Falha não fica memorizada: uma inicialização posterior pode tentar novamente. Proteção local para handle PostgreSQL ausente. | `getStoreContainer` em `whatsmeow.go`; `auth_store_test.go`. |
| [PR 224](https://github.com/evolution-foundation/evolution-go/pull/224) | Recorte de normalização do prefixo `+` em JIDs de usuários; não incorpora uma branch inteira nem altera a regra brasileira de nono dígito. | `pkg/utils/utils.go`, `utils_test.go`. |
| [PR 234](https://github.com/evolution-foundation/evolution-go/pull/234) | Runtime sincronizado e coordenação por geração, complementados pelo ciclo de QR próprio. A adaptação alcança todos os consumidores dos registros compartilhados. | `pkg/runtimestate/*`, `main.go`, serviços de instância/WhatsMeow e demais serviços listados no mapa. |
| [PR 149](https://github.com/evolution-foundation/evolution-go/pull/149) | Polling de QR reutiliza o cliente conectado e não reinicia sessão autenticada; inclui proteção local para ausência do ponteiro do cliente. | `pkg/instance/service/session_operations.go`, `session_operations_test.go`. |
| [PR 187](https://github.com/evolution-foundation/evolution-go/pull/187) | Desconexão conserva inscrições de eventos e configuração, atualizando o estado necessário. | `session_operations.go`, `session_operations_test.go`. |
| [PR 195](https://github.com/evolution-foundation/evolution-go/pull/195) | Corrige o tipo de evento `Archive` ao produzir o evento, evitando asserção inválida/panic. | `whatsmeow.go`, `archive_event_test.go`. |
| [PR 212](https://github.com/evolution-foundation/evolution-go/pull/212) | Dimensões de imagem e orientação EXIF, incluindo miniaturas nos caminhos de envio afetados. As notas nc.2 registram os dois commits dessa proposta; não é uma substituição geral do pipeline de mídia. | `pkg/sendMessage/service/send_service.go`, `dimensions_test.go`. |
| [PR 218](https://github.com/evolution-foundation/evolution-go/pull/218) | Solicitação de histórico enviada por `SendPeerMessage` ao dispositivo primário, conservando o conteúdo da solicitação. | `pkg/chat/service/chat_service.go`, `history_sync_test.go`. |
| [PR 136](https://github.com/evolution-foundation/evolution-go/pull/136) | Configurações parciais e reconexão sem apagar eventos/produtores omitidos; persistência explícita de `false`; validações de payload e escopo entre instância autenticada e UUID do caminho. | `pkg/instance/{model,handler,repository,service}`; testes de settings, persistência e regressões. |
| [PR 135](https://github.com/evolution-foundation/evolution-go/pull/135) | WebSocket com um writer por conexão, deadline de escrita, limpeza por identidade e registro desbloqueado durante I/O. Mantém uma conexão por instância e broadcasts. A falha de um destino não impede os demais; falha WebSocket não cancela a tentativa de webhook. | `pkg/events/websocket/*`, `websocket_delivery_test.go`; documentação de eventos. |
| [PR 137](https://github.com/evolution-foundation/evolution-go/pull/137) | Localização de contexto de menções nos wrappers de documentos com legenda, mídia e mensagens interativas, preservando o contexto existente. | `pkg/sendMessage/service/mention.go`, `send_service_mention_test.go`, `send_service.go`. |
| [PR 207](https://github.com/evolution-foundation/evolution-go/pull/207) | URL explícita e metadados do chamador têm prioridade; download de prévia limitado, upload de miniatura grande e fallback para miniatura inline/texto. Newsletters conservam miniatura inline. Limite local de pixels protege contra bitmaps excessivos. | `link_preview.go`, `link_preview_test.go`, `link_preview_fallback_test.go`, `send_service.go`. |
| [PR 237](https://github.com/evolution-foundation/evolution-go/pull/237) | **Parcial:** exige sessão autenticada e cliente da geração atual antes do envio, inclusive após atraso; remove dados sensíveis de logs selecionados. Não incorpora substituição do ciclo de vida, atualização de dependências/schema ou rotinas automáticas de logout/remoção. A atualização posterior do SDK pertence à nc.5, não a uma aplicação integral deste PR. | `connection_state.go`, `connection_state_test.go`, `send_service.go`, `webhook_producer.go`, `webhook_logging_test.go`, `main.go`. |
| [PR 122](https://github.com/evolution-foundation/evolution-go/pull/122) | Edições descriptografadas antes da normalização de identidades, com nonce e ID alvo validados, prazo de cinco segundos e cópia local do protobuf. Identifica edição/revogação e erro de descriptografia; não inventa conteúdo quando falta segredo. | `pkg/whatsmeow/service/message_edits.go`, `message_edits*_test.go`, `whatsmeow.go`. |
| [PR 120](https://github.com/evolution-foundation/evolution-go/pull/120) | **Parcial/adaptado:** consulta de avatar com prazo global, cancelamento, autenticação, normalização PN/LID e metadados preservados. Reutiliza o runtime sincronizado existente, sem índice paralelo de clientes. Não incorpora o `PictureURL` do PR 121 nem dependências/schema desse conjunto. | `pkg/user/service/avatar_query.go`, `user_service.go`, `pkg/user/handler/user_handler.go`, testes de avatar e contexto SQL. |

Os heads conhecidos das propostas integradas/selecionadas estão em [RELEASE.json](RELEASE.json) e os snapshots em [upstream-patches](upstream-patches). Patches de origem dão rastreabilidade; eventuais cópias sanitizadas precisam ser identificadas como tal. Não atribuir correções próprias aos autores dos PRs apenas por tratarem o mesmo problema.

### Propostas relacionadas que não foram aplicadas integralmente

- [PR 199](https://github.com/evolution-foundation/evolution-go/pull/199): o diagnóstico de handshake/pool é coberto por soluções equivalentes e complementos locais; não aplicado como pacote completo. Sua antiga estrutura de submódulo e o uso de `sync.Once` não foram adotados. Ver `nandocosta/PR-199.md`.
- [PR 206](https://github.com/evolution-foundation/evolution-go/pull/206): preferida a inicialização recuperável do PR 221 em vez de memorizar uma falha inicial com `sync.Once`.
- [PR 121](https://github.com/evolution-foundation/evolution-go/pull/121): não acrescentamos `PictureURL` ao `/user/info`. A imagem retorna metadados em `/user/avatar`; cache/download no consumidor não integra a imagem Go.
- [PR 208](https://github.com/evolution-foundation/evolution-go/pull/208): alteração da regra do nono dígito brasileiro não incorporada por esta distribuição.
- [PR 226](https://github.com/evolution-foundation/evolution-go/pull/226) e [PR 233](https://github.com/evolution-foundation/evolution-go/pull/233): foram referências da avaliação da atualização maior do SDK. A nc.5 realizou a atualização com adaptações e validação próprias; não há indicação de merge integral desses PRs no manifesto.
- Não foi adicionada integração nativa com Chatwoot nem serviço paralelo para isso. O suporte ARM64 é trabalho de build e validação desta distribuição, não um PR de aplicação funcional declarado como integrado.

## Correções e complementos próprios

### Ciclo de QR, pareamento e geração

`qr_lifecycle.go` concentra a publicação de QR, expiração/limite e conclusão de `PairSuccess` sob a mesma exclusão mútua, conferindo se a geração ainda é atual. Se o sucesso for processado primeiro, a rotação para sem derrubar a conexão autenticada. Expiração/limite encerram a tentativa sem `Logout` ou revogação de credenciais; o evento informa `forceLogout: false`.

Se timeout/limite encerrar a geração antes do processamento do sucesso, o evento tardio é ignorado e será necessária nova tentativa. Produtores assíncronos ainda podem entregar depois do pareamento um evento encaminhado anteriormente; não há replay reverso nem retirada de eventos já enfileirados. Os testes determinísticos cobrem as duas ordens, substituição/cancelamento, passkey e ausência de revogação.

A nc.5 trata `RotateADVSecret` no worker atual: renova o segredo no QR atual e nos seguintes, sem gastar um código do limite, sem substituir geração e sem logout. Não imprime os códigos ou segredos. Referências: `qr_lifecycle.go`, `qr_lifecycle_test.go`, `qr_secret_rotation_test.go`, `whatsmeow.go`.

### Handshake e armazenamento compartilhado

A revisão Web é aplicada ao handshake real, não apenas às propriedades do dispositivo. A busca tem timeout de dez segundos, validação HTTP e cache de uma hora; `ClientOutdated` invalida esse cache. A revisão automática inferior não substitui uma superior já aplicada; configuração manual explícita continua global. Isso não é atualização automática do código do WhatsMeow ou da imagem Docker e não garante compatibilidade com mudanças futuras do WhatsApp.

O holder compartilhado do store é necessário porque serviços têm cópias/recebedores por valor. O pool existente não é fechado nem multiplicado por cada inicialização. A falha de abertura/upgrade permite nova tentativa; o handle ausente é rejeitado. Não adicionamos um segundo gerenciador de reconexões.

### Compatibilidade do SDK na nc.5

`blocklist.go` adapta as novas assinaturas do SDK, mas conserva a resposta pública `DHash`/`JIDs`. Solicita lista completa com dHash vazio, ignora entradas inativas e prefere PN, com fallback para LID; resolve somente pelo contexto da instância. Entradas inválidas ou destinatário inexistente retornam 400.

`profile_status.go` adapta o texto Sobre para `SetStatusInput{Text: &text}`; conserva texto puro, inclusive vazio, sem acrescentar emoji ou duração própria. A permanência remota com duração zero da nova API não foi homologada com uma conta pareada. Esses adapters são locais; a alteração de formato de `Blocklist` interna do SDK não é repassada diretamente ao consumidor HTTP.

### Avatar e diagnósticos da nc.6

O total de oito segundos inclui preparação da sessão, espera de autenticação, resolução LID/PN e consulta. O pedido HTTP pode cancelar as consultas SQL/WhatsApp; uma sessão iniciada continua pertencendo ao serviço. Apenas a sessão proprietária e a geração atual podem ser usadas. Não há consulta a outra conta, logout ou reset de sessão para obter foto.

Sucesso preserva `message` e `data` com `url`, `id`, `type`, `direct_path` e `hash`; não há download da imagem nem retorno novo em base64. `ExistingID` fica vazio para que uma resposta condicional não suprima a URL.

| Situação | HTTP | Código estável |
| --- | --- | --- |
| Entrada inválida | 400 | `AVATAR_INVALID_NUMBER` |
| Sessão WhatsApp indisponível/não autenticada | 401 | `WHATSAPP_SESSION_UNAUTHENTICATED` |
| Consulta de foto recusada pelo WhatsApp | 403 | `PROFILE_PICTURE_UNAUTHORIZED` |
| Foto não disponível, resposta nula ou URL vazia | 404 | `PROFILE_PICTURE_NOT_SET` |
| Limite do WhatsApp | 429 | `WHATSAPP_RATE_LIMITED` |
| Falha interna ou inesperada | 500 | `AVATAR_QUERY_FAILED` |
| Timeout/cancelamento da consulta | 504 | `AVATAR_QUERY_TIMEOUT` |

O middleware de autenticação conserva seus próprios erros; o código de sessão acima não transforma qualquer HTTP 401 em ausência de foto. Os clientes precisam tratar o par status/código e continuar identificando erros reais. Recusa de consulta não prova, sozinha, que a foto é privada: autorização, tokens e políticas do WhatsApp podem interferir. Não há promessa de contornar privacidade ou renovação garantida de tokens.

Os diagnósticos novos registram categoria, estágio, IQ numérico, preview, duração, servidor de protocolo e identificador técnico da instância. Não imprimem telefone/JID completo do contato, URL assinada, token, XML ou erro upstream bruto. Essa é uma redução dos logs dos caminhos alterados, não uma declaração de que todos os logs do projeto foram auditados ou anonimizados.

### Ajustes de segurança e contrato

`main.go` omite `/ws` do access logger automático de Gin para não registrar o token da query; o logger não foi desligado para todo o servidor. A mensagem de token inválido não inclui seu valor. Logs selecionados não imprimem QR, payload completo de pareamento, links de pareamento, URLs de mídia ou conteúdo retornado por webhooks.

Os endpoints de configurações avançadas validam UUID, instância autenticada e correspondência entre token/instância e caminho; omissões/null mantêm as atualizações parciais, mas o corpo JSON nulo é inválido. `false` e texto vazio explícitos podem ser persistidos nos campos avançados. Em `/instance/connect`, listas de eventos vazias preservam as existentes (ou usam `MESSAGE` se ainda não existem), e strings vazias de produtores preservam seus valores; desligamento usa os valores explícitos esperados pelo contrato. Não confundir essas duas semânticas.

Antes do envio, o cliente deve estar conectado, autenticado e ainda pertencer à geração atual. A verificação é repetida depois de atrasos de presença, que são interrompíveis. Isso não acrescenta fila persistente, garantia de entrega ou reenvio automático.

## Dependências, toolchain e schema

| Componente | Base original | Estado nc.6 |
| --- | --- | --- |
| Go mínimo do módulo | `1.25.0` | `1.26.0` |
| Builder Docker | `golang:1.25.0-alpine` | `golang:1.26.10-alpine`, fixado por digest de índice |
| Runtime Docker | `alpine:3.19.1` | `alpine:3.23.7`, fixado por digest de índice |
| WhatsMeow | `v0.0.0-20260630180629-b572e5bcb92b` | `v0.0.0-20261102152715-54038813c47f` (passou por `a2381054887e` nas nc.1–nc.4) |
| `golang.org/x/exp` | `v0.0.0-20260611194520-c48552f49976` | `v0.0.0-20261014143641-f0251eb4aa3e` |
| `golang.org/x/net` | `v0.56.0` | `v0.61.0` |
| `google.golang.org/protobuf` | `v1.36.11` | `v1.36.12` |
| `github.com/petermattis/goid` | `v0.0.0-20260330135022-df67b199bc81` | `v0.0.0-20260918085751-abfca077860b` |
| `go.mau.fi/util` | `v0.9.10` | `v0.11.0` |
| `golang.org/x/crypto` | `v0.53.0` | `v0.58.0` |
| `golang.org/x/sync` | `v0.21.0` | `v0.24.0` |
| `golang.org/x/sys` | `v0.46.0` | `v0.49.0` |
| `golang.org/x/text` | `v0.38.0` | `v0.43.0` |
| `golang.org/x/tools` | `v0.46.0` | `v0.52.0` |

`go.sum` registra a resolução das dependências. `github.com/chai2010/webp v1.1.1`, `modernc.org/sqlite v1.33.1` e `golang.org/x/image` permanecem nas versões da base. CGO continua habilitado; WebP/JPEG e ferramentas de mídia continuam presentes. Não há troca do driver de autenticação ou migração para outro banco para oferecer ARM64. Não afirmamos que essas sejam as versões mais recentes em qualquer data futura: são versões fixadas desta release. A [comparação do WhatsMeow](https://github.com/tulir/whatsmeow/compare/a2381054887e16d22428430fdc4d2b027a7f29a7...54038813c47f3f351eb6fcaafab0e1b99d676850) documenta a atualização maior do SDK.

O armazenamento de autenticação passou de schema 14 para 16 na nc.5, pelas migrations oficiais do WhatsMeow: adicionam `whatsmeow_device.companion_meta_nonce` e `whatsmeow_chat_settings.wasa_root_secret_id`, textos não nulos com padrão vazio. A nc.6 não acrescenta nova migration. As tabelas/modelos da aplicação e as configurações existentes não foram substituídos por um schema paralelo. As migrations são executadas pela biblioteca ao abrir seu store.

O probe verificou três sessões sintéticas, 19 tabelas e 64 linhas em PostgreSQL descartável: upgrade, novos campos, reinicialização/idempotência, leitura/gravação pelo código anterior e nova leitura pela nc.5. O retorno não executou downgrade SQL nem edição manual da versão; esse teste controlado não é garantia universal de rollback em qualquer combinação futura.

## Build, distribuição e documentação acrescentados

- Dockerfile dividido em dependências, verificação, build e runtime. O build final depende de `go test ./...`, `go vet ./...` e `go test -race` nos 12 pacotes afetados. Usa `GOTOOLCHAIN=local`, versão explícita no binário/arquivo e metadados OCI identificando a distribuição e sua origem.
- Imagem final contém servidor, console pré-compilado, VERSION, manifesto de release e avisos originais. Não foi feito redesign nem alteração no código do console nesta série; a melhoria visual de Mensagem original pertence ao consumidor externo.
- `.dockerignore` permite fontes de testes para o estágio de verificação e exclui arquivos locais de ambiente, dados, evidências e arquivos tar. O runtime recebe somente os arquivos copiados explicitamente.
- [Workflow de publicação](../.github/workflows/publish_docker_image.yml) passou de push automático em `main` para acionamento manual, com `publish=false` por padrão, validação de VERSION/manifesto, namespace `nandocosta` e trava que exige repositório Docker Hub já público. A preparação atual usa runners nativos separados e recusa sobrescrever tags de release existentes. Não publica em `evoapicloud` e não promove `latest` automaticamente.
- `check-public-repository.py` e seus sete testes tratam falhas HTTP, redirecionamento, respostas inválidas/excessivas e repositório privado como bloqueio de publicação.
- `Dockerfile.runtime-tests`, probes de autenticação, snapshots/hash de fontes e logs por release dão rastreabilidade de builds e testes; não integram novas rotas da API. Os probes recusam alvos fora dos ambientes descartáveis definidos.
- `compose.override.yaml` contém somente a referência de imagem; não é uma stack completa e não deve substituir variáveis, bancos, volumes ou redes existentes.
- Swagger Go/JSON/YAML documenta o contrato de avatar, limites e novos códigos. `docs/wiki/recursos-avancados/events-system.md` corrige a descrição de WebSocket: autenticação global, UUID, eventos somente do servidor, uma conexão por instância, formato `{queue, payload}` com payload JSON como string, escrita com prazo e ausência de replay persistente.
- `RELEASE.json`, históricos, `PR-199.md`, `WHATS-MEOW-nc4.md`, `PUBLIC-REGISTRY.md`, patches e `evidence/*` são materiais novos de manutenção. Resultados de consumidor externo presentes em algumas evidências históricas não devem ser contabilizados como testes da imagem Go.

## ARM64: alterações preparadas e estado de validação

A preparação atual usa dois jobs nativos: `ubuntu-24.04` para Linux AMD64 e `ubuntu-24.04-arm` para Linux ARM64, sem QEMU no workflow. O workflow original já listava as duas arquiteturas; a distribuição nc.2 havia limitado a publicação a AMD64 porque só essa arquitetura estava homologada. Portanto não se apresenta multiarch como uma invenção inédita do projeto.

Cada job exige a arquitetura nativa e o builder com driver `docker`, executa os gates do Dockerfile canônico e roda os testes de envio/mídia na imagem final, exigindo pelo menos 43 testes e nenhum SKIP nessa suíte de runtime. Por padrão, exporta somente a imagem final, com checksums, metadados e logs, sem credenciais de registro. A imagem de verificação é usada e inspecionada no runner, mas não integra o arquivo tar exportado. QR live, PostgreSQL opt-in e migração cruzada não são executados por essa CI e estão identificados separadamente nos metadados.

No modo opcional de publicação, um job posterior aos dois jobs nativos confere os artifacts e os checksums, carrega as imagens e recusa tags existentes. Só então envia os sufixos e cria uma nova tag de versão com exatamente os dois digests/plataformas aprovados, sem `--append` nem promoção de `latest`. Essa preparação ainda não comprova execução real no GitHub; a nova versão exige gates aprovados e evidências antes de publicação. O procedimento completo está em [ARM64.md](ARM64.md).

`nandocosta/Dockerfile.runtime-tests` passou a aceitar `VERIFY_IMAGE` e `RUNTIME_IMAGE`, mantendo defaults, para executar o binário de testes ARM nas bibliotecas da imagem ARM final. `nandocosta/ARM64.md` descreve build/testes isolados, limitações de emulação e publicação preservando o artefato amd64 existente. Não houve alteração do algoritmo da aplicação nem atualização de dependência para ARM64.

O teste `TestSessionQueryDatabaseContext`, em `pkg/instance/repository/query_context_test.go`, dependia de timeout de parede muito curto antes da inicialização de GORM e podia expirar cedo sob emulação. Foi adaptado para cancelar o contexto somente quando a consulta chega ao driver simulado. Conserva o nome, o cenário, a verificação de propagação do cancelamento e as expectativas SQL; não aumenta timeout do produto. A regressão passou em 20 repetições amd64 e cinco com race antes da retomada do build ARM.

### Resultados locais ARM64 e limite do ambiente de emulação

Os resultados abaixo são da execução Linux ARM64 por QEMU 10.2.3, com Go 1.26.10. O Dockerfile canônico continua exigindo todos os gates, inclusive race. Para investigar os caminhos funcionais apesar da limitação do detector, foi criada uma candidata **somente local**, usando `build/arm64/Dockerfile.candidate`, temporário e ignorado, que remove apenas a etapa race. Essa candidata não é a imagem estável aprovada e não foi publicada.

| Verificação | Resultado e alcance |
| --- | --- |
| Testes Go normais e `go vet` | PASS em ARM64. |
| Mídia na candidata ARM final | 43 testes de nível principal passaram, zero falhas e zero skips; inclui WebP, JPEG, dimensões/orientação EXIF e miniatura PDF. |
| Smoke de runtime | PASS: executável ARM ELF, loader `/lib/ld-musl-aarch64.so.1`, versão, avisos de licença e exclusão de arquivos sensíveis; ferramentas de mídia disponíveis. |
| Handshake/QR com dispositivo efêmero | PASS: seis QR codes em 1,58 s, sem pareamento e sem publicar seu conteúdo. |
| Pool PostgreSQL | PASS: 64 inicializações concorrentes reutilizando pool limitado a duas conexões. |
| Migração entre arquiteturas | PASS: fixture criada pela nc.4 amd64/schema 14, aberta pela nc.6 ARM/schema 16; três sessões sintéticas, 19 tabelas, 64 leituras concorrentes, leitura/gravação, isolamento por instância e restart/idempotência. |
| Detector race ARM | **Não executou os testes:** o runtime abortou na inicialização com VMA de 47 bits, exigindo 48. Gate canônico não aprovado. |

Na migração cruzada, o SHA-256 dos dados/colunas legados permaneceu `de25362721022af8210c58fdfe68890e4a2cea42d4667e81818d0b2003fb8a79`. Os novos campos mantiveram o hash `2ae724aff0fbd8d4194c964de0f32132295021b756859bf1df8518fde076c7ab` depois de escrita e nova verificação. Não foram usados dados de produção, conta pareada, downgrade SQL ou edição manual do número de schema.

Os registros locais são `build/arm64/verification-build-retry.log`, `runtime-tests.log`, `runtime-smoke.log`, `qr-postgres-tests.log` e os JSON `auth-seed-amd64.json`, `auth-verify-arm64.json`, `auth-new-fields-arm64-after-upgrade.json` e `auth-verify-arm64-after-upgrade.json`. Esses caminhos ainda são evidências locais ignoradas: devem ser revisados/sanitizados e arquivados em evidências da release antes de serem apresentados como arquivos públicos do fork.

O aborto do detector é uma limitação de inicialização do ThreadSanitizer no layout de memória oferecido pela emulação, e não um teste que encontrou uma corrida no código da aplicação. O log informa `unsupported VMA range` e `Found 47 - Supported 48` antes de rodar os testes. A [issue LLVM 151931](https://github.com/llvm/llvm-project/issues/151931) documenta a classe de falha em AArch64 por QEMU; o [PR LLVM 220040](https://github.com/llvm/llvm-project/pull/220040) trata especificamente o suporte de 47 bits no runtime Go. A existência desse PR não prova que o binário do detector distribuído com a toolchain usada já o contém.

A inspeção do pacote oficial Go 1.27.3 nesta investigação também não ofereceu a correção pronta: seu [README do runtime race](https://github.com/golang/go/blob/go1.27.3/src/runtime/race/README) ainda registra o artefato ARM derivado da revisão LLVM `51bfeff0e4b0757ff773da6882f4d538996c9b04`. Não foi substituído o runtime oficial por um `.syso` próprio nem retirado o gate da imagem estável. A validação seguinte usa o workflow nativo preparado para aprovar race e repetir runtime. QR/PostgreSQL e migração cruzada exigem execução opt-in separada e registro de evidências. Nenhuma execução remota está confirmada neste checkpoint.

**Estado de publicação ARM:** a imagem canônica ARM final e o índice multiarch continuam pendentes pelo gate race não executado. Não houve publicação ARM nem alteração de `latest` nesta homologação; a imagem AMD64 nc.6 existente permanece intacta, com digest `sha256:17689c858288a334407f3003e5b46a2f106fb854ce76ad0873d343dce2b7e3ea`. Testar por QEMU não equivale a testar em hardware ARM físico. ARM de 32 bits (`linux/arm/v7`) permanece fora do escopo.

A estratégia preparada mantém a tag fixa amd64 nc.6 e seu digest filho. Depois de aprovar ARM64, publica-se uma variante própria e cria-se um índice com exatamente os dois manifests executáveis por digest; somente depois esse índice é promovido para `latest`. Um `--append` cego pode manter dois digests diferentes da mesma plataforma: os dry-runs mostraram que Buildx deduplica o digest idêntico, mas não garante unicidade de arquitetura. Inspecionar plataforma/digest e validar pulls antes da promoção. Nada dessa preparação exige redeploy ou alteração de dados do ambiente amd64 existente.

## Mapa de manutenção: área, arquivos, origem e regressões

| Área | Arquivos principais | Origem/decisão a preservar | Testes/evidência para revalidar |
| --- | --- | --- | --- |
| Handshake/revisão Web | `handshake_version.go`, `whatsmeow.go` | PRs 227/229/230 + validação/cache próprios | `handshake_version_test.go`, `handshake_fetch_test.go`; QR live opt-in. |
| Store/pool | `whatsmeow.go` (`getStoreContainer`) | PR 221 + holder compartilhado/recuperação | `auth_store_test.go`; 64 starts em pool PostgreSQL descartável. |
| Runtime e geração | `pkg/runtimestate/*`, `whatsmeow.go`, `instance_service.go`, `session_operations.go` | PR 234 + complementos; PRs 149/187 | `map_test.go`, `stop_signal_test.go`, `context_test.go`, `lifecycle_test.go`, `current_client_test.go`, `start_context_test.go`, `session_operations_test.go`. |
| Consumidores do runtime | `main.go`; serviços call, chat, community, group, label, message, newsletter, sendMessage e user | Substituir acesso a mapas não sincronizados sem criar funcionalidades novas nesses serviços | `go test ./...`, race nos pacotes afetados; regressões funcionais de cada área. |
| QR/passkey/ADV | `qr_lifecycle.go`, `whatsmeow.go` | Coordenação própria; manter geração atual e expiração sem logout | `qr_lifecycle_test.go`, `qr_secret_rotation_test.go`; QR live sem pareamento. |
| Settings e autorização | `advanced_settings.go`, `connect_settings.go`, `instance_model.go`, `instance_handler.go`, `instance_repository.go`, `instance_service.go` | PR 136 + adaptações de contrato/persistência | `advanced_settings_test.go` no handler/repositório; `settings_persistence_test.go`, `connect_settings*_test.go`, `query_context_test.go`. |
| WebSocket e webhook | `websocket_producer.go`, `webhook_producer.go`, `main.go`, despacho em `whatsmeow.go` | PR 135, partes do 237 e redução local de logs | `websocket_producer_test.go`, `websocket_lifecycle_test.go`, `websocket_delivery_test.go`, `webhook_logging_test.go`. |
| Edição/revogação recebida | `message_edits.go`, `whatsmeow.go` | PR 122 + validação de nonce/target/protobuf local | `message_edits_test.go`, `message_edits_handler_test.go`, `message_edits_fixtures_test.go`. |
| Archive/histórico/JID | `whatsmeow.go`, `chat_service.go`, `utils.go` | PRs 195/218/224 | `archive_event_test.go`, `history_sync_test.go`, `utils_test.go`. |
| Envio autenticado | `connection_state.go`, `send_service.go` | Partes do 237; geração correta antes/depois de atraso | `connection_state_test.go`; testes de envio no runtime. |
| Menções e mídia | `mention.go`, `send_service.go` | PRs 137/212; contexto existente e orientação correta | `send_service_mention_test.go`, `dimensions_test.go`; testes no runtime final. |
| Prévia de links | `link_preview.go`, `send_service.go` | PR 207 + limites/fallback próprios | `link_preview_test.go`, `link_preview_fallback_test.go`; testes no runtime final. |
| Avatar/contexto SQL | `avatar_query.go`, `avatar_diagnostics.go`, `user_service.go`, `user_handler.go`, métodos Context do repositório | PR 120 parcial + contrato nc.6 próprio | `avatar_query_test.go`, `avatar_diagnostics_test.go`, `avatar_handler_test.go`, `query_context_test.go`. |
| Compatibilidade perfil/bloqueio | `blocklist.go`, `profile_status.go`, `user_service.go`, `user_handler.go` | Adapters locais do SDK atualizado | `blocklist_test.go`, `profile_status_test.go`, `profile_compat_test.go`. |
| Dependências/schema | `go.mod`, `go.sum`, `auth-migration-probe/*`, `auth-schema-probe.go` | Pin do SDK e migrations oficiais 14→16 | integridade dos módulos; probe de oito etapas PostgreSQL; hashes de dados sintéticos. |
| Build/publicação/multiarch | `Dockerfile`, `.dockerignore`, workflow, runtime-tests, trava pública, `VERSION`, `RELEASE.json`, `ARM64.md` | Artefatos reproduzíveis por plataforma, tags fixas, latest promovida após checks | Go/vet/race e runtime em ambas arquiteturas; sete testes da trava; inspect por digest; pull anônimo; avisos e ausência de `.env`. |
| Documentação/evidência | Swagger, wiki de eventos e `nandocosta/*` | Contrato de API e origem rastreáveis; distinguir testes sintéticos de reais | Diff de contrato, evidência por release e revisão do inventário completo abaixo. |

## Validações demonstradas e limites

As releases amd64 nc.2–nc.6 têm registros de compilação/testes, QR sem pareamento, PostgreSQL descartável, testes de mídia e pull anônimo por artefato. Na nc.2, race cobria seis pacotes e 13 testes de mídia rodavam no runtime; na nc.3, dez pacotes e 43 testes de envio/mídia; nc.4–nc.6 usam race em 12 pacotes e 43 testes de runtime. Esses números não são uma contagem de testes de entrega real.

QR live com dispositivo efêmero demonstra handshake e geração de códigos, sem provar pareamento. Sessões/chaves sintéticas demonstram persistência e migração, sem provar autenticação remota. Os testes de runtime demonstram funcionamento dos caminhos de código e ferramentas, não entrega a outro telefone.

Em conferência posterior AMD64 foram observadas retomada de sessões existentes e resposta de foto própria. Há observação de uso bem-sucedido de fotos no consumidor, mas não foi realizada homologação ativa completa de envio/recebimento, edição, revogação, bloqueio e Sobre com conta pareada. O acompanhamento de mensagens foi limitado ao tráfego passivo disponível. Essas observações específicas não comprovam todos os cenários ou estabilidade prolongada.

O WebSocket continua sendo entrega por tentativa, sem fila persistente, replay ou garantia de ordem entre produtores assíncronos. Descriptografia de edição depende do segredo da mensagem original. Classificar revogação no evento não implementa exclusão em um CRM consumidor. Fotos podem continuar indisponíveis por decisões do WhatsApp; HTTP 403 não é falha interna a esconder.

## Configurações, sessões e sistemas externos

As mudanças de imagem preservam a intenção de reutilizar configuração, bancos, volumes, licença e credenciais. Não há rotina nova de logout em lote, exclusão de sessões ou recriação automática de instâncias para contornar falhas. Logout/exclusão solicitados explicitamente pelo usuário continuam existindo no produto.

`CONNECT_ON_STARTUP` é configuração operacional já existente, não nova funcionalidade do fork. Quando true, retoma registros com `connected=true`, filtrando por `CLIENT_NAME` quando definido; não é uma varredura indiscriminada de todas as contas pareadas. O uso de true no ambiente observado foi decisão de operação. O exemplo de imagem não deve sobrescrever outras configurações ao atualizar.

As correções Laravel/PHP do Evolve Agentes — leitura de `data.url`, download/armazenamento privado, aproveitamento de ID de foto, intervalo de nova consulta, preservação/limpeza de arquivos e UI de Mensagem original — pertencem a outro repositório. Não estão no executável ou console desta imagem. Testes PHP e sua UI não são evidência de teste Go nem devem ser apresentados como recurso nativo da distribuição.

## Como incorporar futuras atualizações oficiais

1. Manter em `RELEASE.json` a origem base conhecida. Na próxima release, registrar também o último commit upstream efetivamente incorporado e o novo intervalo revisado, sem substituir a base histórica nem criar um cadastro paralelo divergente. O esquema/nome desse registro deve ser decidido e validado com a mudança real.
2. Buscar o histórico oficial e comparar o intervalo novo com a base incorporada. Separar correções já absorvidas oficialmente, propostas ainda abertas, alterações locais necessárias e funcionalidades novas não solicitadas. Usar o mapa de áreas acima para não perder adaptações sobre os mesmos arquivos.
3. Antes de reaplicar uma correção, conferir se upstream já resolveu o caso e se o teste de regressão passa com a solução oficial. Quando absorvida, retirar a duplicação local com revisão explícita e preservar atribuição/rastreabilidade. Não reaplicar patches nem mesclar PRs indiscriminadamente.
4. Escolher merge, rebase ou integração pontual a partir do diff e da estrutura reais. O processo não impõe um método universal. Resolver conflitos de negócio/contrato conscientemente; compilar sem conflito textual não prova compatibilidade.
5. Revisar Go/toolchain, SDK, bibliotecas nativas, dependências transitivas, APIs e migrations. Pin de SDK, revisão Web de handshake e tag Docker são três mecanismos distintos. Não confundir atualização da revisão anunciada com atualização do código.
6. Se houver alteração de armazenamento, executar probes em banco descartável com sessões sintéticas: partir do schema atual, migrar pelo mecanismo oficial, comparar configurações/credenciais por hashes, reiniciar e verificar leitura/gravação. Avaliar retorno somente se a compatibilidade for demonstrada; não editar manualmente números de schema ou presumir downgrade seguro.
7. Executar regressões da área, testes Go/vet/race, testes de mídia na imagem final e checks de contrato/segredos/licença. Validar ambas as arquiteturas prometidas; registrar se a execução foi nativa ou emulada. Testes pareados precisam de ambiente/conta próprios e autorização adequada.
8. Usar uma nova tag versionada para a nova release. Atualizar `VERSION`, metadados Docker e o manifesto canônico de forma consistente, com fontes, partes incorporadas/excluídas, dependências/schema, evidências e limites. Não recompilar e sobrescrever silenciosamente versões fixas publicadas.
9. Publicar somente os artefatos testados, validar digests, as plataformas únicas do índice e pulls sem credenciais. Promover `latest` para o índice aprovado após os checks; manter as referências anteriores para retorno. Publicar `latest` não atualiza containers em execução.
10. Para aplicar no destino, comparar a stack e alterar apenas a referência de imagem e configurações especificamente autorizadas. Preservar volumes, bancos, redes, ativação e sessões. Fazer backup proporcional antes de mudanças persistentes que alterem/removam dados/estrutura, conforme o efeito real. Depois conferir retomada e tráfego sem executar logout em lote ou mensagens não autorizadas.

## Pendências de validação e manutenção

- Executar os gates canônicos em ARM nativo e registrar o resultado real. Os testes funcionais locais já passaram, mas o detector race não executou sob QEMU; isso mantém pendentes a imagem ARM estável, o índice e os pulls publicados. Identificadores novos só devem ser acrescentados depois de efetiva publicação.
- Conferir o inventário a cada release, junto com as alterações de código, documentação e workflow, mantendo o registro de origem e os testes correspondentes.
- Preservar a relação do fork com o repositório original e sua atribuição. O URL oficial identifica o upstream; a navegação relativa deste documento corresponde ao código desta distribuição.
- Preservar o manifesto canônico e os patches de origem; corrigir em release futura campos históricos imutáveis apenas por novo registro de estado, sem reescrever evidência do build publicado. Alguns manifestos históricos dizem `production_deployed=false` porque foram emitidos antes do deploy; observação posterior deve ser lida no registro posterior correspondente.
- Manter apenas evidências sintéticas/sanitizadas no material público. Não publicar Compose real, credenciais, QR, IDs de clientes, telefones, URLs assinadas ou detalhes de infraestrutura; distinguir uma cópia sanitizada do registro original e preservar sua procedência.
- Não declarar paridade integral com upstream, todos os PRs mesclados, homologação de hardware ARM físico, aprovação da comunidade ou ausência de manutenção do projeto oficial.

## Inventário de arquivos alterados frente à base

A lista cobre os arquivos de código, testes, build e documentação alterados ou acrescentados em relação à base original, incluindo esta preparação ARM64, os documentos da distribuição e as notas de procedência/sanitização de evidências e patches. Patches e evidências são materiais de rastreabilidade; sua extensão não equivale a código novo do produto. Arquivos ignorados de build e dados locais não integram esta lista.

<details>
<summary>Inventário completo: 208 arquivos</summary>

```text
.dockerignore
.github/workflows/publish_docker_image.yml
Dockerfile
README.md
VERSION
cmd/evolution-go/main.go
docs/docs.go
docs/swagger.json
docs/swagger.yaml
docs/wiki/recursos-avancados/events-system.md
go.mod
go.sum
nandocosta/ARM64.md
nandocosta/Dockerfile.runtime-tests
nandocosta/PR-199.md
nandocosta/PUBLIC-REGISTRY.md
nandocosta/README.md
nandocosta/RELEASE-nc2.md
nandocosta/RELEASE-nc3.md
nandocosta/RELEASE-nc4.md
nandocosta/RELEASE-nc5.md
nandocosta/RELEASE.json
nandocosta/TRANSPARENCIA.md
nandocosta/WHATS-MEOW-nc4.md
nandocosta/auth-migration-probe/fixtures.go
nandocosta/auth-migration-probe/main.go
nandocosta/auth-migration-probe/verify.go
nandocosta/auth-schema-probe.go
nandocosta/check-public-repository.py
nandocosta/compose.override.yaml
nandocosta/evidence/README.md
nandocosta/evidence/artifact.json
nandocosta/evidence/auth-schema-compatibility.log
nandocosta/evidence/live-qr.log
nandocosta/evidence/media-runtime.log
nandocosta/evidence/nc2/anonymous-pull.log
nandocosta/evidence/nc2/artifact.json
nandocosta/evidence/nc2/build.log
nandocosta/evidence/nc2/live-qr.log
nandocosta/evidence/nc2/media-runtime.log
nandocosta/evidence/nc2/postgres-auth-pool.log
nandocosta/evidence/nc2/push.log
nandocosta/evidence/nc2/recovery-regressions.log
nandocosta/evidence/nc2/registry.json
nandocosta/evidence/nc2/runtime-smoke.log
nandocosta/evidence/nc2/source-manifest.json
nandocosta/evidence/nc3/anonymous-pull.log
nandocosta/evidence/nc3/artifact.json
nandocosta/evidence/nc3/build.log
nandocosta/evidence/nc3/edited-consumer.log
nandocosta/evidence/nc3/live-qr.log
nandocosta/evidence/nc3/media-runtime.log
nandocosta/evidence/nc3/postgres-auth-pool.log
nandocosta/evidence/nc3/public-guard-tests.log
nandocosta/evidence/nc3/push.log
nandocosta/evidence/nc3/registry.json
nandocosta/evidence/nc3/runtime-smoke.log
nandocosta/evidence/nc3/source-manifest.json
nandocosta/evidence/nc3/verification.log
nandocosta/evidence/nc4/anonymous-pull.log
nandocosta/evidence/nc4/artifact.json
nandocosta/evidence/nc4/avatar-consumer.log
nandocosta/evidence/nc4/edited-message-ui.json
nandocosta/evidence/nc4/live-qr.log
nandocosta/evidence/nc4/media-runtime.log
nandocosta/evidence/nc4/postgres-auth-pool.log
nandocosta/evidence/nc4/postgres-session-nc3.log
nandocosta/evidence/nc4/postgres-session-nc4.log
nandocosta/evidence/nc4/registry.json
nandocosta/evidence/nc4/runtime-smoke.log
nandocosta/evidence/nc4/source-manifest.json
nandocosta/evidence/nc4/test-summary.json
nandocosta/evidence/nc4/verification.log
nandocosta/evidence/nc4/whatsmeow-compatibility.log
nandocosta/evidence/nc4/whatsmeow-latest.json
nandocosta/evidence/nc5/anonymous-pull.log
nandocosta/evidence/nc5/artifact.json
nandocosta/evidence/nc5/auth-01-nc4-seed.json
nandocosta/evidence/nc5/auth-02-nc4-baseline.json
nandocosta/evidence/nc5/auth-03-nc5-upgrade.json
nandocosta/evidence/nc5/auth-04-nc5-new-fields.json
nandocosta/evidence/nc5/auth-05-nc4-read-schema16.json
nandocosta/evidence/nc5/auth-06-nc4-save-schema16.json
nandocosta/evidence/nc5/auth-07-nc5-after-return.json
nandocosta/evidence/nc5/auth-08-final-image-verify.json
nandocosta/evidence/nc5/auth-migration-readme.md
nandocosta/evidence/nc5/auth-summary.json
nandocosta/evidence/nc5/final-build.log
nandocosta/evidence/nc5/go-tests-vet-race.log
nandocosta/evidence/nc5/live-qr.log
nandocosta/evidence/nc5/module-verification.log
nandocosta/evidence/nc5/postgres-pool.log
nandocosta/evidence/nc5/publication-preflight.json
nandocosta/evidence/nc5/push.log
nandocosta/evidence/nc5/qr-adv-rotation.log
nandocosta/evidence/nc5/registry-tag.json
nandocosta/evidence/nc5/runtime-smoke.log
nandocosta/evidence/nc5/runtime-tests.log
nandocosta/evidence/nc5/source-manifest.json
nandocosta/evidence/nc5/test-summary.json
nandocosta/evidence/nc5/whatsmeow-latest.json
nandocosta/evidence/nc6/anonymous-pull.log
nandocosta/evidence/nc6/artifact.json
nandocosta/evidence/nc6/final-build.log
nandocosta/evidence/nc6/go-tests-vet-race.log
nandocosta/evidence/nc6/publication-preflight.json
nandocosta/evidence/nc6/push.log
nandocosta/evidence/nc6/runtime-smoke.log
nandocosta/evidence/nc6/runtime-tests.log
nandocosta/evidence/nc6/source-manifest.json
nandocosta/evidence/postgres-auth-pool.log
nandocosta/evidence/runtime-smoke.log
nandocosta/evidence/users-schema-comparison.json
nandocosta/test_public_repository.py
nandocosta/upstream-patches/120.patch
nandocosta/upstream-patches/122.patch
nandocosta/upstream-patches/135.patch
nandocosta/upstream-patches/136.patch
nandocosta/upstream-patches/137.patch
nandocosta/upstream-patches/149.json
nandocosta/upstream-patches/149.patch
nandocosta/upstream-patches/187.json
nandocosta/upstream-patches/187.patch
nandocosta/upstream-patches/195.json
nandocosta/upstream-patches/195.patch
nandocosta/upstream-patches/207.patch
nandocosta/upstream-patches/212.json
nandocosta/upstream-patches/212.patch
nandocosta/upstream-patches/218.json
nandocosta/upstream-patches/218.patch
nandocosta/upstream-patches/220-status-ack.patch
nandocosta/upstream-patches/221-auth-store.patch
nandocosta/upstream-patches/224-jid-prefix.patch
nandocosta/upstream-patches/234-runtime.patch
nandocosta/upstream-patches/237.patch
nandocosta/upstream-patches/README.md
pkg/call/service/call_service.go
pkg/chat/service/chat_service.go
pkg/chat/service/history_sync_test.go
pkg/community/service/community_service.go
pkg/events/webhook/webhook_logging_test.go
pkg/events/webhook/webhook_producer.go
pkg/events/websocket/websocket_lifecycle_test.go
pkg/events/websocket/websocket_producer.go
pkg/events/websocket/websocket_producer_test.go
pkg/group/service/group_service.go
pkg/instance/handler/advanced_settings.go
pkg/instance/handler/advanced_settings_test.go
pkg/instance/handler/instance_handler.go
pkg/instance/model/instance_model.go
pkg/instance/repository/advanced_settings_test.go
pkg/instance/repository/instance_repository.go
pkg/instance/repository/query_context_test.go
pkg/instance/repository/settings_persistence_test.go
pkg/instance/service/connect_settings.go
pkg/instance/service/connect_settings_regression_test.go
pkg/instance/service/connect_settings_test.go
pkg/instance/service/instance_service.go
pkg/instance/service/session_operations.go
pkg/instance/service/session_operations_test.go
pkg/label/service/label_service.go
pkg/message/service/message_service.go
pkg/newsletter/service/newsletter_service.go
pkg/runtimestate/context_test.go
pkg/runtimestate/map.go
pkg/runtimestate/map_test.go
pkg/runtimestate/stop_signal.go
pkg/runtimestate/stop_signal_test.go
pkg/sendMessage/service/connection_state.go
pkg/sendMessage/service/connection_state_test.go
pkg/sendMessage/service/dimensions_test.go
pkg/sendMessage/service/link_preview.go
pkg/sendMessage/service/link_preview_fallback_test.go
pkg/sendMessage/service/link_preview_test.go
pkg/sendMessage/service/mention.go
pkg/sendMessage/service/send_service.go
pkg/sendMessage/service/send_service_mention_test.go
pkg/user/handler/avatar_handler_test.go
pkg/user/handler/profile_compat_test.go
pkg/user/handler/user_handler.go
pkg/user/service/avatar_diagnostics.go
pkg/user/service/avatar_diagnostics_test.go
pkg/user/service/avatar_query.go
pkg/user/service/avatar_query_test.go
pkg/user/service/blocklist.go
pkg/user/service/blocklist_test.go
pkg/user/service/profile_status.go
pkg/user/service/profile_status_test.go
pkg/user/service/user_service.go
pkg/utils/utils.go
pkg/utils/utils_test.go
pkg/whatsmeow/service/archive_event_test.go
pkg/whatsmeow/service/auth_store_test.go
pkg/whatsmeow/service/current_client_test.go
pkg/whatsmeow/service/handshake_fetch_test.go
pkg/whatsmeow/service/handshake_version.go
pkg/whatsmeow/service/handshake_version_test.go
pkg/whatsmeow/service/lifecycle_test.go
pkg/whatsmeow/service/message_edits.go
pkg/whatsmeow/service/message_edits_fixtures_test.go
pkg/whatsmeow/service/message_edits_handler_test.go
pkg/whatsmeow/service/message_edits_test.go
pkg/whatsmeow/service/qr_lifecycle.go
pkg/whatsmeow/service/qr_lifecycle_test.go
pkg/whatsmeow/service/qr_secret_rotation_test.go
pkg/whatsmeow/service/start_context_test.go
pkg/whatsmeow/service/websocket_delivery_test.go
pkg/whatsmeow/service/whatsmeow.go
```

</details>
