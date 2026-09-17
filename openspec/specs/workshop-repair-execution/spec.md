# workshop-repair-execution Specification

> **Promoción** catálogo principal dende o change archivado `openspec/changes/archive/2026-04-22-workshop-mechanic-vehicle-lifecycle/` (2026-04-22), actualizado dende `openspec/changes/archive/2026-04-20-taller-benchmark-mvp-priorities/` (2026-04-20) — recepción “app”, listado por día UTC, enlace `repair_ids` en lectura de visita.

## Purpose

Ciclo de servicio de taller por vehículo: la **visita (service job) es la unidad de trazabilidad** — recepción, cierre y, en fases posteriores, diagnóstico, presupuesto y entrega formal. **MVP1 mínima:** recepción + cierre; OBD/ presupuesto como **stubs** con contrato explícito.

| Fase | MVP1 | Post-MVP1 |
|------|------|-----------|
| A Recepción (km, fluidos, neumáticos, notas) | **SHALL** persistir | **MAY** fotos / tercero |
| B OBD / diagnóstico | **MAY** (stub) | códigos + historial |
| C Presupuesto / aprobación | **MAY** estado stub | flujo aprobado |
| D Ejecución (enlace a trabajo) | **SHOULD** enlazar a `repairs` | SLA y desvíos |
| E Entrega re-verificada | **SHALL** persistir cierre básico | firma / comprobante |

## Requirements

### Requirement: Agregado de visita (service job)

El sistema **SHALL** modelar una **visita de taller** (service job) como agregado propio con `car_id` **SHALL** obligatorio; `appointment_id` **MAY** nulo. El ciclo de visita **MUST NOT** depender solo de filas de `repairs` sin este agregado. Las reparaciones históricas sin visita **SHALL** seguir siendo legítimas (sin backfill forzado).

#### Scenario: Alta de visita

- GIVEN un `employee` con permiso de acceso al coche
- WHEN crea una visita para ese `car_id`
- THEN existe identificador de visita y estado mínimo **open** (nome concreto en implementación)

#### Scenario: Listado con legado

- GIVEN reparaciones antiguas sin service job
- WHEN se listan reparaciones del coche
- THEN se muestran como hoy; **MUST NOT** exigir migración automática a visita

### Requirement: Recepción y cierre mínimos

La visita **SHALL** aceptar **recepción** (p. ej. km, fluidos, neumáticos, notas) y **cierre/entrega** re-verificados. El persistido **SHALL** ser **estructurado y versionable** (tablas hijas o bloques con `schema_version`); **MUST NOT** aceptar JSON suelto sin esquema versionado en contrato de API.

#### Scenario: Recepción completa y válida

- GIVEN visita abierta
- WHEN el empleado envía un checklist de recepción **válido** según validación
- THEN la recepción queda persistida con al menos qué usuario y cuándo

#### Scenario: Recepción incompleta

- GIVEN la misma visita
- WHEN faltan campos obligatorios (`odometer_km` **MUST** estar presente en el cuerpo JSON; `0` explícito es válido; oil/coolant/tires/notes siguen opcionales)
- THEN la API **MUST** rechazar (no 2xx) con error interpretable — un `odometer_km` ausente **MUST NOT** ser 2xx

#### Scenario: Cierre

- GIVEN recepción persistida
- WHEN el empleado completa el checklist de cierre/entrega válido
- THEN el cierre queda persistido y el estado de visita **MUST** reflejar cierre (p. ej. **closed**)

### Requirement: OBD y presupuesto (stub MVP1+)

OBD, estimación o aprobación **MAY** exponerse como *stub* (p. ej. 501, o recurso vacío con semántica documentada) sin romper visitas v1.

#### Scenario: Lógica aún no implementada

- GIVEN MVP1 sin lógica de OBD/estimate
- WHEN se llama al sub-recurso correspondiente
- THEN la respuesta **MUST** ser predecible; **MUST NOT** 200 con cuerpo que implique un estado negocio falso

### Requirement: Trazas de transición mínima

Apertura, cierre o cancelación de la visita **SHALL** dejar rastro (quién/cuándo) al menos en API de lectura de la visita.

#### Scenario: Lectura tras cierre

- GIVEN cierre de visita completado
- WHEN se obtiene el recurso `GET` de la visita
- THEN la carga **SHALL** incluir marcas mínimas de apertura y cierre (actor/timestamp según criterio de cierre aprobado en implementación)

### Requirement: Superficie de recepción prioritaria (flujo “app”)

El personal de taller con permiso para mutar checklists de la visita **SHALL** poder completar la **recepción** con el **mismo** contrato de validación y persistencia del requisito *Recepción y cierre mínimos* desde una **superficie** orientada a captura rápida: ruta o **vista dedicada** a recepción, o **diseño responsive** equivalente. La API **MUST NOT** exigir un canal, cabecera o formato de cuerpo distinto al ya definido para la recepción; solo cambia la experiencia de producto (navegación y prioridad de tareas en UI).

#### Scenario: Recepción válida desde la superficie prioritaria

- GIVEN un miembro de staff con permiso de mutación de visita según `mvp-role-access` y visita en estado abierto
- WHEN usa la superficie prioritaria y envía un checklist de recepción válido
- THEN la recepción queda persistida

#### Scenario: Payload inválido o incompleto

- GIVEN la misma visita
- WHEN el cuerpo es inválido o incompleto
- THEN la respuesta **MUST NOT** ser 2xx; la semántica de error **SHALL** ser coherente con *Recepción incompleta*

### Requirement: Listado de visitas del “día” operativo

El sistema **SHALL** ofrecer a personal de staff (matriz de `mvp-role-access`) un medio de **listar o filtrar** visitas (service jobs) según criterio de “día” **documentado** (p. ej. fecha/hora de apertura en UTC, o regla de calendario local fijada en el producto). La visita **SHALL** seguir siendo la unidad de trazabilidad; el listado **MUST NOT** reemplazar la facturación ni usarse como comprobante de cobro.

#### Scenario: Día sin visitas

- GIVEN criterio “día” tal que ninguna visita califica
- WHEN se pide el listado o el recurso de filtrado
- THEN el resultado **SHALL** ser vacío o equivalente explícito; **MUST NOT** 5xx

#### Scenario: Día con al menos una visita

- GIVEN al menos una visita abierta que califica bajo el criterio
- WHEN se pide el listado
- THEN el resultado **SHALL** incluir al menos un identificador de visita accesible al staff autorizado

### Requirement: Lectura: enlace visita–reparaciones

Donde el modelo de datos permita vincular `repairs` a la visita, la lectura de la visita (mismo `GET` principal, o sub-recurso **coherente y documentado** en un solo criterio de producto) **SHOULD** permitir ver si la visita tiene **cero, una o varias** reparaciones vinculadas, **MUST NOT** implicar trabajo inexistente.

#### Scenario: Visita sin reparación

- GIVEN la visita sin reparación asociada
- WHEN se lee el recurso de la visita con el enfoque de enlace
- THEN la carga **MUST NOT** sugerir reparaciones inexistentes

#### Scenario: Visita con reparación vinculada

- GIVEN una reparación persistida vinculada a esa visita
- WHEN se lee de la misma forma
- THEN la carga **SHALL** permitir identificar la vinculación (identificador o lista, según el contrato elegido)

### Requirement: Leitura de visita perceptível na UI

Depois de navegar para o **detalhe de uma visita** (identificador válido na URL da app), o staff autorizado **SHALL** ver **ou** os dados mínimos da visita (estado, recepção/cierre conforme modelo) **ou** uma **mensagem de erro accionável** (p. ex. permissão, não encontrado, falha de rede). A UI **MUST NOT** ficar indefinidamente num estado vazio sem texto ou indicador de carregamento quando a solicitação terminou.

#### Scenario: Carregamento com sucesso

- **GIVEN** visita existente e `GET` de visita devolve 2xx com corpo válido
- **WHEN** o staff abre o detalhe dessa visita na app
- **THEN** o estado da visita **SHALL** tornar-se legível (texto ou componente equivalente) após o fim do carregamento

#### Scenario: Falha ou negação

- **GIVEN** resposta não-2xx ou corpo inválido para o `GET` de visita
- **WHEN** o staff permanece no detalhe
- **THEN** a UI **SHALL** mostrar erro interpretável ou orientação para voltar; **MUST NOT** mostrar apenas área em branco como se não houvesse visita

### Requirement: Nova visita alinhada ao contexto da lista

A acção **Nova visita** desde a **lista taller** **SHALL** seguir um padrão **lista-primário**: confirmação ou passo breve na lista (modal ou painel na mesma rota) **SHOULD** preceder ou acompanhar a criação antes de depender só de navegação para detalhe; **MUST NOT** ser apenas um salto sem feedback coerente com outras áreas staff que criam recursos a partir de lista.

#### Scenario: Criação com viatura seleccionada

- **GIVEN** viatura seleccionada na lista e staff com permissão
- **WHEN** confirma **Nova visita**
- **THEN** o utilizador **SHALL** receber feedback de progresso ou resultado antes de ficar só num ecrã de detalhe sem contexto

<!-- Promoted from change `ui-homogeneity-modal-workshop-parts`, archived 2026-04-27 -->

## Nota: roles

Resumen: las obligaciones **MUST/SHALL** de mutación e a matriz pública vive en `openspec/specs/mvp-role-access/spec.md`. O cliente **MUST NOT** en mutación de visita; o persoal *staff* (employee, manager, admin) na matriz; rotas bajo `RequireWorkshopStaff` / regras de servizo.
