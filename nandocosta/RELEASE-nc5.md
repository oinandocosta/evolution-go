# Evolution GO — nandocosta 0.7.2-nc.5

Publicada em 04/11/2026 no Docker Hub, baseada na `0.7.2-nc.4`, com WhatsMeow atualizado para `v0.0.0-20261102152715-54038813c47f`, a versão mais recente confirmada em 04/11/2026 no Go Proxy e no repositório do mantenedor. O compilador permanece Go 1.26.10, com a exigência mínima do módulo ajustada para Go 1.26.0. Todas as correções anteriores continuam incluídas.

Tag: `nandocosta/evolution-go:0.7.2-nc.5`

Digest: `sha256:8ae70d86964d4a442f9fa8c4551ea3a45e2ca85707242905ef63949b3793c715`

Pull anônimo confirmado para o mesmo image ID testado. [Artefato e resultados](evidence/nc5/artifact.json).

## Atualização e compatibilidade

A nova dependência traz 92 commits desde `a2381054887e`, incluindo correções de LID, processamento de recibos e retry, fila de eventos entre conexões, frames fragmentados, mídia, handshake, redação de credenciais em logs e backoff de app state. A reconexão continua sendo gerenciada pelo runtime existente da nossa distribuição; não há ativação de um segundo gerenciador de reconexões.

Adaptações locais:

- Bloquear, desbloquear e listar bloqueados usam as novas assinaturas da biblioteca. A API preserva `DHash` e `JIDs` na resposta, filtrando entradas inativas e preferindo o telefone quando disponível, com fallback para LID. O hash de consulta fica vazio para solicitar a lista completa. Entradas inválidas e destinatário inexistente recebem 400.
- A alteração do texto “Sobre” usa `SetStatusInput`, mantendo texto puro sem emoji e com a duração zero padrão do SDK. O payload público da nossa API continua igual.
- O evento `RotateADVSecret` renova o campo de segredo nos QR atual e restantes, utilizando o mesmo worker e a mesma geração. A renovação não consome um novo código do limite configurado, não faz logout e não imprime QR ou segredos em logs. Pareamento e substituição de geração continuam protegidos pelo mutex existente.
- Avatar, mensagens editadas, mídia, WebSocket, eventos, configurações parciais e runtime sincronizado da nc.4 permanecem. A correção PHP de fotos e a expansão “Mensagem original” continuam no commit `8565536` da API, sem nova mudança PHP nesta release.

Fontes: [comparação upstream](https://github.com/tulir/whatsmeow/compare/a2381054887e16d22428430fdc4d2b027a7f29a7...54038813c47f3f351eb6fcaafab0e1b99d676850), [versão consultada](evidence/nc5/whatsmeow-latest.json) e [manifesto da distribuição](RELEASE.json). Histórico: [nc.4](RELEASE-nc4.md), [nc.3](RELEASE-nc3.md) e [nc.2](RELEASE-nc2.md).

## Migração de autenticação

O schema do WhatsMeow passa de 14 para 16. As duas migrations oficiais adicionam somente:

- `whatsmeow_device.companion_meta_nonce`, texto não nulo com padrão vazio;
- `whatsmeow_chat_settings.wasa_root_secret_id`, texto não nulo com padrão vazio.

As migrations se declaram compatíveis com v8+ e são executadas pelo mecanismo oficial da biblioteca ao abrir o armazenamento. Não há DROP, renomeação, reset, recriação de instância ou novo pareamento exigido pela atualização. As configurações e os modelos da aplicação não são modificados. Variáveis, bancos, volumes, portas, redes, licença e identidade de origem permanecem.

O probe [auth-migration-probe](auth-migration-probe/main.go) verificou a evolução com múltiplas sessões e configurações sintéticas em PostgreSQL descartável. Compara dados e chaves por hashes, lê e grava através das APIs das duas versões e confere os novos campos. Oito etapas passaram com três sessões, 19 tabelas e 64 linhas preservadas. O retorno à nc.4 com leitura e escrita foi executado sem apagar nonce/WASA; a nova versão releu os mesmos hashes. [Comandos, resultados e limites](evidence/nc5/auth-migration-readme.md).

O retorno à nc.4 deve usar o schema já evoluído; não se executa downgrade SQL ou alteração manual da versão.

## Validação e limites

O build exige `go test ./...`, `go vet ./...` e detector de corridas em doze pacotes. A homologação inclui regressões de QR, cancelamento, geração atual, avatar, bloqueio/status, mensagens editadas, mídia e preservação de configurações. Testes de mídia executam no runtime final. O teste de pool usa 64 inicializações concorrentes e PostgreSQL exclusivamente descartável.

Passaram testes Go, vet, race em 12 pacotes, integridade dos módulos, 43 testes de envio/mídia no runtime final e 64 inicializações concorrentes do pool PostgreSQL. O handshake live retornou seis QR em 1,25 s.

Evidências executadas ficam em [evidence/nc5](evidence/nc5). O teste live de QR usa dispositivo efêmero, sem número, conta pessoal, pareamento ou mensagens. As sessões e chaves de migração são sintéticas; elas comprovam persistência e compatibilidade do armazenamento, não autenticação real no WhatsApp. Envio/recebimento, obtenção de foto e continuidade de uma conta pareada precisam de conferência após a troca no servidor. A permanência remota do texto “Sobre” com a duração zero da nova API também não foi homologada com conta pareada.

## Compilar e publicar

```powershell
docker build --target verification -t nandocosta/evolution-go:0.7.2-nc.5-verify .
docker build --build-arg VERSION=0.7.2-nc.5 -t nandocosta/evolution-go:0.7.2-nc.5 .
python -B -m unittest discover -s nandocosta -p test_public_repository.py -v
python -B nandocosta/check-public-repository.py
docker push nandocosta/evolution-go:0.7.2-nc.5
```

A publicação depende da aprovação dos checks e da migração sintética. Confirmar que a tag nova ainda não existe; publicar somente o artefato validado. Não sobrescrever nc.4, outras releases ou `latest`. O repositório Docker Hub permanece público e o pull é verificado sem login.

## Portainer

Após publicação, mesclar somente a referência `image:` de [compose.override.yaml](compose.override.yaml) no serviço existente, preservando ambiente, volumes, bancos, redes, licença e demais configurações. A atualização usa o mesmo armazenamento das conexões. Não excluir volumes, fazer logout ou recriar instâncias. Guardar a referência da nc.4 para retorno.

Esta preparação não realiza a troca no servidor nem opera os dados de produção. LICENSE, NOTICE, TRADEMARKS e ativação da Evolution GO são preservados. A descoberta da revisão Web do WhatsApp continua automática; a versão do código da biblioteca permanece fixada no manifesto para tornar a imagem reproduzível. [Registro público](PUBLIC-REGISTRY.md).
