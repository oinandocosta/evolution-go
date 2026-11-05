# Evolution GO — distribuição nandocosta 0.7.2-nc.2

Imagem `nandocosta/evolution-go:0.7.2-nc.2`, plataforma `linux/amd64`, compilada, testada e [publicada no Docker Hub](https://hub.docker.com/r/nandocosta/evolution-go/tags?name=0.7.2-nc.2) em 04/11/2026. Build, testes, detector de corridas, QR sem pareamento, PostgreSQL descartável e mídia/runtime passaram. O repositório público e o pull anônimo foram verificados. Não foi instalada em produção. Por decisão do responsável, não foi usado WhatsApp pessoal para homologação. Pareamento real, envio, recebimento e retomada de uma conta realmente pareada continuam pendentes.

O nome `nandocosta` identifica a distribuição. O código, o console, a identificação de origem, os direitos autorais, o arquivo LICENSE e a ativação/licença da Evolution GO foram preservados. Não há remoção de verificação de licença.

## Diagnóstico e alcance

Na imagem de produção `evoapicloud/evolution-go:0.7.2-beta`, o log obtém a revisão Web atual e depois envia uma revisão antiga no handshake: `Client outdated (405)`. Isso também impede a criação do QR diretamente no gerenciador da Evolution GO. O HTTP 400 apresentado pelo nosso sistema propaga essa falha externa.

Alterar somente `DeviceProps.Version` não atualiza o `UserAgent.AppVersion` nem o hash de registro do WhatsMeow. A correção aplica a revisão obtida da Web com `store.SetWAVersion` antes de criar o cliente. Não fixa a revisão de hoje na imagem. A consulta tem prazo de 10 segundos, rejeita respostas HTTP sem sucesso e conserva o cache existente de uma hora. Um evento `ClientOutdated` invalida esse cache, para uma nova inicialização consultar novamente. Uma revisão automática inferior não substitui a já aplicada; as variáveis explícitas de versão continuam permitindo configuração manual global.

Isso trata a causa observada para todas as instâncias que iniciem por esse serviço. Não constitui garantia contra futuras alterações de protocolo, indisponibilidade do WhatsApp ou todos os problemas possíveis de reconexão. A nc.2 também incorpora registros de runtime sincronizados, cancelamento por geração e correções pontuais do ciclo de conexão. Não foi adicionada uma rotina automática de logout, exclusão de sessões ou recriação de instâncias.

Na rotação de QR, a publicação de um código, o limite/timeout e a conclusão do pareamento passam pela mesma exclusão mútua e conferem a geração ativa. Quando `PairSuccess` é processado primeiro, cancela apenas a rotação e preserva a conexão recém-autenticada. A expiração encerra a tentativa sem chamar `Logout` nem revogar credenciais. Se o limite/timeout encerrar a geração antes do processamento de `PairSuccess`, o evento tardio é ignorado e a conexão precisa de nova tentativa. O polling de QR não reinicia um cliente já autenticado, e desconectar conserva as inscrições de eventos. Esses comportamentos passaram nos testes determinísticos com detector de corridas.

A trava ordena o encaminhamento dos eventos ao produtor. Webhooks e filas assíncronos ainda podem entregar mais tarde um evento encaminhado antes do pareamento; a correção não retira mensagens já encaminhadas desses produtores.

## Origem e revisão das propostas da comunidade

Base oficial: [`9337afc47e10b86cc896a6f432240e40fee95dd1`](https://github.com/evolution-foundation/evolution-go/commit/9337afc47e10b86cc896a6f432240e40fee95dd1). O manifesto com commits selecionados está em [RELEASE.json](RELEASE.json); os patches pontuais reaplicados estão em [upstream-patches](upstream-patches). Nenhum workflow de terceiros foi executado ou usado para publicar.

| Proposta | Decisão nesta versão |
| --- | --- |
| [Issue 225](https://github.com/evolution-foundation/evolution-go/issues/225), [PR 227](https://github.com/evolution-foundation/evolution-go/pull/227) | Aplicada a correção do handshake com `SetWAVersion`, com testes de payload de login e registro. |
| [PR 229](https://github.com/evolution-foundation/evolution-go/pull/229) | Aproveitado o limite de tempo. Sem fallback com revisão fixa, que também envelheceria. |
| [PR 230](https://github.com/evolution-foundation/evolution-go/pull/230) | Incorporadas validação, sincronização do setter e proteção contra downgrade automático. Sem piso de revisão estático. |
| [PR 220](https://github.com/evolution-foundation/evolution-go/pull/220) | Mantido o salto limitado de WhatsMeow até `a2381054887e`: reconhecimento de stanzas `<status>`, relacionado a desconexões periódicas. A nc.2 conserva essa dependência. A evidência de schema de autenticação 14 pertence ao teste de compatibilidade da nc.1. |
| [PR 221](https://github.com/evolution-foundation/evolution-go/pull/221) | Aplicado o compartilhamento do store e do pool PostgreSQL já aberto pelo processo. Inicialização com erro permite tentar novamente. |
| [PR 224](https://github.com/evolution-foundation/evolution-go/pull/224) | Aplicada somente a normalização do `+` em JIDs de usuários e seus testes; não foi mesclada a branch develop inteira. |
| [PR 206](https://github.com/evolution-foundation/evolution-go/pull/206) | Não aplicado o `sync.Once`, que memoriza uma falha inicial. A abordagem do PR 221 permite recuperação. Incluída proteção para handle PostgreSQL ausente. |
| [PR 233](https://github.com/evolution-foundation/evolution-go/pull/233), [PR 226](https://github.com/evolution-foundation/evolution-go/pull/226) | Atualização maior de dependência adiada: inclui mudança de API/schema 16 e amplia a migração sem ser necessária para o defeito reproduzido. |
| [PR 234](https://github.com/evolution-foundation/evolution-go/pull/234) | Aplicados registros de runtime sincronizados, sinais de parada idempotentes e coordenação das gerações de conexão. A integração recebeu proteção adicional da rotação de QR; testes integrados e detector de corridas passaram na nc.2. Retomada de conta real ainda pendente. |
| [PR 237](https://github.com/evolution-foundation/evolution-go/pull/237) | Adiado. A mudança sobreposta de ciclo de vida não foi mesclada automaticamente sobre o PR 234 e os ajustes locais. Exige revisão e homologação próprias. |
| [PR 149](https://github.com/evolution-foundation/evolution-go/pull/149) | Adaptada a proteção do polling de QR: sessão autenticada não é reiniciada; socket de pareamento conectado é reutilizado. Incluído teste da fronteira com ponteiro de cliente ausente. |
| [PR 187](https://github.com/evolution-foundation/evolution-go/pull/187) | Desconectar atualiza somente o estado da conexão e preserva as inscrições de eventos e configurações existentes. |
| [PR 195](https://github.com/evolution-foundation/evolution-go/pull/195) | Corrigida a conversão do evento `Archive` antes de montar o webhook, evitando a asserção de tipo que podia causar panic. |
| [PR 212](https://github.com/evolution-foundation/evolution-go/pull/212) | Aplicados os dois commits: dimensões de imagem e orientação EXIF, incluindo a orientação das miniaturas nos fluxos de envio afetados. |
| [PR 218](https://github.com/evolution-foundation/evolution-go/pull/218) | Pedido de histórico passa a usar `SendPeerMessage` para o próprio dispositivo primário, preservando o conteúdo do pedido. |
| [PR 199](https://github.com/evolution-foundation/evolution-go/pull/199) | Handshake e store compartilhado cobertos por soluções equivalentes. A nc.2 acrescenta coordenação entre QR, pareamento e timeout, com testes determinísticos e detector de corridas aprovados. Não foi aplicado o patch inteiro; pareamento real continua pendente. [Auditoria detalhada](PR-199.md). |
| [PR 208](https://github.com/evolution-foundation/evolution-go/pull/208) | Mudança de regra de numeração brasileira adiada para testes de destinatários; não necessária para gerar QR. |

Essas propostas foram examinadas como contribuições independentes, sem pressupor que estejam aprovadas ou incorporadas ao produto oficial. O reconhecimento de `<status>` também foi conferido no [commit do WhatsMeow](https://github.com/tulir/whatsmeow/commit/a2381054887e16d22428430fdc4d2b027a7f29a7). A ausência de novas quedas periódicas exige uma conta conectada e observação prolongada.

## Validação executada da nc.2

Os resultados abaixo foram obtidos no build Linux final da nc.2 e nas verificações isoladas da imagem e de suas regressões. Os testes de QR e PostgreSQL são opt-in, sem pareamento e com banco descartável. Nenhuma conta, sessão, licença ou mensagem de produção foi usada.

| Verificação da nc.2 | Situação |
| --- | --- |
| Build Linux, `go test ./... -count=1` e `go vet ./...` | Passaram. [Log do build final](evidence/nc2/build.log). |
| `go test -race` completo de seis pacotes: `runtimestate`, `whatsmeow/service`, `instance/service`, `chat/service`, `sendMessage/service` e `utils` | Passou durante o build final. [Log](evidence/nc2/build.log). |
| Intercalações de QR com `PairSuccess`, limite, timeout, substituição de geração e passkey; expiração sem `Logout` | Passaram com detector de corridas. [Regressões](evidence/nc2/recovery-regressions.log). |
| Polling autenticado sem reinicialização, cliente ausente sem panic e desconexão preservando inscrições | Passaram com detector de corridas. [Regressões](evidence/nc2/recovery-regressions.log). |
| Evento `Archive` e pedido de histórico via peer | Passaram com detector de corridas. [Regressões](evidence/nc2/recovery-regressions.log). |
| Handshake real com WhatsApp, sem pareamento | Aceito com a revisão `2.3000.1053100174`; devolveu 6 códigos QR em 1,24 s. Conteúdo omitido. [Log](evidence/nc2/live-qr.log). |
| Pool PostgreSQL real descartável | 64 inicializações concorrentes reutilizaram o pool limitado a 2 conexões. [Log](evidence/nc2/postgres-auth-pool.log). |
| Dimensões, orientação EXIF e miniaturas JPEG/PDF na imagem final | 13 testes passaram, incluindo renderização de PDF com `pdftoppm`. [Log](evidence/nc2/media-runtime.log). |
| Binário, dependências de mídia e conteúdo da imagem final | Ajuda do servidor, LDD, ffmpeg e pdftoppm passaram; LICENSE, NOTICE e TRADEMARKS presentes, sem `.env` empacotado. [Log](evidence/nc2/runtime-smoke.log). |
| Pareamento real, licença ativa, envio/recebimento, webhooks e retomada após reinício | Pendente; nenhuma conta ou dado de produção usado. |
| Publicação pública e digest remoto da nc.2 | Concluída. API do Docker Hub confirmou `is_private: false` e o digest da tag. [Registro](evidence/nc2/registry.json). |
| Pull anônimo da imagem publicada | Passou com configuração Docker sem helper ou credenciais; a imagem recuperada correspondeu ao ID local validado. [Log](evidence/nc2/anonymous-pull.log). |

## Evidência anterior da nc.1

Os resultados desta seção foram executados na `0.7.2-nc.1`, antes dos ajustes de runtime, QR e dos cinco PRs adicionais da nc.2. A dependência WhatsMeow continua em `a2381054887e`; os resultados anteriores de schema e contrato API são referências de compatibilidade da base e não substituem os testes da candidata atual.

| Verificação | Resultado |
| --- | --- |
| `go test ./... -count=1` e `go vet ./...` durante o build | Passaram. |
| `go test -race` para handshake, store e ParseJID | Passou. |
| Handshake real com WhatsApp, sem pareamento | Aceito; devolveu 6 códigos QR em 1,23 s. Conteúdo dos códigos omitido. [Log](evidence/live-qr.log). |
| Pool PostgreSQL real descartável | 64 inicializações concorrentes compartilharam pool limitado a 2 conexões; permaneceu aberto. [Log](evidence/postgres-auth-pool.log). |
| Sessão sintética, biblioteca antiga → candidata | Schema 14 e mesma chave pública; não autenticou no WhatsApp. [Log](evidence/auth-schema-compatibility.log). |
| Servidor oficial beta e candidato com bancos vazios isolados | Ambos iniciaram. Metadados consultados de 40 colunas do banco de usuários idênticos. [Comparação](evidence/users-schema-comparison.json). |
| Contrato Swagger público da instalação atual → base oficial | Mesmos 91 caminhos e definições. Não prova identidade entre os binários ou compatibilidade de dados de produção. |
| Miniaturas JPEG/PDF na imagem final | 6 testes passaram; PDF renderizado com `pdftoppm`. [Log](evidence/media-runtime.log). |
| Binário e dependências de mídia da imagem final | Inicialização de ajuda, LDD, ffmpeg e pdftoppm passaram. [Log](evidence/runtime-smoke.log). |
| Licença, conta pareada, envio/recebimento, webhooks, retomada após reinício | Pendentes. Os dois servidores descartáveis estavam sem licença ativa; nenhuma licença, sessão ou mensagem de produção foi usada. |

Os testes de QR e de banco são opt-in e não são acionados pelo build normal. Os testes normais usam dados sintéticos. As verificações de PostgreSQL recusam DSN fora do contêiner e banco descartáveis nomeados no teste. Os logs não contêm QR, chaves privadas ou dados de produção.

## Gerar, importar e publicar

Na raiz deste repositório:

```powershell
docker build --target verification -t nandocosta/evolution-go:0.7.2-nc.2-verify .
docker build --build-arg VERSION=0.7.2-nc.2 -t nandocosta/evolution-go:0.7.2-nc.2 .
New-Item -ItemType Directory -Force build
docker save --output build/nandocosta-evolution-go-0.7.2-nc.2.tar nandocosta/evolution-go:0.7.2-nc.2
Get-FileHash build/nandocosta-evolution-go-0.7.2-nc.2.tar -Algorithm SHA256
```

O tar produzido por `docker save` pode ser importado pelo Portainer ou com:

```sh
docker load --input nandocosta-evolution-go-0.7.2-nc.2.tar
```

O workflow `.github/workflows/publish_docker_image.yml` foi preparado para execução manual, imagem `nandocosta/evolution-go`, versão explícita e apenas `linux/amd64`, arquitetura validada na nc.2. Por padrão somente compila; publicar exige marcar `publish`, credenciais do namespace e confirmação automática de que o repositório de destino é público. Ele não publica em `evoapicloud` nem cria automaticamente `latest`. A nc.2 já foi publicada e verificada no registro; não é necessário executar uma nova publicação para puxar essa release.

O fluxo público e o pull pelo Portainer estão em [PUBLIC-REGISTRY.md](PUBLIC-REGISTRY.md). A [tag publicada](https://hub.docker.com/r/nandocosta/evolution-go/tags?name=0.7.2-nc.2) tem digest `sha256:038e39eb94db9abe08440c64cbf019f19e71198340deafdd683c4fc7df6f9909`. No serviço existente do Compose, usar exatamente:

```yaml
image: nandocosta/evolution-go:0.7.2-nc.2@sha256:038e39eb94db9abe08440c64cbf019f19e71198340deafdd683c4fc7df6f9909
```

O pull anônimo desse conteúdo passou com configuração Docker sem helper ou credenciais. A consulta do registro confirmou repositório público, plataforma `linux/amd64` e correspondência entre tag e digest. Os registros estão em [registry.json](evidence/nc2/registry.json) e [anonymous-pull.log](evidence/nc2/anonymous-pull.log).

Antes de subir uma nova release, atualizar VERSION, RELEASE.json e as referências de versão do Dockerfile. Não substituir uma versão já publicada. Atualizações oficiais devem entrar por revisão do código e nova homologação; não utilizar atualização automática para trocar silenciosamente esta imagem por uma oficial.

A descoberta automática da revisão Web do WhatsApp continua ativa e agora atualiza o handshake real. Isso é separado da atualização da imagem Docker: esta deve permanecer na tag escolhida. Não foi encontrado mecanismo de auto-substituição do binário no código público usado como base; automações externas do servidor precisam respeitar a referência `nandocosta`.

ID da imagem final local nc.2 e da imagem recuperada do registro: `sha256:dcd75acfe639294bab9870de079fc67ee548746dae0611a54066d4aef5a4f11a`. O digest remoto é `sha256:038e39eb94db9abe08440c64cbf019f19e71198340deafdd683c4fc7df6f9909`. O pacote `build/nandocosta-evolution-go-0.7.2-nc.2.tar` tem 221.972.480 bytes e SHA256 `25226f3b471b02c3396422e20903db939c19d693d13d1be4560d2f1094914019`, acompanhado de `.tar.sha256`. O ID local, o digest de registro e o hash do tar são identificadores diferentes. O [manifesto do artefato](evidence/nc2/artifact.json) também registra o fingerprint das fontes compiladas; o [manifesto de fontes](evidence/nc2/source-manifest.json) permite conferir os 121 arquivos correspondentes.

O pacote anterior, da nc.1, está em `build/nandocosta-evolution-go-0.7.2-nc.1.tar` (221.850.112 bytes). SHA256 do arquivo: `f3729b9e4a701bfbb2e8d0aaafbbef08cb7b14006bd2ab2475ed7b05a1fc16c5`. ID daquela imagem local: `sha256:90d7db0e6d2a8b74128d8662551f4345029476a660768d86d7cbd58d0ac31b96`. Esses identificadores não pertencem à nc.2.

## Troca controlada no Compose e retorno

1. O responsável optou por seguir sem parear o WhatsApp pessoal. Na troca, validar conexão, envio/recebimento de texto e mídia, webhooks e retomada após reinício. Testar também uma instância nova e reconexão solicitada pelo cliente. Acompanhar estabilidade por mais de um ciclo da falha periódica reportada. Essa confirmação ainda não foi executada.
2. No ambiente de destino, identificar a chave real do serviço e a imagem/digest atual. A instalação observada utiliza `evoapicloud/evolution-go:0.7.2-beta`, digest `sha256:6f62232e5bf36ca97ea3380c0fef627682444e8cffefbcdd7a4ae20047d90245`. Registrar a configuração atual e guardar esse digest para retorno.
3. Antes de uma atualização que execute alterações em bancos existentes, fazer e validar backup dos bancos de usuários e autenticação do destino. Preservar volumes, sessões e arquivos existentes; não usar dados de produção na homologação. O teste sintético de compatibilidade não substitui essa proteção.
4. Puxar a imagem pública validada no Docker do destino pelo [procedimento do Portainer](PUBLIC-REGISTRY.md), ou importar o tar validado, e trocar somente `image:` no serviço correspondente pela referência exata com tag e digest acima. [compose.override.yaml](compose.override.yaml) é um exemplo a mesclar, não um Compose completo. Conservar ambiente, portas, volumes e redes. A instalação observada já usa `GLOBAL_API_KEY`, `POSTGRES_USERS_DB` e `POSTGRES_AUTH_DB`; não há renomeação de variáveis a fazer por este diagnóstico.
5. Conferir a configuração mesclada em ambiente seguro, pois sua saída pode conter segredos. Recriar somente o serviço de aplicação sem remover os volumes e sem recriar PostgreSQL. Confirmar o handshake aplicado, status das instâncias e geração de QR no fluxo normal do nosso sistema.
6. Se necessário, voltar a referência de imagem para o digest registrado e recriar somente a aplicação. Se houver alteração persistente de schema/dados incompatível, o retorno também exige o backup validado correspondente. Não excluir sessões nem executar logout em lote.

Os nomes e comandos de deploy devem ser ajustados ao Compose real. O exemplo não foi aplicado no Plesk/Portainer de produção. Nenhuma correção PHP foi necessária para a causa reproduzida.
