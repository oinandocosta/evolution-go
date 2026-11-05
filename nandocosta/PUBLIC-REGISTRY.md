# Imagem pÃºblica no Docker Hub e pull pelo Portainer

Esta distribuição está no repositório público [nandocosta/evolution-go](https://hub.docker.com/r/nandocosta/evolution-go). A versão `0.7.2-nc.5` foi publicada e validada por pull anônimo em 04/11/2026. Digest: `sha256:8ae70d86964d4a442f9fa8c4551ea3a45e2ca85707242905ef63949b3793c715`. [Evidências da release](evidence/nc5/artifact.json). A troca da imagem no servidor não foi executada. A nc.4 permanece disponível para retorno ao código anterior, sem downgrade SQL, conforme a migração sintética testada.

A tag de cada release deve corresponder a VERSION e RELEASE.json; nÃ£o usar uma tag mutÃ¡vel para substituir silenciosamente uma versÃ£o instalada. A preparaÃ§Ã£o do workflow nÃ£o publica a imagem por si sÃ³. As instruÃ§Ãµes seguintes descrevem futuras publicaÃ§Ãµes; o workflow GitHub ainda nÃ£o foi executado.

## Publicar

1. Entrar na conta que controla o namespace `nandocosta` no Docker Hub e criar previamente o repositÃ³rio `evolution-go` com visibilidade **Public**. Confirmar o namespace e a visibilidade. O workflow nÃ£o cria repositÃ³rios nem muda a visibilidade de um repositÃ³rio existente.
2. Para a publicaÃ§Ã£o, usar um PAT com **Repo Read & Write**, guardado em `DOCKER_PASSWORD`; `DOCKER_USERNAME` deve identificar a conta correspondente. No GitHub Actions, cadastrar ambos como secrets do repositÃ³rio que hospedarÃ¡ este cÃ³digo. Nunca colocar tokens no Compose, cÃ³digo, arquivos de evidÃªncia ou chat. Esses dados sÃ£o utilizados somente no login para publicar; a consulta de visibilidade Ã© anÃ´nima.
3. Executar manualmente o workflow `Build nandocosta Docker Image`, marcando `publish` somente para publicar. O padrÃ£o permanece desmarcado e apenas compila. O workflow valida VERSION/RELEASE.json, executa testes offline da trava, consulta anonimamente o Docker Hub para confirmar que o repositÃ³rio jÃ¡ existe e Ã© pÃºblico, entÃ£o autentica e publica a tag fixa. NÃ£o cria `latest` nem publica no namespace oficial.
4. Se o repositÃ³rio nÃ£o existir, for privado, omitir o campo de visibilidade ou a consulta falhar, a publicaÃ§Ã£o Ã© interrompida. NÃ£o existe fallback para criar um repositÃ³rio automaticamente ou mudar sua visibilidade. ApÃ³s o push, registrar o digest remoto da imagem para o deploy.

A trava usa somente `GET https://hub.docker.com/v2/namespaces/nandocosta/repositories/evolution-go`, sem token, cookie ou requisiÃ§Ã£o de autenticaÃ§Ã£o. Aceita exclusivamente HTTP 200 com `is_private` booleano igual a `false`; rejeita redirecionamentos e limita a requisiÃ§Ã£o a 15 segundos. NÃ£o imprime corpos de respostas. Os testes desta preparaÃ§Ã£o sÃ£o offline; nÃ£o houve push nem login pelo script.

A consulta e o push nÃ£o sÃ£o atÃ´micos. Limitar quem pode alterar a visibilidade do repositÃ³rio evita mudanÃ§as durante a publicaÃ§Ã£o. A trava nÃ£o altera permissÃµes ou visibilidade de repositÃ³rios.

## Puxar no Portainer

1. No ambiente Docker de destino no Portainer, abrir **Images â†’ Pull image** e selecionar o registro Docker Hub pÃºblico/anÃ´nimo disponÃ­vel. NÃ£o Ã© necessÃ¡rio cadastrar token para acessar esta imagem pÃºblica.
2. Informar `nandocosta/evolution-go` com a tag exata da release registrada em VERSION e RELEASE.json. Confirmar que o pull terminou com sucesso antes da troca.
3. Na stack existente, alterar somente `image:` no serviÃ§o da aplicaÃ§Ã£o. Manter ambiente, portas, redes, volumes e banco existentes. NÃ£o remover volumes nem recriar o serviÃ§o PostgreSQL por causa dessa alteraÃ§Ã£o.
4. Preservar a referÃªncia da imagem anterior para retorno e seguir o procedimento de backup e validaÃ§Ã£o descrito no README desta distribuiÃ§Ã£o. A conta pareada, licenÃ§a, envio/recebimento e retomada apÃ³s reinÃ­cio ainda devem ser confirmados na troca.

O Docker Hub tambÃ©m permite autenticaÃ§Ã£o opcional para pulls pÃºblicos, por exemplo quando os limites anÃ´nimos forem insuficientes. Se isso for necessÃ¡rio, o token Ã© cadastrado no Portainer, nunca no YAML. A imagem pÃºblica nÃ£o exige dar permissÃ£o de escrita ao Portainer.

## Validar a trava sem rede ou credenciais

Na raiz do fork:

```sh
python -B -m unittest discover -s nandocosta -p test_public_repository.py -v
```

## ReferÃªncias oficiais

- [Visibilidade e acesso no Docker Hub](https://docs.docker.com/docker-hub/repos/manage/access/).
- [Consulta do repositÃ³rio e campo is_private](https://docs.docker.com/reference/api/hub/latest/operations/GetRepository/).
- [PermissÃµes dos access tokens](https://docs.docker.com/security/access-tokens/reference/).
- [Docker Hub pÃºblico e autenticado no Portainer](https://docs.portainer.io/admin/registries/add/dockerhub).
