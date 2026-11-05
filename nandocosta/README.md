# Evolution GO — nandocosta 0.7.2-nc.6

Versão publicada baseada na nc.5 para corrigir o tratamento de consultas de foto. WhatsMeow, Go, dependências, autenticação, configurações e modelo de conexão permanecem os mesmos da nc.5. Esta release não acrescenta migrações ao schema 16.

## Fotos de contato

O sucesso mantém os metadados em `data.url`; download e armazenamento continuam no consumidor PHP. Anteriormente, ausência de foto e consulta não autorizada pelo WhatsApp eram convertidas em HTTP 500 indistinguível de falha interna. Agora:

| Situação | HTTP | code |
| --- | --- | --- |
| Foto não disponível, resposta nula ou URL vazia | 404 | PROFILE_PICTURE_NOT_SET |
| WhatsApp não autorizou a consulta da foto | 403 | PROFILE_PICTURE_UNAUTHORIZED |
| Falha interna ou inesperada na consulta | 500 | AVATAR_QUERY_FAILED |

Os demais status de validação, sessão, limite e timeout são preservados, assim como o campo público `error`. O consumidor PHP reconhece somente os pares 403/404 e códigos correspondentes como foto indisponível; não mascara 500 genérico, erro de autenticação ou resposta inesperada. Mantém foto existente, downloads `data.url`, formato legado e armazenamento privado.

O diagnóstico registra categoria, estágio, código IQ numérico, duração, preview e servidor do alvo efetivo, sem telefone, JID completo, URL assinada, token, XML ou erro bruto. Uma recusa do WhatsApp não comprova que a foto é privada: tokens ou políticas do provedor também podem impedir acesso. Não há tentativa de contornar restrições de privacidade nem mudança especulativa de JID/preview.

## Verificação

O build exige testes Go, vet e race em 12 pacotes. Os contratos 403/404/500, retorno nulo/URL vazia, preservação de geração e sanitização são testados. O consumidor PHP passou 90 testes e 636 assertions isolados, com HTTP/storage simulados e SQLite em memória, incluindo consulta, download e salvamento. A validação no servidor deve correlacionar uma tentativa nova do job com o código retornado e a categoria registrada pela Evolution GO.

Histórico: [nc.5](RELEASE-nc5.md), [nc.4](RELEASE-nc4.md), [nc.3](RELEASE-nc3.md). A nc.5 já validou a migração 14→16 com 3 sessões sintéticas, 19 tabelas e retorno ao código nc.4. Configurações e credenciais das instâncias não são alteradas por este patch.

## Atualização

Publicar cada release em uma nova tag fixa, preservando as versões anteriores. Após os checks e a validação do artefato, promover `latest` para o mesmo digest da versão publicada. A promoção atual foi aprovada: `nandocosta/evolution-go:latest` aponta para a mesma imagem da `0.7.2-nc.6`, com pull anônimo confirmado e sem reconstrução.

Atualizar `latest` no registro não modifica um contêiner já em execução: é necessário fazer pull e redeploy para aplicar uma nova versão. A stack atual usa `image: nandocosta/evolution-go:latest`, sem digest na referência. Preservar ambiente, volumes, banco, rede e licença ao atualizar, e conferir as sessões e o tráfego após a troca.

A stack em produção usa `CONNECT_ON_STARTUP=true`, conforme opção aprovada. Na inicialização, a aplicação tenta retomar as instâncias marcadas como conectadas no banco, respeitando o filtro `CLIENT_NAME` quando definido. Essa retomada reutiliza as credenciais existentes, sem logout ou novo pareamento; instâncias marcadas como desconectadas não são iniciadas por essa flag.

LICENSE, NOTICE, TRADEMARKS e ativação são preservados. Evidências desta release: `evidence/nc6`.

## Validação em produção

Publicada e aplicada por digest fixo no Portainer. O Compose foi comparado integralmente: somente a imagem mudou. O contêiner PostgreSQL permaneceu o mesmo. As três sessões antes online retomaram e responderam Connected=true e LoggedIn=true, sem logout ou QR.

Consulta da própria foto: HTTP200, data.url e data.id presentes. O contato investigado: HTTP403 em preview, tamanho completo e LID autoritativo; categoria PROFILE_PICTURE_UNAUTHORIZED, IQ401. A sessão foi localizada e não tinha token de privacidade local para esse alvo. Isso restringe o diagnóstico, sem provar privacidade nem garantir renovação do token. A tentativa do job PHP concluiu após o novo contrato. Nenhuma mensagem de teste foi enviada. Envio/recebimento por tráfego passivo permanece dependente de tráfego observado.

O consumidor PHP publicado em c39a2e9 conserva data.id, respeita o intervalo também sem foto, permite recuperar arquivo perdido e aproveita PictureID recebido sem realizar /user/info desnecessário. Jobs antigos permanecem compatíveis. Em produção, repetir o job do contato dentro do intervalo concluiu sem nova consulta /user/avatar. Assinaturas PICTURE existentes e variáveis de ambiente não foram modificadas.

Em atualização operacional posterior, a stack passou a usar `latest` sem digest e `CONNECT_ON_STARTUP=true`. A comparação do Compose confirmou somente essas duas alterações aprovadas. A imagem efetiva continuou sendo a nc.6, o PostgreSQL permaneceu no mesmo contêiner e as sessões anteriormente online retomaram automaticamente; as demais permaneceram offline. O histórico acima registra a aplicação inicial por digest fixo.
