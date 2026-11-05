# Normalização das evidências públicas

Esta pasta contém registros históricos de testes e cópias preparadas para publicação. Alguns arquivos foram normalizados; eles não devem ser descritos como logs brutos imutáveis.

Os caminhos locais foram substituídos por `<workspace>` (checkout da Evolution GO) e `<consumer-workspace>` (checkout do sistema consumidor). Em `nc5/auth-migration-readme.md`, `<fixture-postgres-dsn>` substitui duas ocorrências do DSN com credencial descartável de teste. Os placeholders são declarações de contexto para reprodução, não valores utilizáveis sem preparação do ambiente isolado descrito no documento.

Arquivos normalizados: `nc3/edited-consumer.log`, `nc4/avatar-consumer.log`, `nc5/auth-migration-readme.md` e `nc5/qr-adv-rotation.log`. Os resultados, limites e checksums de dados sintéticos registrados foram preservados. Esta edição documental não executou novos testes nem valida conexões reais.

Os originais desses arquivos foram preservados no workspace de trabalho, separados desta cópia pública. Manifestos de releases anteriores continuam identificando os artefatos e fontes do momento em que foram produzidos; não foram recalculados para ocultar as normalizações. O tratamento do patch upstream 136 está explicado em [upstream-patches/README.md](../upstream-patches/README.md).
