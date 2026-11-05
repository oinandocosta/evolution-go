# Evolution GO — nandocosta 0.7.2-nc.4

Publicada no [Docker Hub](https://hub.docker.com/r/nandocosta/evolution-go/tags?name=0.7.2-nc.4) em 04/11/2026, com pull anônimo verificado. Baseada na `0.7.2-nc.3`, com correções de consulta da foto de perfil adaptadas do [PR 120](https://github.com/evolution-foundation/evolution-go/pull/120). Mantém todas as correções anteriores, o WhatsMeow `a2381054887e` e o schema de autenticação 14. Não troca licença, identidade de origem, variáveis, rotas, bancos ou armazenamento de sessões.

## Consulta de foto de perfil

- Uma única janela de oito segundos cobre preparação da sessão, espera de autenticação, resolução LID/PN e consulta ao WhatsApp. O cancelamento HTTP é propagado; uma sessão iniciada continua pertencendo ao serviço.
- Identificadores são normalizados sem dispositivo (`ToNonAD`); o LID usa o telefone mapeado quando conhecido na própria instância. Não procura sessões de outros clientes.
- Clientes e gerações ativos ou em inicialização são reutilizados. Uma consulta de foto não provoca logout, exclusão de credenciais ou rotação de uma conexão ativa.
- Entrada nula/inválida retorna 400, sessão indisponível retorna 401, limite do WhatsApp retorna 429 e timeout/cancelamento retorna 504. Falhas internas não expõem detalhes de banco, proxy ou sessão. Foto privada/inexistente continua retornando 500, conforme o contrato do PR.
- Sucesso preserva exatamente `message` e `data` com `url`, `id`, `type`, `direct_path` e `hash`. A GO retorna metadados da foto; não baixa a imagem nem converte para base64.

A integração é seletiva: utiliza o runtime sincronizado da nc.3, sem adicionar o índice paralelo de clientes proposto pelo PR. O [PR 121](https://github.com/evolution-foundation/evolution-go/pull/121), que adiciona `PictureURL` ao `/user/info`, não faz parte desta release. Commit e origem estão em [RELEASE.json](RELEASE.json) e [upstream-patches/120.patch](upstream-patches/120.patch).

## Correção do consumidor Laravel

O consumidor `EvolutionGo::obterAvatar` da API Agente passa a ler `data.url` e baixar a imagem antes de entregá-la ao armazenamento já existente. Preserva os retornos base64 legados, a validação de imagem e a foto anterior em falhas. O download usa HTTPS para os hosts de foto do WhatsApp, sem credenciais da GO, sem redirecionamentos automáticos, com limite de tempo e de 2 MiB. Essa alteração foi enviada para `origin/master` da API no commit `8565536`, junto com a melhoria de “Mensagem original”.

A melhoria visual de “Mensagem original” também pertence à API Agente. Ela não muda o protocolo das mensagens editadas da nc.3.

## Compatibilidade do WhatsMeow

A dependência permanece fixada para preservar o contrato e o armazenamento usados na nc.3. A comparação com a versão mais recente está em [WHATS-MEOW-nc4.md](WHATS-MEOW-nc4.md). A atualização da revisão Web anunciada ao WhatsApp continua automática; isso não atualiza o código da biblioteca Go.

## Validação

Resultados executados e identificação do artefato ficam em [evidence/nc4](evidence/nc4). O build exige `go test ./...`, `go vet ./...` e race em doze pacotes, incluindo serviço e handler de usuário. As regressões cobrem JSON nulo, escopo por instância, formato real da resposta, prazo compartilhado, SQL cancelado, identificadores PN/LID, geração atual e preservação das conexões. Os testes PHP usam SQLite exclusivamente em memória, HTTP e storage simulados.

A compatibilidade de sessões é verificada com sessão sintética e PostgreSQL descartável entre nc.3 e nc.4, sem conta pareada. QR live não usa conta pessoal nem envia mensagens. Obter foto real, envio e recebimento com conta pareada continuam dependendo da conferência após a troca. Não se contorna privacidade do WhatsApp.

## Compilar e publicar

```powershell
docker build --target verification -t nandocosta/evolution-go:0.7.2-nc.4-verify .
docker build --build-arg VERSION=0.7.2-nc.4 -t nandocosta/evolution-go:0.7.2-nc.4 .
python -B -m unittest discover -s nandocosta -p test_public_repository.py -v
python -B nandocosta/check-public-repository.py
docker push nandocosta/evolution-go:0.7.2-nc.4
```

Publicar somente depois da validação e de confirmar que a tag nova ainda não existe. Não sobrescrever `latest`, nc.3 ou releases anteriores. Manter a nc.3 disponível para retorno.

## Portainer e preservação

O [compose.override.yaml](compose.override.yaml) fornece somente `image:` após a publicação. Mesclar essa referência no serviço existente e preservar ambiente, portas, redes, volumes, PostgreSQL, licença e demais configurações. Não remover volumes, encerrar sessões, fazer logout ou recriar instâncias.

A publicação no Docker Hub não executa a troca no servidor. Conferir status, texto/mídia, webhooks, mensagens editadas, fotos e retomada após reiniciar somente a aplicação. Nenhum dado ou estrutura de produção é alterado nesta preparação.

As correções de configurações, WebSocket, mídia, QR e mensagens editadas da nc.3 permanecem. [Histórico nc.3](RELEASE-nc3.md), [histórico nc.2](RELEASE-nc2.md) e [acesso ao registro público](PUBLIC-REGISTRY.md). LICENSE, NOTICE, TRADEMARKS e ativação da Evolution GO são preservados.

Referência publicada e verificada:

```yaml
image: nandocosta/evolution-go:0.7.2-nc.4@sha256:119152284b813d6273c5fc83db880e321471f1dd8ae22dec0a0e8bf9bb00fc40
```

Resultados: 31 testes PHP/176 asserções; testes completos Go, vet e race em 12 pacotes; 43 testes de envio/mídia no runtime final; seis QR em 1,12 s sem pareamento; 64 inicializações usando o pool limitado. A sessão sintética nc.3 foi lida pela nc.4 com schema14 e a mesma chave. A UI foi validada no Edge em desktop/mobile com dados sintéticos.

O tar `build/nandocosta-evolution-go-0.7.2-nc.4.tar` tem 222345728 bytes e SHA256 `5f1d0a85819a76a68179eaa869bf167be88822fc16d5e0f79befbdc1d90830e2`. A nc.3 permanece publicada. [Artefato e limites](evidence/nc4/artifact.json).
