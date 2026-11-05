# Evolution GO â€” nandocosta 0.7.2-nc.3

Publicada no [Docker Hub](https://hub.docker.com/r/nandocosta/evolution-go/tags?name=0.7.2-nc.3) em 04/11/2026 e validada por pull anônimo. Baseada na `0.7.2-nc.2`, com WhatsMeow mantido em `a2381054887e` (schema de autenticaÃ§Ã£o 14). MantÃ©m configuraÃ§Ã£o, sessÃµes, variÃ¡veis de ambiente, rotas, licenÃ§a e identidade de origem. Esta preparaÃ§Ã£o nÃ£o altera o ambiente de produÃ§Ã£o. O responsÃ¡vel confirmou que a nc.2 restaurou suas conexÃµes; os testes desta nova versÃ£o usam somente dados sintÃ©ticos e recursos descartÃ¡veis.

## CorreÃ§Ãµes da nc.3

| Origem | IntegraÃ§Ã£o |
| --- | --- |
| [PR 136](https://github.com/evolution-foundation/evolution-go/pull/136) | Reconectar com campos omitidos conserva eventos e produtores. ConfiguraÃ§Ãµes avanÃ§adas sÃ£o parciais, incluindo `false` explÃ­cito. A autorizaÃ§Ã£o exige correspondÃªncia entre token e instÃ¢ncia no caminho. |
| [PR 135](https://github.com/evolution-foundation/evolution-go/pull/135) | Um writer por socket, prazo de escrita de 10 s, limpeza por identidade e registro livre durante I/O. A falha do WebSocket nÃ£o impede a tentativa do webhook. Mantida uma conexÃ£o por instÃ¢ncia e broadcasts. |
| [PR 137](https://github.com/evolution-foundation/evolution-go/pull/137) | MenÃ§Ãµes localizam o conteÃºdo em wrappers de documento com legenda, mÃ­dia e mensagens interativas, conservando o contexto existente. |
| [PR 207](https://github.com/evolution-foundation/evolution-go/pull/207) | Link explÃ­cito e metadados fornecidos tÃªm prioridade; leitura HTTP com limites de tempo/bytes/pixels; miniaturas grandes enviadas ao WhatsApp, com fallback para inline/texto. Newsletters continuam com miniatura inline. |
| [PR 237, partes selecionadas](https://github.com/evolution-foundation/evolution-go/pull/237) | Envio exige socket autenticado e cliente da geraÃ§Ã£o atual, inclusive depois do atraso. Logs selecionados omitem tokens, links de pareamento, URLs de mÃ­dia e corpos de resposta dos webhooks. Sem trocar a arquitetura de ciclo de vida nem atualizar dependÃªncias/schema. |
| [PR 122](https://github.com/evolution-foundation/evolution-go/pull/122) | Descriptografia de ediÃ§Ãµes antes de normalizar identidades, nonce validado, prazo de 5 s, mensagem original identificada por `protocolMessage.key.ID` e marcadores de ediÃ§Ã£o/falha. CÃ³pia local do protobuf protege outros handlers. |

Os commits exatos estÃ£o em [RELEASE.json](RELEASE.json); patches de origem em [upstream-patches](upstream-patches). SÃ£o integraÃ§Ãµes selecionadas de propostas da comunidade, sem afirmar que jÃ¡ foram mescladas oficialmente. Os ajustes anteriores de QR, PostgreSQL, runtime, orientaÃ§Ã£o/miniaturas e histÃ³rico permanecem; veja [evidÃªncias e notas da nc.2](RELEASE-nc2.md).

## Mensagens editadas

O receptor existente em `api-agente` jÃ¡ trata `IsEdit` atualizando a mensagem original em transaÃ§Ã£o, por `protocolMessage.key.ID`, e extrai o novo texto de `protocolMessage.editedMessage`. O novo produtor fornece esse contrato quando consegue descriptografar. Se nÃ£o houver o segredo original ou o conteÃºdo for invÃ¡lido, sinaliza `decryptFailed` e conserva o envelope; nÃ£o inventa texto. Os dois testes existentes do consumidor passaram com SQLite exclusivamente em memÃ³ria, HTTP/fila simulados (24 asserÃ§Ãµes); nenhum arquivo PHP foi alterado.

A classificaÃ§Ã£o de revogaÃ§Ã£o do PR 122 tambÃ©m foi preservada. O consumidor PHP ainda nÃ£o possui tratamento especÃ­fico de `IsRevoke`; esta release nÃ£o implementa exclusÃ£o de mensagens no CRM.

## Testes e limites

Resultados da candidata e identificaÃ§Ã£o do artefato serÃ£o registrados em [evidence/nc3](evidence/nc3). Build inclui `go test ./...`, `go vet ./...` e detector de corridas nos pacotes de runtime, eventos, instÃ¢ncias, WhatsMeow, chat, envio e utilitÃ¡rios. A homologaÃ§Ã£o de QR nÃ£o pareia conta. Os testes PostgreSQL usam contÃªiner e banco descartÃ¡veis sem dados de produÃ§Ã£o. Testes de mÃ­dia executam no runtime final.

Envio/recebimento e ediÃ§Ã£o em uma conta realmente pareada precisam de confirmaÃ§Ã£o apÃ³s a troca; a validaÃ§Ã£o sintÃ©tica nÃ£o comprova a entrega real ao WhatsApp. NÃ£o foi usado WhatsApp pessoal na homologaÃ§Ã£o.

## Compilar e publicar

```powershell
docker build --target verification -t nandocosta/evolution-go:0.7.2-nc.3-verify .
docker build --build-arg VERSION=0.7.2-nc.3 -t nandocosta/evolution-go:0.7.2-nc.3 .
python -B -m unittest discover -s nandocosta -p test_public_repository.py -v
python -B nandocosta/check-public-repository.py
docker push nandocosta/evolution-go:0.7.2-nc.3
```

Publicar somente o artefato validado e a tag nova; preservar a nc.2 como retorno. O workflow existente Ã© manual. NÃ£o utiliza `latest` nem substitui uma release publicada.

## Portainer e preservaÃ§Ã£o das conexÃµes

ApÃ³s publicaÃ§Ã£o e verificaÃ§Ã£o anÃ´nima, [compose.override.yaml](compose.override.yaml) contÃ©m apenas a referÃªncia da imagem validada. Mesclar essa linha no serviÃ§o existente, mantendo ambiente, portas, redes, volumes, bancos e licenÃ§a. NÃ£o Ã© um Compose completo. Nenhum logout, exclusÃ£o de sessÃ£o ou recriaÃ§Ã£o de instÃ¢ncia faz parte da atualizaÃ§Ã£o.

A troca no servidor nÃ£o Ã© executada por esta preparaÃ§Ã£o. Guardar a referÃªncia atual para retorno e validar conexÃ£o, texto/mÃ­dia, webhooks, ediÃ§Ã£o e retomada apÃ³s reiniciar somente a aplicaÃ§Ã£o. Preservar os bancos existentes; qualquer operaÃ§Ã£o que altere/remova dados ou estruturas existentes exige backup validado do alvo. InstruÃ§Ãµes de acesso pÃºblico: [PUBLIC-REGISTRY.md](PUBLIC-REGISTRY.md).

A descoberta da revisÃ£o Web do WhatsApp continua automÃ¡tica. A imagem Docker permanece na tag escolhida. NÃ£o remover ou desabilitar LICENSE, NOTICE, TRADEMARKS ou verificaÃ§Ã£o de ativaÃ§Ã£o da Evolution GO.

Referência publicada e verificada:

```yaml
image: nandocosta/evolution-go:0.7.2-nc.3@sha256:5bc9b1d6dd55d018b07623495a1e3c9bad938bd649f5559caf676d726dcea2e2
```

O tar `build/nandocosta-evolution-go-0.7.2-nc.3.tar` tem 222302208 bytes e SHA256 `430fa4a7396e968319ab9a1e5de28e143e72b7ec12a0b11ab725418aff77447d`. A nc.2 permanece publicada.
