# Revisão da dependência WhatsMeow — nc.4

Consulta em 04/11/2026 ao Go Proxy oficial e ao repositório do mantenedor:

| Item | Nossa nc.3 / nc.4 | Mais recente consultada |
| --- | --- | --- |
| Versão | `v0.0.0-20260720135917-a2381054887e` | `v0.0.0-20261102152715-54038813c47f` |
| Data UTC | 20/07/2026 | 02/11/2026 15:27:15 |
| Schema de autenticação | 14 | 16 |
| Go mínimo do módulo | 1.25 | 1.26 |

Há 92 commits posteriores. Entre as melhorias aplicáveis estão confirmação de retry receipts após processamento, correções LID em armazenamento e grupos, isolamento da fila de eventos entre conexões, contexto correto dos frames, tamanho de frames fragmentados, correção do canal QR, retomada após handshake sem resposta, redação de credenciais de mídia em logs e backoff em rate-overlimit do app state.

A atualização não é uma troca direta de versão. Em cópia isolada da nc.3, o simples bump e `go test ./...` falharam na compilação: duas chamadas de `UpdateBlocklist` precisam do parâmetro adicional de hash, `GetBlocklist` também mudou de assinatura e `SetStatusMessage` recebe agora `types.SetStatusInput`. A nova biblioteca também inclui as migrations 15 (companion meta nonce) e 16 (WASA root secret), além de atualizações transitivas. Não foi testada uma conta real com essa atualização.

Decisão desta release: manter a dependência atual, integrar a correção pontual de avatar do PR120 e publicar nc.4 sem mudança de schema. Uma atualização completa do WhatsMeow precisa adaptar essas chamadas e homologar upgrade e retorno de sessão sintética entre os schemas, envio, recebimento, grupos e autenticação antes de levar o novo armazenamento ao servidor.

Fontes primárias: [Go Proxy @latest](https://proxy.golang.org/go.mau.fi/whatsmeow/@latest), [comparação dos 92 commits](https://github.com/tulir/whatsmeow/compare/a2381054887e16d22428430fdc4d2b027a7f29a7...54038813c47f3f351eb6fcaafab0e1b99d676850), [schema mais recente](https://github.com/tulir/whatsmeow/blob/54038813c47f3f351eb6fcaafab0e1b99d676850/store/sqlstore/upgrades/00-latest-schema.sql) e [go.mod](https://github.com/tulir/whatsmeow/blob/54038813c47f3f351eb6fcaafab0e1b99d676850/go.mod).

Evidências locais: [whatsmeow-compatibility.log](evidence/nc4/whatsmeow-compatibility.log) e [whatsmeow-latest.json](evidence/nc4/whatsmeow-latest.json). Nenhuma dependência ou banco persistente da aplicação foi alterado por essa investigação.
